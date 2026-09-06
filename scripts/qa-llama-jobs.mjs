// Run against an isolated picode-docs-fixture; only llama endpoints are mocked.
// PICODE_PLAYWRIGHT_MODULE points to an externally installed Playwright module.
import assert from 'node:assert/strict';
import { mkdirSync, writeFileSync } from 'node:fs';
const { chromium } = await import(process.env.PICODE_PLAYWRIGHT_MODULE || 'playwright');
const base = process.env.PICODE_QA_BASE || 'http://127.0.0.1:18763';
const fleet = await fetch(base + '/api/workspaces').then(r => r.json());
assert.ok((Array.isArray(fleet) ? fleet : fleet.workspaces).some(w => /^\/tmp\/picode-docs-fixture-/.test(w.path)), 'isolated fixture required');
const browser = await chromium.launch({ headless: true, ...(process.env.PICODE_QA_CHROME ? { executablePath: process.env.PICODE_QA_CHROME } : {}) });
const out = process.env.PICODE_QA_OUT || 'docs/screenshots';
mkdirSync(out, { recursive: true });
const evidence = { checks: [], screenshots: [], audits: [] };
try {
for (const app of ['desktop', 'mobile']) {
 const context = await browser.newContext({viewport:app==='desktop'?{width:1440,height:900}:{width:390,height:844}});
 await context.addInitScript(() => localStorage.setItem('picode-theme','dark'));
 const page=await context.newPage();const errors=[];page.on('pageerror',e=>errors.push(e.message));
 let jobs=[], unavailable=false, cancels=0, mutations=0;
 const job={id:'qa-download',model:'Example/compact-model:Q4_K_M',operation:'download',state:'running',stage:'observing',observed:'downloading',cancelSupported:true,cancelRequested:false,endpoint:'http://127.0.0.1:18081',createdAt:'2026-09-06T12:00:00Z',message:'Waiting for the server: downloading.',progress:[{file:'model-00001.gguf',done:524288000,total:1073741824},{file:'model-00002.gguf',done:10485760,total:0}]};
 await page.route('**/api/llama',r=>r.fulfill({json:{url:job.endpoint,ok:true,models:[],connection:{code:'ready',message:'Connected'},capabilities:{build:'b10809-qa',events:true,cancelDownload:true}}}));
 await page.route('**/api/llama/jobs',r=>unavailable?r.fulfill({status:503,json:{error:'Unavailable'}}):r.fulfill({json:{jobs}}));
 await page.route('**/api/llama/jobs/*/cancel',r=>{cancels++;jobs=jobs.map(j=>({...j,cancelRequested:true,message:'Waiting for the download to stop.'}));return r.fulfill({json:{job:jobs[0]}});});
 await page.route('**/api/llama/jobs/*/reconcile',r=>{jobs=jobs.map(j=>({...j,state:'interrupted',message:'Loading was interrupted. The model is not loaded.'}));return r.fulfill({json:{job:jobs[0]}});});
 for(const op of ['load','unload','download']) await page.route('**/api/llama/'+op,r=>{mutations++;return r.fulfill({status:500,json:{error:'Unexpected mutation'}});});
 async function shot(name){await page.evaluate(()=>document.fonts.ready);await page.waitForTimeout(650);const audit=await page.evaluate(()=>window.__picodeOverlayAudit());assert.equal(audit.ok,true,JSON.stringify(audit));const file=`llama-${app}-activity-${name}.png`;await page.screenshot({path:out+'/'+file});evidence.screenshots.push(file);evidence.audits.push({file,...audit});}
 await page.goto(base+`/${app}/#/llama/activity`);
 const manager=page.locator('#llama-manager');await manager.getByText('No model operations yet.').waitFor();await shot('empty');
 jobs=[job];await manager.getByRole('button',{name:'Refresh activity'}).click();await manager.getByRole('button',{name:'Cancel download',exact:true}).waitFor();await shot('progress');
 await manager.getByRole('link',{name:'Models',exact:true}).click();await manager.getByRole('link',{name:/View activity/}).click();await manager.getByText('model-00001.gguf',{exact:true}).waitFor();
 await page.reload();await manager.getByRole('button',{name:'Cancel download',exact:true}).click();await manager.getByRole('button',{name:'Cancel requested…',exact:true}).waitFor();assert.equal(cancels,1);await shot('cancel');
 jobs=[{...job,state:'canceled',message:'The server stopped the download.'}];await page.reload();await manager.getByText('Canceled',{exact:true}).waitFor();assert.equal(cancels,1);
 jobs=[{...job,state:'unknown',operation:'load',progress:[],message:'Result unknown. Check the server for the latest status.'}];await manager.getByRole('button',{name:'Refresh activity'}).click();await manager.getByRole('button',{name:'Check result'}).waitFor();await shot('unknown');
 await manager.getByRole('button',{name:'Check result'}).click();await manager.getByText('Interrupted',{exact:true}).waitFor();
 unavailable=true;await manager.getByRole('button',{name:'Refresh activity'}).click();await manager.getByRole('alert').waitFor();await shot('error');
 unavailable=false;jobs=[{...job,cancelSupported:false}];await manager.getByRole('button',{name:'Refresh activity'}).click();await manager.getByText('Cancellation is not verified for this server build.').waitFor();
 await page.evaluate(()=>document.documentElement.dataset.theme='light');await shot('light');
 assert.equal(mutations,0);assert.deepEqual(errors,[]);evidence.checks.push(`${app}: activity empty/progress, navigation and reload without mutation replay, cancel once, unknown reconciliation, refresh failure preserves history, unsupported cancellation and light theme`);
 await context.close();
}
writeFileSync(out+'/llama-jobs-qa.json',JSON.stringify(evidence,null,2)+'\n');console.log(JSON.stringify(evidence,null,2));
} finally {await browser.close();}
