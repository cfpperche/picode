#!/usr/bin/env node
// Route and mutation boundaries against an owned synthetic fixture only.
import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { mkdirSync, writeFileSync, readFileSync } from "node:fs";
import { resolve, dirname, join } from "node:path";
const base = new URL(process.argv[2]);
assert.ok(base.protocol === "http:" && ["localhost", "127.0.0.1"].includes(base.hostname));
const out = resolve(process.argv[3] || "var/screenshots/cli-packages/browser");
mkdirSync(out, { recursive: true });
const api = async (path, method = "GET", body) => {
  const res = await fetch(new URL(path, base), { method, ...(body ? { headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) } : {}) });
  assert.ok(res.ok, `${method} ${path}: ${res.status} ${res.ok ? "" : await res.text()}`);
  return res.status === 204 ? null : res.json();
};
const workspaces = await api("/api/workspaces");
const workspace = workspaces.find(w => w.name === "picode" && /\/picode-docs-fixture-[^/]+\/work\/picode$/.test(w.path));
assert.ok(workspace, "Refuse non-synthetic fixture");
const agent = workspace.agents.find(a => a.name === "Atlas");
assert.equal(agent.mode, "stopped");
const id = agent.id, ws = workspace.id;
const fixture = dirname(dirname(workspace.path));
const source = join(fixture, "packages/pi-roles");
mkdirSync(source, { recursive: true });
writeFileSync(join(source, "package.json"), JSON.stringify({ name: "pi-roles", version: "1.0.0", pi: { extensions: [] } }));
const settingsFile = join(fixture, "home/.pi/agent/settings.json");
writeFileSync(settingsFile, JSON.stringify({ packages: [] }));
let session = "cli-packages-" + process.pid;
const browser = (...args) => execFileSync("agent-browser", ["--session", session, ...args], { encoding: "utf8", maxBuffer: 4 * 1024 * 1024 }).trim();
const ev = code => JSON.parse(browser("eval", code));
const wait = code => browser("wait", "--fn", code);
const nav = hash => ev(`location.hash=${JSON.stringify(hash)}`);
const root = "#/clis/packages/pi";
const context = root + "?workspaceId=" + ws + "&agentId=" + id;
const input = '[aria-label="Package source"]';
const ready = () => wait(`!!document.querySelector(${JSON.stringify(input)}) && !document.querySelector(${JSON.stringify(input)}).matches(':disabled')`);
let visit = 0;
const open = (app, hash, width = app === "mobile" ? 390 : 1365, theme = "dark") => {
  browser("open", new URL(`/${app}/?theme=${theme}&qa=${process.pid}-${++visit}${hash}`, base).href);
  browser("set", "viewport", String(width), String(app === "mobile" ? 844 : 1000));
  assert.deepEqual(ev("[innerWidth,innerHeight]"), [width, app === "mobile" ? 844 : 1000]);
};
async function capture(name, audit = true) {
  await new Promise(done => setTimeout(done, 600));
  if (audit) {
    const report = ev("window.__picodeOverlayAudit()");
    writeFileSync(join(out, name + "-audit.json"), JSON.stringify(report, null, 2));
    assert.equal(report.ok, true, name + " geometry");
  }
  browser("screenshot", join(out, name + ".png"));
}
const results = [];
const fault = code => ev(`window.qaFetch ||= window.fetch; ${code}`);
try {
  for (const app of (process.env.QA_APPS?.split(",") || ["desktop", "mobile"])) {
    try { browser("close"); } catch {}
    session = "cli-packages-" + process.pid + "-" + app;
    writeFileSync(settingsFile, JSON.stringify({ packages: [] }));
    open(app, root); ready();
    assert.equal(ev('document.querySelectorAll(".pkg-scope [role=radio]").length'), 1);
    await capture(app + "-empty");
    browser("click", '[aria-label="Packages CLI: Pi"]');
    await capture(app + "-cli-picker"); browser("press", "Escape");
    results.push(app + ": empty machine view and contained CLI selector");
    // Seed native package state; no package manager process or download is run.
    writeFileSync(settingsFile, JSON.stringify({ packages: [source] }));
    open(app, context); ready();
    assert.equal(ev('document.querySelectorAll(".pkg-scope [role=radio]").length'), 3);
    browser("find", "role", "radio", "click", "--name", "This agent", "--exact");
    wait('location.hash.includes("scope=agent")'); ready();
    browser("fill", input, source);
    browser("click", ".pkg-by-source button");
    wait('!document.querySelector(".pkg-job")'); ready();
    assert.ok((await api('/api/packages?workspace=' + ws + '&agent=' + id)).packages.some(p => p.scope === 'agent' && p.source === source));
    browser("reload"); ready();
    assert.equal(ev('document.querySelector(".pkg-scope [aria-checked=true]").textContent'), "This agent");
    await capture(app + "-context");
    results.push(app + ": real agent package override, scoped URL and reload");
    open(app, context); ready();
    fault(`window.qaWrites=[]; window.fetch=(url,opts)=>{
      if(opts?.method && opts.method!=="GET")window.qaWrites.push({url:String(url),method:opts.method,body:opts.body});
      if(String(url).startsWith("/api/packages")&&opts?.method && opts.method!=="GET")return Promise.resolve(new Response(JSON.stringify({error:"Package operation failed"}),{status:503,headers:{"Content-Type":"application/json"}}));
      return window.qaFetch(url,opts);
    }`);
    for (const [label, scope] of [["This machine", "user"], [workspace.name, "project"]]) {
      browser("find", "role", "radio", "click", "--name", label, "--exact"); ready();
      browser("fill", input, "npm:fixture-only"); browser("click", ".pkg-by-source button");
      wait('!!document.querySelector(".pkg-job-err")');
      const req = ev('window.qaWrites.at(-1)');
      assert.equal(JSON.parse(req.body).scope, scope); assert.equal(JSON.parse(req.body).workspaceId, ws); assert.equal(JSON.parse(req.body).agentId, id);
      await capture(app + "-" + scope + "-install-error");
      wait("!document.querySelector('.dlg-overlay')"); browser("click", ".pkg-job button"); ready();
    }
    // Remove confirmation and failure preserve the installed list.
    ev('document.querySelector(".pkg-card .pkg-card-foot button:last-child").click()');
    browser("wait", '[role="alertdialog"], [role="dialog"]');
    await capture(app + "-remove-confirm");
    browser("find", "role", "button", "click", "--name", "Remove", "--exact");
    wait('!!document.querySelector(".pkg-job-err")');
    assert.equal(ev('window.qaWrites.at(-1).method'), "DELETE");
    await new Promise(done => setTimeout(done, 800));
    await capture(app + "-remove-error");
    wait("!document.querySelector('.dlg-overlay')"); browser("click", ".pkg-job button"); ready();
    assert.ok(ev('document.querySelectorAll(".pkg-card").length') > 0);
    ev('window.fetch=window.qaFetch');
    results.push(app + ": machine/workspace mutation payloads, errors and removal confirmation");
    // List read errors remain errors and can be retried without a fabricated empty view.
    fault(`window.qaListFail=true;window.fetch=(url,opts)=>{
      if(String(url).split("?")[0]==="/api/packages"&&(!opts?.method||opts.method==="GET")&&window.qaListFail)return Promise.resolve(new Response(JSON.stringify({error:"Package list unavailable"}),{status:503,headers:{"Content-Type":"application/json"}}));
      return window.qaFetch(url,opts);
    }`);
    nav(root); wait('!!document.querySelector("#packages-view [role=alert]")');
    await capture(app + "-list-error");
    ev('window.qaListFail=false'); browser('click','#packages-view [role=alert] button'); ready();
    ev('window.fetch=window.qaFetch'); nav(context); ready();
    results.push(app + ": list error and retry preserve truthful state");
    // Update is scoped; pending and failed responses cannot report success.
    fault(`window.qaUpdateMode="hold";window.qaUpdates=true;window.fetch=(url,opts)=>{
      if(String(url).startsWith("/api/packages/updates"))return Promise.resolve(new Response(JSON.stringify({updates:window.qaUpdates?[{source:${JSON.stringify(source)},scope:"user",current:"1.0.0",latest:"1.0.1"}]:[]}),{headers:{"Content-Type":"application/json"}}));
      if(String(url)==="/api/packages/update"&&opts?.method==="POST"){
        window.qaUpdateBody=JSON.parse(opts.body);
        if(window.qaUpdateMode==="hold")return new Promise(resolve=>{window.qaRelease=()=>resolve(new Response(JSON.stringify({error:"Update failed"}),{status:503,headers:{"Content-Type":"application/json"}}))});
        window.qaUpdates=false;return window.qaFetch(${JSON.stringify('/api/packages?workspace='+ws+'&agent='+id)});
      }
      return window.qaFetch(url,opts);
    }`);
    nav(root);ready();nav(context);ready();
    browser('find','role','button','click','--name','Update','--exact');
    wait('typeof window.qaRelease==="function"');
    assert.equal(ev('window.qaUpdateBody.scope'),'user');
    await capture(app+'-update-pending');
    ev('window.qaRelease();true');wait('!!document.querySelector(".pkg-job-err")');
    wait("!document.querySelector('.dlg-overlay')"); browser('click','.pkg-job button');ready();ev('window.qaUpdateMode="success"');
    browser('find','role','button','click','--name','Update','--exact');
    wait('!document.querySelector(".pkg-job")');ready();
    wait("!document.querySelector('.cli-tabs span[aria-label]')");
    ev('window.fetch=window.qaFetch');
    results.push(app + ": update pending/failure/success and update badge use authoritative responses");
    // Retain an unfinished source across a failed context refresh and retry.
    browser("fill", input, "keep-" + app);
    fault(`window.qaRead="fail";window.fetch=(url,opts)=>{
      if(String(url)==="/api/workspaces"&&(!opts?.method||opts.method==="GET")&&window.qaRead==="fail")return Promise.resolve(new Response(JSON.stringify({error:"Temporary package context failure"}),{status:503,headers:{"Content-Type":"application/json"}}));
      return window.qaFetch(url,opts);
    }`);
    await api('/api/agents/' + id, 'PATCH', { thinking: app === "desktop" ? "low" : "high" });
    wait('!!document.querySelector("#cli-packages-view [role=alert]")');
    assert.equal(ev(`document.querySelector(${JSON.stringify(input)}).value`), "keep-" + app);
    assert.equal(ev(`document.querySelector(${JSON.stringify(input)}).matches(':disabled')`), true);
    await capture(app + "-context-error");
    browser("click", "#cli-packages-view [role=alert] button");
    wait('!document.querySelector("#cli-packages-view [role=alert] button").disabled');
    assert.equal(ev(`document.querySelector(${JSON.stringify(input)}).matches(':disabled')`), true);
    ev('window.qaRead="pass"'); browser("click", "#cli-packages-view [role=alert] button"); ready();
    assert.equal(ev(`document.querySelector(${JSON.stringify(input)}).value`), "keep-" + app);
    ev('window.fetch=window.qaFetch');
    results.push(app + ": refresh failure/retry keeps source draft and blocks writes");
    fault(`window.fetch=async(url,opts)=>{
      const response=await window.qaFetch(url,opts);
      if(String(url)==="/api/workspaces"&&response.ok){const rows=await response.json();return new Response(JSON.stringify(rows.map(w=>w.id===${JSON.stringify(ws)}?{...w,path:w.path+"/moved"}:w)),{headers:{"Content-Type":"application/json"}})}
      return response;
    }`);
    await api('/api/agents/'+id,'PATCH',{thinking:app==='desktop'?'medium':'low'});
    wait('!!document.querySelector("#cli-packages-view [role=alert]")');
    assert.equal(ev(`document.querySelector(${JSON.stringify(input)}).value`),'keep-'+app);
    assert.equal(ev(`document.querySelector(${JSON.stringify(input)}).matches(':disabled')`),true);
    browser('find','role','button','click','--name','Reload page','--exact');
    browser('wait','[role="alertdialog"], [role="dialog"]');
    await capture(app+'-location-changed');
    browser('find','role','button','click','--name','Cancel','--exact');
    wait("!document.querySelector('.dlg-overlay')");ev('window.fetch=window.qaFetch');
    await api('/api/agents/'+id,'PATCH',{thinking:app==='desktop'?'low':'medium'});ready();
    assert.equal(ev(`document.querySelector(${JSON.stringify(input)}).value`),'keep-'+app);
    results.push(app+': changed file target blocks old drafts and requires confirmed reload');
    for (const hash of [root.replace('/pi','/codex'), root+'?agentId=missing', root+'?workspaceId='+workspaces.find(w=>w.id!==ws).id+'&agentId='+id, root+'/config/unknown']) {
      nav(hash); wait('!!document.querySelector("#cli-packages-view .cli-notice")');
      assert.equal(ev(`!!document.querySelector(${JSON.stringify(input)})`), false);
    }
    await capture(app + "-blocked");
    results.push(app + ": unsupported CLI/adapter and missing/mismatched targets block writes");
    const freeAgent = await api('/api/agents','POST',{name:'Packages QA free'});
    nav(root+'?agentId='+freeAgent.id+'&scope=agent');ready();
    assert.equal(ev('document.querySelectorAll(".pkg-scope [role=radio]").length'),2);
    await api('/api/agents/'+freeAgent.id,'DELETE');
    wait('!!document.querySelector("#cli-packages-view [role=alert]")');
    assert.equal(ev(`!!document.querySelector(${JSON.stringify(input)})`),false);
    results.push(app + ": free-agent scope and deletion never fall back to another target");
    open(app, (app === "mobile" ? "#/more/packages" : "#/packages") + '?workspaceId='+ws+'&agentId='+id); ready();
    assert.equal(ev('location.hash'), context);
    open(app, '#/packages/config/pi-roles?workspaceId='+ws+'&agentId='+id);
    wait('location.hash.startsWith("#/clis/packages/pi/config/pi-roles")');
    if (app === "mobile") {
      wait("!![...document.querySelectorAll('#cli-packages-view button')].find(b => b.textContent === 'Open desktop layout') && !document.querySelector('.cli-loading')");
      await new Promise(done => setTimeout(done, 700));
      wait("!![...document.querySelectorAll('#cli-packages-view button')].find(b => b.textContent === 'Open desktop layout')");
      await capture('mobile-config-boundary');
      const before = ev('location.hash'); browser("find", "role", "button", "click", "--name", "Open desktop layout", "--exact");
      wait('location.pathname.startsWith("/desktop")'); assert.equal(ev('location.hash'), before);
    } else browser("wait", '#packages-config');
    results.push(app + ": legacy list/config links retain explicit target; mobile desktop action preserves URL");
  }
  try { browser("close"); } catch {}
  session = "cli-packages-" + process.pid + "-roles";
  // Roles use the unchanged native files and endpoints.
  await api('/api/packages/config','PUT',{package:'pi-roles',scope:'workspace',workspaceId:ws,config:{builtin:{default:{model:'anthropic/claude-sonnet-4',thinking:'low'}},custom:[]}});
  await api('/api/packages/config','PUT',{package:'pi-roles',scope:'agent',workspaceId:ws,agentId:id,config:{builtin:{default:{model:'anthropic/claude-sonnet-4',thinking:'low'}},custom:[]}});
  open('desktop', root+'/config/pi-roles?workspaceId='+ws+'&agentId='+id); browser('wait','[aria-label="default thinking level"]');
  browser('select','[aria-label="default thinking level"]','high');
  await capture('desktop-roles-draft');
  browser('find','role','button','click','--name','Save','--exact');
  wait('!!document.querySelector(".pkg-notice.ok")');
  assert.equal((await api('/api/packages/config?package=pi-roles&workspace='+ws+'&agent='+id)).workspace.config.builtin.default.thinking,'high');
  browser('select','[aria-label="default thinking level"]','medium');
  browser('find','role','radio','click','--name','Atlas — overrides','--exact');
  wait('location.hash.includes("scope=agent")');
  assert.equal(ev(`document.querySelector('[aria-label="default thinking level"]').value`),'low');
  browser('select','[aria-label="default thinking level"]','high');
  browser('find','role','button','click','--name','Save','--exact');wait('!!document.querySelector(".pkg-notice.ok")');
  assert.equal((await api('/api/packages/config?package=pi-roles&workspace='+ws+'&agent='+id)).agent.config.builtin.default.thinking,'high');
  await capture('desktop-roles-agent');
  browser('find','role','radio','click','--name','Workspace — shared','--exact');
  wait('location.hash.includes("scope=project")');
  assert.equal(ev(`document.querySelector('[aria-label="default thinking level"]').value`),'medium');
  browser('find','role','button','click','--name','Discard','--exact');
  assert.equal(ev(`document.querySelector('[aria-label="default thinking level"]').value`),'high');
  browser('find','role','radio','click','--name','Atlas — overrides','--exact');
  wait('location.hash.includes("scope=agent")');
  browser('find','role','button','click','--name','Clear file…','--exact');
  browser('wait','[role="alertdialog"], [role="dialog"]');await capture('desktop-roles-clear');
  browser('find','role','button','click','--name','Delete file','--exact');
  wait('!!document.querySelector(".pkg-notice.ok")');
  const afterClear=await api('/api/packages/config?package=pi-roles&workspace='+ws+'&agent='+id);
  assert.equal(afterClear.agent.exists,false);assert.equal(afterClear.workspace.config.builtin.default.thinking,'high');
  browser('find','role','radio','click','--name','Workspace — shared','--exact');
  wait('location.hash.includes("scope=project")');
  browser('select','[aria-label="default thinking level"]','medium');
  assert.ok(afterClear.workspace.path.startsWith(fixture+'/work/'));
  writeFileSync(afterClear.workspace.path,'{broken');
  browser('find','role','button','click','--name','Save','--exact');
  wait("[...document.querySelectorAll('button')].some(b=>b.textContent==='Replace anyway')");
  await capture('desktop-roles-conflict');
  browser('find','role','button','click','--name','Replace anyway','--exact');
  wait('!!document.querySelector(".pkg-notice.ok")');
  assert.equal((await api('/api/packages/config?package=pi-roles&workspace='+ws+'&agent='+id)).workspace.config.builtin.default.thinking,'medium');
  writeFileSync(afterClear.workspace.path,'{broken');browser('reload');
  browser('wait','.pkg-notice.err');
  browser('find','role','button','click','--name','Replace file…','--exact');
  wait('!!document.querySelector(".pkg-notice.ok")');
  assert.equal((await api('/api/packages/config?package=pi-roles&workspace='+ws+'&agent='+id)).workspace.invalid,undefined);
  results.push('desktop: independent layer drafts, scoped clear and real malformed-file conflicts/replacement');
  browser('click','.pkg-back'); ready(); assert.ok(ev('location.hash').includes('agentId='+id));
  results.push('desktop: real roles-file save and layer/config/back navigation preserve target');
  // Packages must remain usable when terminal inventory is unavailable.
  open('desktop',context); ready();
  fault(`window.qaTerms=0;window.fetch=(url,opts)=>{if(["/api/terminals","/api/clis/profiles","/api/cli-jobs"].includes(String(url))){window.qaTerms++;return Promise.reject(new Error("terminal inventory unavailable"));}return window.qaFetch(url,opts);}`);
  nav(root); ready(); nav(context); ready(); assert.equal(ev('window.qaTerms'),0);
  results.push('packages context is independent of terminal inventory and installation jobs');
  for (const width of [1365,560]) {
    open('desktop',context,width,width===560?'light':'dark');ready();
    browser('click','[aria-label="Packages CLI: Pi"]'); await capture('desktop-'+width+'-selector'); browser('press','Escape');
    if(width===560){ nav(root+'/config/pi-roles?workspaceId='+ws+'&agentId='+id);browser('wait','#packages-config');await capture('desktop-narrow-config'); }
  }
  results.push('wide/narrow desktop overlays remain contained in dark/light themes');
  writeFileSync(join(out,'results.json'),JSON.stringify({ok:true,results,realPackageManagerProcesses:false,realModelTurns:false},null,2));
  console.log(JSON.stringify({ok:true,results},null,2));
} catch(error) {
  try { writeFileSync(join(out,'failure.json'),JSON.stringify({error:String(error),snapshot:browser('snapshot','-i')},null,2)); browser('screenshot',join(out,'failure.png')); } catch {}
  throw error;
} finally { try { browser('close'); } catch {} }
