#!/usr/bin/env node
// Adversarial settings recovery checks against a disposable docs fixture only.
import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { existsSync, mkdirSync, readFileSync, unlinkSync, writeFileSync } from "node:fs";
import { resolve } from "node:path";

const base = new URL(process.argv[2] || "http://localhost:18861");
assert.ok(["localhost", "127.0.0.1"].includes(base.hostname) && base.protocol === "http:");
const out = resolve(process.argv[3] || "var/screenshots/cli-settings-regressions/recovery");
mkdirSync(out, { recursive: true });
const api = async (path, method = "GET", body) => {
  const response = await fetch(new URL(path, base), { method, ...(body ? { headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) } : {}) });
  assert.ok(response.ok, `${method} ${path}: ${response.status}`);
  return response.status === 204 ? null : response.json();
};
const workspace = (await api("/api/workspaces")).find(w => w.name === "picode" && /[\\/]picode-docs-fixture-[^\\/]+[\\/]work[\\/]picode$/.test(w.path));
assert.ok(workspace, "Refuse to mutate a non-synthetic fixture");
const agent = workspace.agents.find(a => a.name === "Atlas");
assert.equal(agent?.mode, "stopped");
const id = agent.id;
let session = "settings-recovery-" + process.pid;
const browser = (...args) => execFileSync("agent-browser", ["--session", session, ...args], { encoding: "utf8", maxBuffer: 4 * 1024 * 1024 }).trim();
const ev = code => JSON.parse(browser("eval", code));
const wait = condition => browser("wait", "--fn", condition);
const globalHash = "#/clis/settings/pi";
const contextHash = globalHash + "?agentId=" + encodeURIComponent(id);
// The pane is ready when its sub-tabs and the visible layer body are painted.
const ready = () => wait('!!document.querySelector("#pi-settings-view .pkg-tabs .pkg-tab") && !!document.querySelector("#pi-settings-view .settings-section")');
const selectLayer = (name, id) => { browser("find", "role", "radio", "click", "--name", name, "--exact"); wait(`document.querySelector("#pi-settings-view .settings-section")?.dataset.layer === ${JSON.stringify(id)}`); };
const tabClick = label => ev(`[...document.querySelectorAll("#pi-settings-view .pkg-tabs .pkg-tab")].find(b=>b.textContent===${JSON.stringify(label)}).click(); true`);
const openKeys = () => { tabClick("Keys"); wait('!!document.querySelector("#keys-filter")'); };
const openSettingsTab = () => { tabClick("Settings"); wait('!!document.querySelector("#pi-settings-view .settings-layer")'); };
const results = [];
let visit = 0;
const open = (app, hash, theme = "light", width = 1365) => {
  browser("open", new URL(`/${app}/?theme=${theme}&qa=${process.pid}-${++visit}${hash}`, base).href);
  const [w, h] = app === "mobile" ? [390, 844] : [width, 1000];
  browser("set", "viewport", String(w), String(h));
  assert.deepEqual(ev("[innerWidth,innerHeight]"), [w, h]);
};
async function capture(name, audit = false) {
  await new Promise(done => setTimeout(done, 250));
  if (audit) {
    const report = ev("window.__picodeOverlayAudit()");
    writeFileSync(resolve(out, name + "-audit.json"), JSON.stringify(report, null, 2));
    assert.equal(report.ok, true, name);
  }
  browser("screenshot", resolve(out, name + ".png"));
}
try {
  for (const app of ["desktop", "mobile"]) {
    browser("set", "viewport", ...(app === "desktop" ? ["1365", "1000"] : ["390", "844"]));
    open(app, contextHash); ready();
    selectLayer("This machine", "global");
    const pattern = "keep-" + app + "-*";
    browser("fill", '[aria-label="Model pattern"]', pattern);
    openKeys();
    ev('document.querySelector(".key-row").scrollIntoView({block:"center"})');
    browser("click", ".key-row .btn");
    ev(`window.qaWrites=[];window.qaFetch=window.fetch;window.qaReadMode="fail";window.fetch=(url,opts)=>{
      if(opts?.method==="PUT")window.qaWrites.push(String(url));
      if(String(url).includes("/api/pi-settings")&&(!opts?.method||opts.method==="GET")){
        if(window.qaReadMode==="fail")return Promise.resolve(new Response(JSON.stringify({error:"Temporary settings read failure"}),{status:503,headers:{"Content-Type":"application/json"}}));
        if(window.qaReadMode==="hold")return new Promise(resolve=>{window.qaRelease=()=>resolve(window.qaFetch(url,opts))});
      }
      return window.qaFetch(url,opts);
    }`);
    await api("/api/agents/" + id, "PATCH", { thinking: app === "desktop" ? "low" : "high" });
    browser("wait", "#agent-clis-view [role=alert]");
    // The key capture is its own tab: cancel it there (leaving the tab also
    // cancels it — the listener dies with the mounted section).
    browser("press", "Control+Alt+y");
    assert.deepEqual(ev("window.qaWrites"), [], "A blocked key capture must not write through its window listener");
    browser("find", "role", "button", "click", "--name", "Cancel");
    openSettingsTab();
    const draft = () => ev('document.querySelector("[data-layer=global] .set-pat-add input").value');
    assert.equal(draft(), pattern, "the layer keeps its own unfinished pattern");
    assert.equal(ev('document.querySelector(".pi-settings-fields").disabled'), true, "writes are blocked");
    // The agent layer is blocked by the same stale context.
    selectLayer("Atlas", "agent");
    assert.equal(ev('document.querySelector("#ag-set-thinking").matches(":disabled")'), true);
    selectLayer("This machine", "global");
    ev('document.querySelector("#agent-clis-view [role=alert]").scrollIntoView({block:"center"})');
    await capture(app + "-refresh-error");
    browser("click", "#agent-clis-view [role=alert] button");
    wait('!document.querySelector("#agent-clis-view [role=alert] button").disabled');
    assert.equal(draft(), pattern);
    selectLayer("Atlas", "agent");
    assert.equal(ev('document.querySelector("#ag-set-thinking").matches(":disabled")'), true);
    selectLayer("This machine", "global");
    ev('window.qaReadMode="hold"');
    browser("click", "#agent-clis-view [role=alert] button");
    wait('typeof window.qaRelease==="function"');
    selectLayer("Atlas", "agent");
    assert.equal(ev('document.querySelector("#ag-set-thinking").matches(":disabled")'), true);
    selectLayer("This machine", "global");
    assert.equal(draft(), pattern);
    ev('window.qaReadMode="pass";window.qaRelease();true'); ready();
    assert.equal(draft(), pattern);
    ev('window.fetch=window.qaFetch');
    // The keyboard map still works after the retry, on its own tab.
    openKeys();
    ev('document.querySelector(".key-row").scrollIntoView({block:"center"})');
    browser("click", ".key-row .btn");
    browser("find", "role", "button", "click", "--name", "Cancel");
    results.push(app + ": draft retained across failed refresh, repeated failure and pending retry; writes resume only after validation");
  }

  // Exercise the actual desktop App adapter. Every runtime POST is intercepted;
  // only PATCH and reads reach the synthetic server. A real process never starts.
  browser("set", "viewport", "1365", "1000");
  open("desktop", "#/preferences"); browser("wait", "#preferences-view");
  wait('document.querySelector("#sidebar")?.textContent.includes("Borealis")');
  for (const mode of ["managed", "interactive"]) for (const failure of ["patch", "stop", "start", "none"]) {
    console.log("Desktop restart boundary:", mode, failure);
    await api("/api/agents/" + id, "PATCH", { opMode: "full" });
    ev('location.hash="#/preferences"'); browser("wait", "#preferences-view");
    ev(`window.qaFetch=window.fetch;window.qaCalls=[];window.qaMode=${JSON.stringify(mode)};window.fetch=async(url,opts)=>{
      const path=String(url);const prefix=${JSON.stringify("/api/agents/" + id)};
      const response=(status,body)=>new Response(JSON.stringify(body),{status,headers:{"Content-Type":"application/json"}});
      if(opts?.method==="PATCH"&&path===prefix){
        window.qaCalls.push("patch");
        if(${JSON.stringify(failure)}==="patch")return response(503,{error:"Save unavailable"});
      }
      if(opts?.method==="POST"&&path.startsWith("/api/agents/")&&["/close","/open","/managed/start","/managed/stop"].some(suffix=>path.endsWith(suffix))){
        const stage=path.endsWith("/close")||path.endsWith("/stop")?"stop":"start";
        window.qaCalls.push(stage);
        if(${JSON.stringify(failure)}===stage)return response(503,{error:stage+" unavailable"});
        window.qaMode=stage==="stop"?"stopped":${JSON.stringify(mode)};
        return response(200,{});
      }
      const res=await window.qaFetch(url,opts);
      if(path==="/api/workspaces"&&res.ok){const rows=await res.json();for(const w of rows)for(const a of w.agents||[])if(a.id===${JSON.stringify(id)})a.mode=window.qaMode;return response(200,rows)}
      return res;
    };location.hash=${JSON.stringify(contextHash)}`);
    ready();
    ev('document.querySelector("[data-layer=agent]").scrollIntoView({block:"center"})');
    browser("click", "#pi-settings-view #agent-mode");
    browser("wait", ".search-combo-pop");
    await capture("desktop-tools-" + mode + "-" + failure, true);
    browser("find", "role", "option", "click", "--name", "Read-only read, grep, find, ls");
    if (failure === "none") {
      wait('window.qaCalls.join(",")==="patch,stop,start"');
      wait('document.querySelector("[data-layer=agent] [role=status]")?.textContent==="Saved."');
      assert.equal(ev('document.querySelectorAll("[data-layer=agent] [role=alert]").length'), 0);
    } else {
      browser("wait", "[data-layer=agent] [role=alert]");
      const message = ev('document.querySelector("[data-layer=agent] [role=alert]").textContent');
      assert.match(message, failure === "patch" ? /Save unavailable/ : /Settings saved, but the agent could not restart/);
      assert.notEqual(ev('document.querySelector("[data-layer=agent] [role=status]").textContent'), "Saved.");
      if (failure !== "patch") wait('document.querySelector("#pi-settings-view #agent-mode").textContent.includes("Read-only")');
      assert.deepEqual(ev("window.qaCalls"), failure === "patch" ? ["patch"] : failure === "stop" ? ["patch", "stop"] : ["patch", "stop", "start"]);
      await capture("desktop-" + mode + "-" + failure + "-error");
    }
    ev('window.fetch=window.qaFetch');
    results.push("desktop " + mode + ": " + failure + " failure boundary");
  }
  await api("/api/agents/" + id, "PATCH", { opMode: "full" });
  for (const [width, theme] of [[1365, "dark"], [560, "light"]]) {
    // Start a clean browser after the runtime fault-injection matrix.
    browser("close");
    session = "settings-recovery-" + process.pid + "-visual-" + width;
    open("desktop", contextHash, theme, width); ready();
    for (const chip of ["mode", "checklist"]) {
      ev('document.querySelector("[data-layer=agent]").scrollIntoView({block:"center"})');
      browser("click", "#pi-settings-view #agent-" + chip);
      browser("wait", ".search-combo-pop");
      await capture("desktop-" + width + "-" + chip, true);
      browser("press", "Escape");
    }
  }
  results.push("desktop: Tools and Checklist contained at wide/narrow widths in dark/light themes");
  // Real malformed native JSON must not block explicit agent-only settings.
  await api("/api/agents/" + id, "PATCH", { provider: "", model: "", thinking: "low", opMode: "full" });
  const settingsPath = (await api("/api/pi-settings")).global.path;
  assert.ok(/[\\/]picode-docs-fixture-[^\\/]+[\\/]home[\\/]/.test(settingsPath));
  const original = existsSync(settingsPath) ? readFileSync(settingsPath) : null;
  try {
    writeFileSync(settingsPath, '{"compaction":');
    open("mobile", "#/agent/" + id); browser("wait", "textarea");
    browser("find", "role", "button", "click", "--name", "Settings");
    browser("wait", "#pi-settings-view [role=alert]");
    assert.equal(ev('document.querySelector("#ag-set-thinking").matches(":disabled")'), false);
    browser("select", "#ag-set-thinking", "high");
    wait('document.querySelector("[data-layer=agent] [role=status]")?.textContent==="Saved."');
    const saved = (await api("/api/workspaces")).find(w => w.id === workspace.id).agents.find(a => a.id === id);
    assert.equal(saved.thinking, "high");
    assert.ok(!saved.provider && !saved.model, "Do not guess inherited defaults when their file cannot be read");
    await capture("mobile-malformed-defaults", true);
    results.push("mobile: malformed native file leaves explicit quick settings editable without inventing provider/model overrides");
    open("desktop", globalHash); browser("wait", "#pi-settings-view [role=alert]");
    openKeys();
    browser("wait", 'input[placeholder="Filter keys"]');
    assert.equal(ev('[...document.querySelectorAll("input")].find(input=>input.placeholder==="Filter keys").matches(":disabled")'), false);
    results.push("global: keys remain available when native defaults fail");
  } finally { if (original) writeFileSync(settingsPath, original); else unlinkSync(settingsPath); }

  assert.equal((await api("/api/workspaces")).find(w => w.id === workspace.id).agents.find(a => a.id === id).mode, "stopped");
  writeFileSync(resolve(out, "results.json"), JSON.stringify({ ok: true, results }, null, 2));
  console.log(JSON.stringify({ ok: true, results }, null, 2));
} catch (error) {
  try {
    writeFileSync(resolve(out, "failure.json"), JSON.stringify(ev('({url:location.href,body:document.body.innerText,calls:window.qaCalls})'), null, 2));
    browser("screenshot", resolve(out, "failure.png"));
  } catch { /* preserve the original assertion */ }
  throw error;
} finally { browser("close"); }
