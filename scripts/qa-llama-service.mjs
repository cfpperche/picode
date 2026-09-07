// Disposable fixture only. Install the browser driver outside the repository.
import assert from "node:assert/strict";
import { mkdirSync, writeFileSync } from "node:fs";
const { chromium } = await import(process.env.PICODE_PLAYWRIGHT_MODULE || "/tmp/picode-owned-playwright/node_modules/playwright-core/index.mjs");
const base=process.env.PICODE_QA_BASE||"http://127.0.0.1:18866";
const out=process.env.PICODE_QA_OUT||"var/screenshots/llama-service";
mkdirSync(out,{recursive:true});
const workspaces=await fetch(base+"/api/workspaces").then(r=>r.json());
assert.ok((Array.isArray(workspaces)?workspaces:workspaces.workspaces).some(w=>w.path.startsWith("/tmp/picode-docs-fixture-")),"synthetic fixture required");
const browser=await chromium.launch({headless:true,executablePath:process.env.PICODE_QA_CHROME||"/usr/bin/google-chrome"});
const report={checks:[],screenshots:[],audits:[],errors:[]};
const snapshot=()=>fetch(base+"/api/llama/service").then(r=>r.json());
async function shot(page,name){
 await page.evaluate(()=>document.fonts.ready);await page.waitForTimeout(700);
 const audit=await page.evaluate(()=>window.__picodeOverlayAudit());
 const path=out+"/"+name+".png";await page.screenshot({path});
 assert.equal(audit.ok,true,JSON.stringify(audit));
 report.screenshots.push(path);report.audits.push({name,...audit});
}
try{
 if(!process.env.PICODE_QA_STATES_ONLY){
 const context=await browser.newContext({viewport:{width:1440,height:1000}});
 await context.addInitScript(()=>localStorage.setItem("picode-theme","dark"));
 const page=await context.newPage();page.on("pageerror",e=>report.errors.push(e.message));
 await page.goto(base+"/desktop/#/llama/service");await page.getByRole("heading",{name:"Local service",exact:true}).waitFor();
 await page.getByRole("button",{name:"Atlas",exact:true}).waitFor();
 let s=await snapshot();
 if(!s.created){
  await shot(page,"desktop-empty");
  await page.locator("summary").filter({hasText:"Advanced settings"}).click();
  await page.getByLabel("CPU threads",{exact:true}).fill("0");
  await page.getByRole("button",{name:"Create local service",exact:true}).click();
  await page.getByRole("alert").waitFor();await shot(page,"desktop-error");
  assert.equal((await snapshot()).created,false);
  await page.getByLabel("CPU threads",{exact:true}).fill("2");
  await page.getByRole("button",{name:"Create local service",exact:true}).click();
  await page.getByRole("button",{name:"Save settings",exact:true}).waitFor();
  await page.locator("summary").filter({hasText:"Advanced settings"}).click();
  report.checks.push("invalid settings refused before persistence; profile created through UI");
 }
 async function action(label,name){
  console.log("Checking",label);
  const old=(await snapshot()).jobs?.[0]?.id;
  await page.getByRole("button",{name:label,exact:true}).click();
  const dialog=page.locator('[role="dialog"],[role="alertdialog"]');await dialog.waitFor();
  if(name)await shot(page,name);
  const checkbox=dialog.getByRole("checkbox");
  if(await checkbox.count())await checkbox.check();
  await dialog.getByRole("button",{name:label,exact:true}).click();
  let done;const deadline=Date.now()+120000;
  while(Date.now()<deadline){
   done=await snapshot();
   if(done.jobs?.[0]?.id!==old&&!done.busy&&["succeeded","failed","interrupted"].includes(done.jobs?.[0]?.state))break;
   await page.waitForTimeout(200);
  }
  assert.equal(done.jobs[0].state,"succeeded",JSON.stringify(done.jobs[0]));
  await page.waitForTimeout(250);report.checks.push(label+" succeeded through real API/UI");return done;
 }
 s=await snapshot();
 if(!s.current)await action("Install","desktop-install-review");
 if(!(await snapshot()).running)await action("Start","desktop-start-review");
 await shot(page,"desktop-running");
 await page.getByRole("button",{name:"Use this connection",exact:true}).click();
 await page.getByLabel("Server URL",{exact:true}).waitFor();
 assert.equal(await page.getByLabel("Server URL",{exact:true}).inputValue(),(await snapshot()).url);
 await page.getByRole("link",{name:"Local service",exact:true}).click();
 await page.getByRole("combobox",{name:"Verified version",exact:true}).selectOption("b10826");
 await action((await snapshot()).current.version==="b10826"?"Reinstall":"Update","desktop-update-review");
 await action("Restore previous version","desktop-rollback-review");
 await action("Stop","desktop-interruption-review");
 await page.locator("summary").filter({hasText:"Cache"}).click();
 const eligible=page.locator(".llama-cache-row input[type=checkbox]");
 assert.ok(await eligible.count()>0);await eligible.first().check();
 await action("Clean selected files","desktop-cleanup-review");
 await shot(page,"desktop-stopped");
 const diagnostics=await fetch(base+"/api/llama/service/diagnostics").then(r=>r.json());
 assert.deepEqual(Object.keys(diagnostics).sort(),["busy","config","jobs","platform","running","version"]);
 report.checks.push("diagnostics contains only allowlisted fields");
 await context.close();
 }
 // Synthetic unreachable-platform and API-error states; no mutations mocked as success.
 for(const app of ["desktop","mobile"]){
  const ctx=await browser.newContext({viewport:app==="desktop"?{width:1440,height:1000}:{width:390,height:844}});
  await ctx.addInitScript(()=>localStorage.setItem("picode-theme","dark"));
  const p=await ctx.newPage();p.on("pageerror",e=>report.errors.push(e.message));
  await p.route("**/api/llama/service",r=>r.fulfill({json:{created:false,revision:0,supported:false,host:"Other host",versions:[],jobs:[]}}));
  await p.goto(base+"/"+app+"/#/llama/service");await p.getByRole("link",{name:"Configure external server",exact:true}).waitFor();
  await shot(p,app+"-blocked");
  await p.unroute("**/api/llama/service");
  await p.route("**/api/llama/service",r=>r.fulfill({status:503,json:{error:"Local service management is unavailable."}}));
  await p.reload();await p.getByRole("alert").waitFor();await shot(p,app+"-unavailable");
  await p.unroute("**/api/llama/service");
  await p.reload();await p.getByRole("button",{name:"Save settings",exact:true}).waitFor();
  await shot(p,app+"-ready");
  await p.getByRole("button",{name:"Start",exact:true}).click();
  await p.locator('[role="dialog"],[role="alertdialog"]').waitFor();await shot(p,app+"-confirmation");
  await p.locator('[role="dialog"],[role="alertdialog"]').getByRole("button",{name:"Cancel",exact:true}).click();
  await p.evaluate(()=>document.documentElement.dataset.theme="light");
  await shot(p,app+"-light");
  await ctx.close();
 }
 assert.deepEqual(report.errors,[]);
 writeFileSync(out+"/report.json",JSON.stringify(report,null,2)+"\n");
 console.log(JSON.stringify(report));
}finally{await browser.close();}
