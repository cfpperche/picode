#!/usr/bin/env node
// Exercise real native-settings APIs only against the synthetic docs fixture.
import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { mkdirSync, writeFileSync } from "node:fs";
import { resolve } from "node:path";
const base = new URL(process.argv[2] || "http://localhost:18861");
assert.ok(["localhost", "127.0.0.1"].includes(base.hostname) && base.protocol === "http:");
const out = resolve(process.argv[3] || "var/screenshots/cli-native-settings/browser");
mkdirSync(out, { recursive: true });
const api = async (path, method = "GET", body) => {
  const response = await fetch(new URL(path, base), { method, ...(body ? { headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) } : {}) });
  assert.ok(response.ok, `${method} ${path}: ${response.status}`);
  return response.status === 204 ? null : response.json();
};
const workspaces = await api("/api/workspaces");
const workspace = workspaces.find(w => w.name === "picode" && /[\\/]picode-docs-fixture-[^\\/]+[\\/]work[\\/]picode$/.test(w.path));
assert.ok(workspace, "Refuse to mutate a non-synthetic fixture");
const agent = workspace.agents.find(a => a.name === "Atlas");
assert.equal(agent?.mode, "stopped");
const id = agent.id;
const session = "cli-settings-qa-" + process.pid;
const browser = (...args) => execFileSync("agent-browser", ["--session", session, ...args], { encoding: "utf8", maxBuffer: 4 * 1024 * 1024 }).trim();
const evaluate = code => JSON.parse(browser("eval", code));
const navigate = hash => evaluate(`location.hash=${JSON.stringify(hash)}`);
const ready = () => browser("wait", "--fn", '!document.querySelector(".pi-settings-fields")?.disabled && !!document.querySelector("#g-compact")');
const globalHash = "#/clis/settings/pi";
const contextHash = globalHash + "?agentId=" + encodeURIComponent(id);
const results = [];
async function capture(name, audit = false) {
  // Let native controls finish painting before judging pixels.
  await new Promise(resolve => setTimeout(resolve, 250));
  if (audit) {
    const report = evaluate("window.__picodeOverlayAudit()");
    writeFileSync(resolve(out, name + "-audit.json"), JSON.stringify(report, null, 2));
    assert.equal(report.ok, true, name);
  }
  browser("screenshot", resolve(out, name + ".png"));
}
try {
  for (const app of ["desktop", "mobile"]) {
    browser("set", "viewport", ...(app === "desktop" ? ["1365", "1000"] : ["390", "844"]));
    browser("open", new URL(`/${app}/?theme=dark${globalHash}`, base).href);
    ready();
    assert.equal(evaluate('document.querySelectorAll("[data-layer=agent]").length'), 0);
    assert.equal(evaluate('document.querySelectorAll("[data-layer=workspace]").length'), 0);
    await capture(app + "-global");
    const pattern = `qa-${app}-${process.pid}-*`;
    const before = await api("/api/pi-settings");
    browser("click", "#g-compact"); ready();
    assert.equal((await api("/api/pi-settings")).global.compactionEnabled, !before.global.compactionEnabled);
    results.push(app + ": global round-trip");

    navigate(globalHash.replace("/pi", "/codex"));
    browser("wait", ".cli-notice");
    assert.equal(evaluate('document.querySelectorAll("#g-compact").length'), 0);
    await capture(app + "-unsupported");
    results.push(app + ": unsupported CLI refuses Pi editor");

    navigate(globalHash + "?agentId=missing-settings-fixture-agent");
    browser("wait", "[role=alert]");
    assert.equal(evaluate('document.querySelectorAll("#g-compact").length'), 0);
    await capture(app + "-missing-agent");
    results.push(app + ": missing agent refuses fallback");

    navigate((app === "desktop" ? "#/settings" : "#/more/settings") + "?agentId=" + encodeURIComponent(id));
    ready();
    assert.equal(evaluate("location.hash"), contextHash);
    assert.equal(evaluate('document.querySelectorAll("[data-layer=agent]").length'), 1);
    await capture(app + "-context");
    if (!(await api("/api/pi-settings?agentId=" + encodeURIComponent(id))).writable.project) {
      assert.equal(evaluate('document.querySelectorAll("#w-steer").length'), 0);
      results.push(app + ": untrusted workspace blocks project controls");
    }
    browser("reload"); ready();
    assert.equal(evaluate("location.hash"), contextHash);
    browser("wait", "--fn", '[...document.querySelector("#ag-set-thinking").options].some(o=>o.value==="medium")');
    browser("select", "#ag-set-thinking", "medium"); ready();
    assert.equal((await api("/api/pi-settings?agentId=" + encodeURIComponent(id))).agent.thinking, "medium");
    results.push(app + ": legacy redirect, reload and correct agent write");
    await api("/api/agents/" + encodeURIComponent(id), "PATCH", { thinking: "high" });
    browser("wait", "--fn", 'document.querySelector("#ag-set-thinking")?.value==="high"');
    results.push(app + ": external agent changes arrive through feed");

    navigate(globalHash);
    ready();
    evaluate('window.qaFetch=window.fetch;window.fetch=(url,opts)=>opts?.method==="PUT"&&String(url).includes("/api/pi-settings")?Promise.resolve(new Response(JSON.stringify({error:"QA: save unavailable"}),{status:503,headers:{"Content-Type":"application/json"}})):window.qaFetch(url,opts)');
    const compact = evaluate('document.querySelector("#g-compact").getAttribute("aria-checked")');
    browser("click", "#g-compact"); ready(); browser("wait", "[role=alert]");
    assert.equal(evaluate('document.querySelector("#g-compact").getAttribute("aria-checked")'), compact);
    browser("fill", '[aria-label="Model pattern"]', pattern);
    browser("click", ".set-pat-add button"); ready();
    assert.equal(evaluate('document.querySelector(".set-pat-add input").value'), pattern);
    await capture(app + "-save-error");
    evaluate('window.fetch=window.qaFetch');
    browser("click", ".set-pat-add button"); ready();
    assert.equal(evaluate('document.querySelector(".set-pat-add input").value'), "");
    results.push(app + ": failed save rolls back and retains pattern for retry");

    navigate("#/preferences");
    evaluate('window.qaFetch=window.fetch;window.fetch=(url,opts)=>String(url).includes("/api/pi-settings")?Promise.resolve(new Response(JSON.stringify({error:"QA: settings unavailable"}),{status:503,headers:{"Content-Type":"application/json"}})):window.qaFetch(url,opts)');
    navigate(globalHash); browser("wait", "#pi-settings-view [role=alert]");
    await capture(app + "-read-error");
    evaluate('window.fetch=window.qaFetch');
    browser("click", "#pi-settings-view [role=alert] button"); ready();
    results.push(app + ": initial failure retries");

    navigate("#/preferences");
    evaluate('window.qaFetch=window.fetch;window.fetch=(url,opts)=>/^\\/api\\/(clis|terminals|cli-jobs)(?:[/?]|$)/.test(String(url))?Promise.resolve(new Response(JSON.stringify({error:"QA: terminal manager unavailable"}),{status:503,headers:{"Content-Type":"application/json"}})):window.qaFetch(url,opts)');
    navigate(globalHash); ready();
    assert.equal(evaluate('document.querySelectorAll("#agent-clis-view [role=alert]").length'), 0);
    evaluate('window.fetch=window.qaFetch');
    results.push(app + ": independent from terminal manager requests");

    navigate(contextHash + "&focus=scoped-models"); ready();
    browser("wait", "--fn", '(()=>{const r=document.querySelector("#scoped-models")?.getBoundingClientRect();return r&&r.top>=0&&r.bottom<=innerHeight})()');
    results.push(app + ": scoped models revealed after load");
    if (app === "mobile") {
      browser("find", "role", "button", "click", "--name", "Back");
      browser("wait", "--fn", 'location.hash==="#/clis"');
      results.push("mobile: Back returns to Agent CLIs");
    }
  }
  // A fresh owned folder proves the blocked branch even on repeat runs.
  const untrustedPath = resolve(workspace.path, "..", "settings-untrusted-" + process.pid);
  mkdirSync(untrustedPath);
  const untrustedWorkspace = await api("/api/workspaces", "POST", { name: "Settings untrusted", path: untrustedPath });
  const untrustedAgent = await api("/api/workspaces/" + untrustedWorkspace.id + "/agents", "POST", { name: "Settings scope" });
  browser("open", new URL("/mobile/?theme=dark" + globalHash + "?agentId=" + encodeURIComponent(untrustedAgent.id), base).href); ready();
  assert.equal(evaluate('document.querySelectorAll("#w-steer").length'), 0);
  browser("click", '[data-layer="workspace"] a');
  browser("wait", "--fn", `location.hash===${JSON.stringify("#/agent/" + untrustedAgent.id)}`);
  browser("back"); ready();
  evaluate('document.querySelector("[data-layer=workspace]").scrollIntoView({block:"center"})');
  await capture("mobile-untrusted-workspace");
  const rejected = await fetch(new URL("/api/pi-settings", base), { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ agentId: untrustedAgent.id, layer: "project", patch: { steeringMode: "all" } }) });
  assert.equal(rejected.status, 409);
  results.push("untrusted workspace: Trust action targets its agent; server refuses project write");
  // Trust acceptance uses the real endpoint on this disposable workspace.
  await api("/api/agents/" + encodeURIComponent(id) + "/trust", "POST");
  browser("open", new URL("/mobile/?theme=light" + contextHash, base).href); ready();
  browser("wait", "#w-steer");
  browser("select", "#w-steer", "all"); ready();
  assert.equal((await api("/api/pi-settings?agentId=" + encodeURIComponent(id))).project.steeringMode, "all");
  await capture("mobile-trusted-workspace");
  results.push("trusted workspace: project round-trip");
  const free = await api("/api/agents", "POST", { name: "Settings QA free" });
  assert.ok(free.id);
  browser("open", new URL("/mobile/" + globalHash + "?agentId=" + encodeURIComponent(free.id), base).href); ready();
  assert.equal(evaluate('document.querySelectorAll("[data-layer=workspace]").length'), 0);
  assert.equal(evaluate('document.querySelectorAll("[data-layer=agent]").length'), 1);
  results.push("free agent: agent layer without workspace");
  await api("/api/agents/" + encodeURIComponent(free.id), "DELETE");
  browser("wait", "[role=alert]");
  assert.equal(evaluate('document.querySelectorAll("#ag-set-thinking").length'), 0);
  results.push("deleted open agent: feed removes editable fields");
  browser("set", "viewport", "560", "900");
  browser("open", new URL("/desktop/?theme=light" + globalHash, base).href); ready();
  await capture("desktop-narrow");
  console.log(JSON.stringify({ ok: true, results }, null, 2));
  writeFileSync(resolve(out, "results.json"), JSON.stringify({ ok: true, results }, null, 2));
} finally { browser("close"); }
