#!/usr/bin/env node
// Exercise native provider navigation against an owned synthetic fixture.
// All credential, OAuth, quota and verification requests are intercepted.
import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { mkdirSync, writeFileSync } from "node:fs";
import { resolve, join } from "node:path";

const base = new URL(process.argv[2]);
assert.ok(base.protocol === "http:" && ["localhost", "127.0.0.1"].includes(base.hostname));
const out = resolve(process.argv[3] || "var/screenshots/cli-providers/browser");
mkdirSync(out, { recursive: true });
const workspaces = await (await fetch(new URL("/api/workspaces", base))).json();
const workspace = workspaces.find(w => w.name === "picode" && /\/picode-docs-fixture-[^/]+\/work\/picode$/.test(w.path));
assert.ok(workspace, "Refuse non-synthetic fixture");
const agent = workspace.agents.find(a => a.name === "Atlas");
assert.equal(agent.mode, "stopped");

const model = { id: "fixture-model", thinking: false, images: false };
const account = (id, label, extra = {}) => ({ id, label, type: "oauth", quotaKind: "oauth", active: false, ...extra });
const catalog = { thinking: [], providers: [
  { id: "anthropic", login: "both", signedIn: true, source: "vault", quotaKind: "oauth", models: [model], agents: 2,
    accounts: [account("work", "Work", { active: true }), account("personal", "Personal"), account("paused", "Paused", { paused: true })] },
  { id: "groq", login: "api_key", signedIn: true, source: "environment", envVar: "GROQ_API_KEY", models: [model] },
  { id: "openai-codex", login: "oauth", signedIn: true, models: [model], accounts: [account("solo", "Solo", { active: true })] },
  { id: "openrouter", login: "api_key", signedIn: true, models: [model], accounts: [account("router", "Router", { active: true, type: "api_key" })] },
  { id: "xai", login: "both", signedIn: false, models: [model] },
  { id: "llama.cpp", login: "api_key", signedIn: false, models: [model] },
] };
const empty = { thinking: [], providers: catalog.providers.map(p => ({ ...p, signedIn: false, accounts: [] })) };
let session = "cli-providers-" + process.pid;
const ab = (...args) => execFileSync("agent-browser", ["--session", session, ...args], { encoding: "utf8", maxBuffer: 4 * 1024 * 1024 }).trim();
const ev = code => JSON.parse(ab("eval", code));
const wait = code => ab("wait", "--fn", code);
const nav = hash => ev(`location.hash=${JSON.stringify(hash)}`);
const list = "#/clis/providers/pi";
const ready = () => wait('!!document.querySelector("#providers-view .prov-bar")');
const pause = ms => new Promise(done => setTimeout(done, ms));
const results = [];
async function capture(name) {
  await pause(550);
  const audit = ev("window.__picodeOverlayAudit()");
  writeFileSync(join(out, name + "-audit.json"), JSON.stringify(audit, null, 2));
  ab("screenshot", join(out, name + ".png"));
  assert.equal(audit.ok, true, name + " overlay/control audit");
  assert.ok(ev('document.documentElement.scrollWidth <= innerWidth + 1'), name + " page overflow");
}
const button = name => ab("find", "role", "button", "click", "--name", name, "--exact");
const menu = label => button("More actions for " + label);
async function chooseXai() {
  wait('!!document.querySelector("[role=dialog]")');
  await pause(550); // Wait for the mobile sheet entrance before a pointer click.
  ab("click", '[cmdk-item][data-value="xai both"]');
  wait('document.querySelector("[role=dialog]")?.textContent.includes("Choose how to sign in.")');
}
function intercept() {
  ev(`window.qaFetch = window.fetch; window.qa={catalog:${JSON.stringify(catalog)},calls:[],catalogFail:false,writeFail:false,oauthMode:'success'};
    window.open=(url)=>{window.qa.opened=url;return null};
    window.fetch=async(url,options={})=>{
      const path=new URL(String(url),location.origin).pathname,method=options.method||'GET';
      const q=window.qa;const reply=(body,status=200)=>new Response(JSON.stringify(body),{status,headers:{'Content-Type':'application/json'}});
      if(path==='/api/catalog') {q.calls.push({path,method});if(q.catalogWait)await new Promise(resolve=>{q.catalogResolve=resolve});return q.catalogFail?reply({error:'Catalog unavailable'},503):reply(q.catalog)}
      if(path.startsWith('/api/providers')||path.startsWith('/api/oauth')) {
        q.calls.push({path,method,body:options.body?JSON.parse(options.body):null});
        if(path==='/api/providers/usage')return reply({entries:[]});
        if(path==='/api/oauth/start')return reply({url:location.origin+'/oauth-fixture'});
        if(path==='/api/oauth/status'){
          if(q.oauthMode==='delay')await new Promise(resolve=>{q.oauthResolve=resolve});
          return reply({done:true,pending:false,...(q.oauthMode==='error'?{error:'Fixture login failed'}:{})});
        }
        if(path.endsWith('/usage'))return q.usageFail?reply({error:'Usage unavailable in fixture'},503):reply({status:'ok',windows:[{id:'five-hour',label:'5 hour window',usedPercent:42}],resets:[]});
        if(method!=='GET'&&q.writeFail)return reply({error:'Fixture write failed'},503);
        if(path.endsWith('/verify'))return reply({ok:true,status:'ready'});
        return reply({ok:true});
      }
      if(path.startsWith('/api/clis')||path==='/api/cli-jobs')return reply({error:'Launch inventory unavailable in fixture'},503);
      return window.qaFetch(url,options);
    }`);
}
const lastWrite = () => ev('qa.calls.findLast(c=>c.method!=="GET")');
try {
  for (const app of (process.env.QA_APPS?.split(",") || ["desktop", "mobile"])) {
    try { ab("close"); } catch {}
    session = "cli-providers-" + process.pid + "-" + app;
    ab("open", new URL(`/${app}/?theme=dark#/more`, base).href);
    ab("set", "viewport", app === "mobile" ? "390" : "1365", app === "mobile" ? "844" : "1000");
    wait('!!document.querySelector("#m-app, #app")');
    await pause(1000); intercept();
    for (const hash of ["#/providers", "#/more/providers", "#/clis/providers"]) {
      nav(hash); wait(`location.hash===${JSON.stringify(list)}`); ready();
    }
    await capture(app + "-roster");
    button("Providers CLI: Pi"); await capture(app + "-cli-picker"); ab("press", "Escape");
    for (const hash of ["#/providers/new", "#/more/providers/new", list + "/new"]) {
      nav(hash); wait(`location.hash===${JSON.stringify(list + "/new")} && !!document.querySelector('[role=dialog]')`);
      ab("press", "Escape"); wait(`location.hash===${JSON.stringify(list)} && !document.querySelector('[role=dialog]')`);
    }
    nav(list + "/new"); wait('!!document.querySelector("[role=dialog]")'); nav(list); wait('!document.querySelector("[role=dialog]")');
    results.push(app + ": legacy/list/new redirects, add close and CLI picker");
    for (const [name, hash] of [["unsupported", "#/clis/providers/codex"], ["malformed", "#/clis/providers/%ZZ"], ["invalid", list + "/extra"], ["scoped", list + "?agentId=" + agent.id]]) {
      ev("qa.calls=[]"); nav(hash);
      wait('!!document.querySelector("#cli-providers-view .cli-notice")');
      assert.equal(ev('!!document.querySelector("#providers-view")'), false);
      assert.equal(ev('qa.calls.some(c=>c.path.startsWith("/api/providers"))'), false);
      await capture(app + "-" + name);
    }
    nav("#/providers/llama"); wait('!!document.querySelector(".llama-nav")');
    results.push(app + ": unsupported/invalid/scoped blocks and llama alias");
    nav("#/more"); ev("qa.catalogFail=true;qa.catalogWait=true"); nav(list);
    wait(`!!document.querySelector('[aria-label="Loading providers"]')`); await capture(app + "-loading");
    ev("qa.catalogWait=false;qa.catalogResolve()"); wait('!!document.querySelector("#cli-providers-view [role=alert]")');
    assert.equal(ev('!!document.querySelector("#providers-view")'), false); await capture(app + "-catalog-error");
    ev("qa.catalogFail=false"); button("Try again"); ready();
    button("Add provider"); await chooseXai(); button("Sign in with an API key");
    ab("fill", 'input[type="password"]', "fixture-key-not-a-secret");
    ev("qa.catalogFail=true;window.dispatchEvent(new Event('focus'))");
    wait('!!document.querySelector("#cli-providers-view [role=alert]")');
    assert.equal(ev('document.querySelector("input[type=password]").value'), "fixture-key-not-a-secret");
    await capture(app + "-retained-draft");
    ev("qa.writeFail=true"); button("Save"); wait('!!document.querySelector(".toast-err, .toast-error, [role=alert]")'); await pause(400);
    assert.equal(lastWrite().path, "/api/providers/xai"); assert.equal(lastWrite().method, "PUT");
    assert.deepEqual(lastWrite().body, { key: "fixture-key-not-a-secret" });
    assert.equal(ev('document.querySelector("input[type=password]").value'), "fixture-key-not-a-secret");
    ev("qa.writeFail=false;qa.catalogFail=false"); button("Save"); wait('!document.querySelector("[role=dialog]")'); ready();
    results.push(app + ": loading, catalog failure/retry, retained draft and unchanged key-save endpoint");
    // Empty results remain a truthful state, independent of launch inventory.
    ev(`qa.catalog=${JSON.stringify(empty)};window.dispatchEvent(new Event('focus'))`);
    wait('document.querySelector("#providers-view")?.textContent.includes("No providers connected.")'); await capture(app + "-empty");
    ev(`qa.catalog=${JSON.stringify(catalog)};window.dispatchEvent(new Event('focus'))`); wait('document.querySelectorAll(".prov-group").length===4');
    ab("fill", '[aria-label="Search providers and accounts"]', "no-fixture-match"); await capture(app + "-search-empty"); button("Clear search");
    menu("Work"); await capture(app + "-account-menu"); ab("find", "role", "menuitem", "click", "--name", "Verify with pi", "--exact");
    wait('document.querySelector("#providers-view")?.textContent.includes("pi can use this")'); assert.equal(lastWrite().path, "/api/providers/anthropic/verify");
    menu("Work"); ab("find", "role", "menuitem", "click", "--name", "Pause", "--exact"); await pause(400);
    assert.equal(lastWrite().path, "/api/providers/anthropic/accounts/work/pause"); assert.deepEqual(lastWrite().body, { paused: true });
    menu("Paused"); ab("find", "role", "menuitem", "click", "--name", "Resume", "--exact"); await pause(400);
    assert.deepEqual(lastWrite().body, { paused: false });
    button("Use"); await pause(400); assert.equal(lastWrite().path, "/api/providers/anthropic/accounts/personal/activate");
    menu("Solo"); assert.ok(!ev('document.querySelector("[role=menu]").textContent').includes("Pause")); ab("press", "Escape");
    menu("GROQ_API_KEY"); assert.ok(!ev('document.querySelector("[role=menu]").textContent').includes("Sign out")); await capture(app + "-environment"); ab("press", "Escape");
    menu("Personal"); ab("find", "role", "menuitem", "click", "--name", "Sign out", "--exact");
    wait('!!document.querySelector("[role=alertdialog], [role=dialog]")'); await capture(app + "-signout-confirm"); button("Sign out"); wait('!document.querySelector(".dlg-overlay")');
    assert.equal(lastWrite().path, "/api/providers/anthropic/accounts/personal"); assert.equal(lastWrite().method, "DELETE");
    ab("click", ".prov-group:first-child .prov-acc:first-child .prov-acc-top > button.btn:not(.prov-more)"); wait('!!document.querySelector("[role=dialog]")'); await capture(app + "-usage"); ev("qa.usageFail=true"); button("Refresh"); wait('document.querySelector(".usage-empty")?.textContent.includes("Couldn")'); await capture(app + "-usage-error"); ab("press", "Escape");
    results.push(app + ": verify, pause/resume/use/signout, environment/last-account restrictions and usage");
    for (const mode of ["error", "success", "delay"]) {
      ev(`qa.oauthMode=${JSON.stringify(mode)};qa.calls=[]`); nav(list + "/new");
      await chooseXai(); button("Sign in with an account"); button("Continue in browser");
      wait('qa.calls.some(c=>c.path==="/api/oauth/start")');
      const request = ev('qa.calls.find(c=>c.path==="/api/oauth/start")');
      assert.equal(new URL(request.body.returnTo).pathname, "/" + app + "/"); assert.equal(new URL(request.body.returnTo).hash, list);
      if (mode === "error") { wait('document.querySelector("[role=dialog] .form-error")?.textContent.includes("Fixture login failed")'); await capture(app + "-oauth-error"); ab("press", "Escape"); }
      if (mode === "success") { wait(`location.hash===${JSON.stringify(list)} && !document.querySelector('[role=dialog]')`); }
      if (mode === "delay") { wait('!!qa.oauthResolve'); nav("#/clis/settings/pi"); wait('!document.querySelector("#providers-view")'); ev("qa.oauthResolve()"); await pause(1200); assert.equal(ev("location.hash"), "#/clis/settings/pi"); }
    }
    results.push(app + ": OAuth app path, success/failure and ignored late completion after navigation");
    if (app === "desktop") {
      ab("click", "#um-trigger"); ab("fill", '.um-search-input', "providers"); ab("click", "#um-providers"); ready();
    }
    if (app === "mobile") {
      nav("#/more"); wait(`!!document.querySelector('[aria-label="Search tools and settings"]')`);
      ab("fill", '[aria-label="Search tools and settings"]', "providers"); ab("click", 'a[href="#/clis/providers/pi"]'); ready();
    }
    for (const width of (app === "desktop" ? [1920, 740, 560] : [390, 320])) {
      ab("set", "viewport", String(width), app === "mobile" ? "844" : "1000");
      ev(`document.documentElement.dataset.theme=${JSON.stringify("light")};document.documentElement.style.colorScheme="light"`);
      nav(list); ready(); await capture(app + "-width-" + width);
      const geometry = ev('(()=>{const r=document.querySelector(".cli-page .settings-card").getBoundingClientRect();return {x:r.x,width:r.width,y:r.y}})()');
      for (const hash of ["#/clis/settings/pi", "#/clis/packages/pi", list]) {
        nav(hash); wait('!!document.querySelector(".cli-page .cli-tabs")'); await pause(300);
        const current = ev('(()=>{const r=document.querySelector(".cli-page .settings-card").getBoundingClientRect();return {x:r.x,width:r.width,y:r.y}})()');
        assert.deepEqual(current, geometry, app + " " + width + " tab geometry");
        assert.ok(ev('(()=>{const nav=document.querySelector(".cli-tabs"),item=nav.querySelector("[aria-current=page]").getBoundingClientRect(),box=nav.getBoundingClientRect();return item.left>=box.left-1&&item.right<=box.right+1})()'), app + " " + width + " selected tab visible");
      }
      ready(); button("Providers CLI: Pi"); await capture(app + "-width-" + width + "-picker"); ab("press", "Escape");
    }
    results.push(app + ": responsive tab geometry and narrow overlays");
    // Enable only the composer's displayed running state at the HTTP boundary.
    // The real fixture agent remains stopped; no process or prompt is started.
    const running = workspaces.map(w => ({ ...w, agents: w.agents.map(a => a.id === agent.id ? { ...a, mode: "managed", running: true } : a) }));
    const initPath = join(out, app + "-composer-init.js");
    writeFileSync(initPath, `const originalFetch=window.fetch;window.fetch=(url,options)=>new URL(String(url),location.origin).pathname==='/api/workspaces'?Promise.resolve(new Response(${JSON.stringify(JSON.stringify(running))},{headers:{'Content-Type':'application/json'}})):originalFetch(url,options);`);
    ab("close"); session += "-composer";
    ab("--init-script", initPath, "open", new URL(`/${app}/?theme=dark#/agent/${agent.id}`, base).href);
    ab("set", "viewport", app === "mobile" ? "390" : "1365", app === "mobile" ? "844" : "1000");
    wait('!!document.querySelector("#task-input")'); intercept();
    for (const command of ["login", "logout"]) {
      nav("#/agent/" + agent.id); wait('!!document.querySelector("#task-input")');
      ab("fill", "#task-input", "/" + command);
      wait(`!!document.querySelector('.slash-menu [data-value="${command}"]')`);
      ab("click", `.slash-menu [data-value="${command}"] .slash-label`);
      wait(`location.hash===${JSON.stringify(list + (command === "login" ? "/new" : ""))}`); ready();
      if (command === "login") { wait('!!document.querySelector("[role=dialog]")'); ab("press", "Escape"); }
    }
    results.push(app + ": composer login/logout and menu search shortcuts");
    writeFileSync(join(out, app + "-results.json"), JSON.stringify({ ok: true, results: results.filter(row => row.startsWith(app + ":")) }, null, 2));
    writeFileSync(join(out, app + "-browser-errors.txt"), ab("errors"));
    console.log(app + ": PASS");
  }
  const finalFleet = await (await fetch(new URL("/api/workspaces", base))).json();
  assert.equal(finalFleet.flatMap(w => w.agents).find(a => a.id === agent.id).mode, "stopped", "Real fixture agent must remain stopped");
  writeFileSync(join(out, "results.json"), JSON.stringify({ ok: true, results }, null, 2));
} catch (error) {
  try { writeFileSync(join(out, "failure.json"), JSON.stringify({ error: String(error), snapshot: ab("snapshot", "-i") }, null, 2)); ab("screenshot", join(out, "failure.png")); } catch {}
  throw error;
} finally { try { ab("close"); } catch {} }
