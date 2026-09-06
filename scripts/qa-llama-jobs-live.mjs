// Real model operations against a disposable PiCode fixture and router only.
import assert from 'node:assert/strict';
import { mkdirSync, writeFileSync } from 'node:fs';
const {chromium}=await import(process.env.PICODE_PLAYWRIGHT_MODULE||'playwright');
const base=process.env.PICODE_QA_BASE||'http://127.0.0.1:18763';
const fleet=await fetch(base+'/api/workspaces').then(r=>r.json());
assert.ok((Array.isArray(fleet)?fleet:fleet.workspaces).some(w=>/^\/tmp\/picode-docs-fixture-/.test(w.path)),'isolated fixture required');
const result=await fetch(base+'/api/providers/llama.cpp',{method:'PUT',headers:{'Content-Type':'application/json'},body:JSON.stringify({url:'http://127.0.0.1:18081',key:''})});assert.ok(result.ok);
const browser=await chromium.launch({headless:true,executablePath:process.env.PICODE_QA_CHROME});
const out=process.env.PICODE_QA_OUT||'docs/screenshots';mkdirSync(out,{recursive:true});
const evidence={checks:[],screenshots:[],audits:[]};
try {
 const context=await browser.newContext({viewport:{width:1440,height:900}});await context.addInitScript(()=>localStorage.setItem('picode-theme','dark'));
 const page=await context.newPage();const errors=[];page.on('pageerror',e=>errors.push(e.message));let loads=0,unloads=0;
 page.on('request',r=>{if(r.method()==='POST'&&r.url().endsWith('/api/llama/load'))loads++;if(r.method()==='POST'&&r.url().endsWith('/api/llama/unload'))unloads++;});
 async function shot(page,name){await page.evaluate(()=>document.fonts.ready);await page.waitForTimeout(650);const audit=await page.evaluate(()=>window.__picodeOverlayAudit());const file='llama-live-'+name+'.png';await page.screenshot({path:out+'/'+file});assert.equal(audit.ok,true,JSON.stringify(audit));evidence.screenshots.push(file);evidence.audits.push({file,...audit});}
 await page.goto(base+'/desktop/#/llama/models');const manager=page.locator('#llama-manager');const model=manager.locator('.prov-row').filter({hasText:'Qwen3-4B-Q4_K_M'});
 await model.getByRole('button',{name:'Load',exact:true}).click();
 await manager.getByRole('link',{name:'Activity',exact:true}).click();await manager.locator('.llama-job').first().waitFor();
 await manager.getByRole('link',{name:'Server',exact:true}).click();await manager.getByRole('link',{name:'Activity',exact:true}).click();
 await page.reload();await manager.locator('.llama-job[data-state="succeeded"]').waitFor({timeout:90000});await shot(page,'desktop-completed');assert.equal(loads,1);
 await manager.getByRole('link',{name:'Models',exact:true}).click();await model.getByRole('button',{name:'Unload',exact:true}).click();await page.locator('.dlg:visible').getByRole('button',{name:'Unload',exact:true}).click();
 await manager.getByRole('link',{name:'Activity',exact:true}).click();await page.waitForFunction(()=>document.querySelectorAll('.llama-job[data-state="succeeded"]').length===2,{},{timeout:90000});assert.equal(unloads,1);
 await shot(page,'desktop-history');assert.deepEqual(errors,[]);
 const mobile=await browser.newContext({viewport:{width:390,height:844}});await mobile.addInitScript(()=>localStorage.setItem('picode-theme','dark'));const phone=await mobile.newPage();phone.on('pageerror',e=>errors.push(e.message));await phone.goto(base+'/mobile/#/llama/activity');await phone.waitForFunction(()=>document.querySelectorAll('.llama-job[data-state="succeeded"]').length===2);await shot(phone,'mobile-history');
 const jobs=await fetch(base+'/api/llama/jobs').then(r=>r.json());assert.equal(jobs.jobs.filter(j=>j.state==='succeeded').length,2);assert.deepEqual(errors,[]);
 evidence.checks=['desktop real load, navigation and reload with one POST','real unload and two durable completed jobs','mobile reads the same history','no JavaScript errors; overlay audits pass'];evidence.jobs=jobs.jobs;
 writeFileSync(out+'/llama-jobs-live-ui.json',JSON.stringify(evidence,null,2)+'\n');console.log(JSON.stringify(evidence,null,2));
 await mobile.close();await context.close();
} finally {await browser.close();}
