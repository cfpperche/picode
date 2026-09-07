// Real installation cleanup on a fresh synthetic docs fixture; never loads a model.
import assert from 'node:assert/strict';
import {mkdirSync,writeFileSync,existsSync} from 'node:fs';
import {dirname,join,basename} from 'node:path';
const {chromium}=await import(process.env.PICODE_PLAYWRIGHT_MODULE||'/tmp/picode-owned-playwright/node_modules/playwright-core/index.mjs');
const base=process.env.PICODE_QA_BASE||'http://127.0.0.1:18871';
const out=process.env.PICODE_QA_OUT||'var/screenshots/llama-installations';
mkdirSync(out,{recursive:true});
async function api(path,body,method=body?'POST':'GET'){
 const r=await fetch(base+path,{method,headers:{'Content-Type':'application/json'},body:body?JSON.stringify(body):undefined});
 const data=await r.json();assert.ok(r.ok,JSON.stringify(data));return data;
}
const fleet=await api('/api/workspaces');assert.ok(fleet.length&&fleet.every(w=>w.path.startsWith('/tmp/picode-docs-fixture-')));
const snapshot=()=>api('/api/llama/service');
assert.equal((await snapshot()).created,false);
const report={checks:[],audits:[],errors:[]};
const browser=await chromium.launch({headless:true,executablePath:'/usr/bin/google-chrome'});
async function shot(p,name){await p.evaluate(()=>document.fonts.ready);await p.waitForTimeout(700);const audit=await p.evaluate(()=>window.__picodeOverlayAudit());await p.screenshot({path:out+'/'+name+'.png'});assert.equal(audit.ok,true,JSON.stringify(audit));report.audits.push({name,...audit});}
async function waitJob(id){for(let i=0;i<180;i++){const s=await snapshot();if(!s.busy&&s.jobs[0]?.id===id){assert.equal(s.jobs[0].state,'succeeded',JSON.stringify(s.jobs[0]));return s;}await new Promise(r=>setTimeout(r,1000));}throw Error('job timeout');}
async function action(action,version){const p=await api('/api/llama/service/preview',{action,version,revision:(await snapshot()).revision});const j=await api('/api/llama/service/execute',{token:p.token,interrupt:true});return waitJob(j.job.id);}
try {
 await api('/api/llama/service',{config:{port:18085,context:4096,threads:2,jinja:true},revision:0},'PUT');
 const desktop=await browser.newContext({viewport:{width:1440,height:1000}});await desktop.addInitScript(()=>localStorage.setItem('picode-theme','dark'));
 const p=await desktop.newPage();p.on('pageerror',e=>report.errors.push(e.message));
 await p.goto(base+'/desktop/#/llama/service');await p.getByRole('button',{name:'Save settings',exact:true}).waitFor();
 await p.locator('summary').filter({hasText:'Cache and installations'}).click();await shot(p,'desktop-empty');
 await action('install','b10809');await action('update','b10826');await action('update','b10809');
 const s=await snapshot(), root=dirname(s.modelsDir);
 const unknown=join(root,'release-unknown');mkdirSync(unknown);writeFileSync(join(unknown,'keep'),'unknown file');
 const files=(await api('/api/llama/service/cache')).files;
 const old=files.filter(f=>f.label&&f.eligible);assert.equal(old.length,1);
 for(const r of [s.current,s.previous])assert.equal(files.find(f=>f.name===basename(r.dir)).eligible,false);
 assert.equal(files.find(f=>f.name==='release-unknown').eligible,false);
 report.checks.push('three real verified installations; only superseded installation eligible; current/rollback/unknown retained');
 await p.getByRole('button',{name:'Refresh service',exact:true}).click();
 const row=p.locator('.llama-cache-row').filter({hasText:old[0].name});await row.getByRole('checkbox').check();await row.scrollIntoViewIfNeeded();await shot(p,'desktop-selection');
 await p.route('**/api/llama/service/preview',route=>route.fulfill({status:409,contentType:'application/json',body:JSON.stringify({error:'The selection changed. Refresh and review again.'})}));
 await p.getByRole('button',{name:'Clean selected items',exact:true}).click();await p.getByText('The selection changed. Refresh and review again.',{exact:true}).waitFor();await p.getByText('The selection changed. Refresh and review again.',{exact:true}).scrollIntoViewIfNeeded();await shot(p,'desktop-error');assert.ok(existsSync(join(root,old[0].name)));await p.unroute('**/api/llama/service/preview');
 await p.getByRole('button',{name:'Clean selected items',exact:true}).click();const dlg=p.locator('[role="dialog"],[role="alertdialog"]');await dlg.waitFor();await shot(p,'desktop-review');
 assert.ok((await dlg.innerText()).includes(old[0].name));await dlg.getByRole('button',{name:'Cancel',exact:true}).click();assert.ok(existsSync(join(root,old[0].name)));
 await desktop.close();
 const mobile=await browser.newContext({viewport:{width:390,height:844}});await mobile.addInitScript(()=>localStorage.setItem('picode-theme','dark'));
 const phone=await mobile.newPage();phone.on('pageerror',e=>report.errors.push(e.message));await phone.goto(base+'/mobile/#/llama/service');
 await phone.getByRole('button',{name:'Save settings',exact:true}).waitFor();await phone.locator('summary').filter({hasText:'Cache and installations'}).click();
 const mobileRow=phone.locator('.llama-cache-row').filter({hasText:old[0].name});await mobileRow.getByRole('checkbox').check();await mobileRow.scrollIntoViewIfNeeded();await shot(phone,'mobile-selection');
 await phone.getByRole('button',{name:'Clean selected items',exact:true}).click();const md=phone.locator('[role="dialog"],[role="alertdialog"]');await md.waitFor();await shot(phone,'mobile-review');
 const response=phone.waitForResponse(r=>r.url().endsWith('/api/llama/service/execute')&&r.request().method()==='POST');
 await md.getByRole('button',{name:'Clean selected items',exact:true}).click();const accepted=await (await response).json();await waitJob(accepted.job.id);
 assert.equal(existsSync(join(root,old[0].name)),false);assert.ok(existsSync(s.current.dir)&&existsSync(s.previous.dir)&&existsSync(join(unknown,'keep')));
 await phone.getByRole('button',{name:'Refresh service',exact:true}).click();await phone.locator('.llama-cache-row').filter({hasText:'release-unknown'}).scrollIntoViewIfNeeded();await shot(phone,'mobile-retained');assert.equal(await phone.getByRole('button',{name:'Clean selected items',exact:true}).isDisabled(),true);
 await phone.evaluate(()=>document.documentElement.dataset.theme='light');await shot(phone,'mobile-light');
 report.checks.push('desktop cancel retains installation; mobile confirmation removes exact older directory; protected directories survive');
 await mobile.close();assert.deepEqual(report.errors,[]);
 report.pass=true;writeFileSync(out+'/report.json',JSON.stringify(report,null,2)+'\n');console.log(JSON.stringify(report));
} finally {await browser.close();}
