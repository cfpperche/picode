#!/usr/bin/env node
// Exercise real native-settings APIs only against the synthetic docs fixture.
// The settings pane edits one layer at a time and keeps the keyboard map on
// its own sub-tab (docs/plans/cli-settings-ux.md): this script asserts the
// layer switcher, the route it writes, provenance and the reset round trip.
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
// The pane is ready when its sub-tabs are painted and the visible form is
// editable (the keyboard map has no form of its own).
const ready = () => browser("wait", "--fn", '!!document.querySelector("#pi-settings-view .pkg-tabs .pkg-tab") && !document.querySelector(".pi-settings-fields[disabled]:not([hidden])")');
const tabClick = label => evaluate(`[...document.querySelectorAll("#pi-settings-view .pkg-tabs .pkg-tab")].find(b=>b.textContent===${JSON.stringify(label)}).click(); true`);
const layerOf = () => evaluate('document.querySelector("#pi-settings-view .settings-section")?.dataset.layer || ""');
const globalHash = "#/clis/settings/pi";
const contextHash = globalHash + "?agentId=" + encodeURIComponent(id);
// The legacy forms redirect to the canonical route; the agent survives.
const canonicalContext = "#/clis/pi/settings?agentId=" + encodeURIComponent(id);
// The fixture's machine layer is disposable: start every run from empty so a
// leftover key from an earlier run cannot shadow the provenance assertions.
writeFileSync((await api("/api/pi-settings")).global.path, "{}\n");
const compactRowSet = () => evaluate('[...document.querySelectorAll("#pi-settings-view .set-row")].find(r=>r.querySelector("#g-compact"))?.classList.contains("is-set") === true');
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
    // No agent in the route: one body (this machine), no agent pill at all.
    assert.equal(evaluate('document.querySelectorAll("#pi-settings-view .settings-section").length'), 1);
    assert.equal(layerOf(), "global");
    assert.equal(evaluate('document.querySelectorAll("#pi-settings-view .settings-layer .pkg-scope-btn").length'), 1);
    assert.equal(evaluate('document.querySelectorAll("[data-layer=agent]").length'), 0);
    assert.equal(evaluate('!!document.querySelector("#pi-settings-view .settings-ctx")'), false);
    await capture(app + "-global", true);
    const pattern = `qa-${app}-${process.pid}-*`;
    const before = await api("/api/pi-settings");
    browser("click", "#g-compact"); ready();
    assert.equal((await api("/api/pi-settings")).global.compactionEnabled, !before.global.compactionEnabled);
    results.push(app + ": global round-trip");

    // The keyboard map is a sub-tab of its own: no layer switcher, own filter.
    tabClick("Keys");
    browser("wait", "--fn", '!!document.querySelector("#keys-filter")');
    assert.equal(evaluate('location.hash.includes("tab=keys")'), true);
    assert.equal(evaluate('document.querySelectorAll("#pi-settings-view .settings-layer").length'), 0);
    assert.equal(evaluate('document.querySelectorAll("#pi-settings-view .key-row").length > 50'), true);
    await capture(app + "-keys", true);
    tabClick("Settings");
    browser("wait", "--fn", '!!document.querySelector("#pi-settings-view .settings-layer")');
    results.push(app + ": the keyboard map is its own sub-tab");

    // Provenance: a value this layer sets is marked, and Use inherited hands it
    // back to the parent instead of freezing a copy.
    browser("wait", "--fn", '[...document.querySelectorAll("#pi-settings-view .set-row")].find(r=>r.querySelector("#g-compact"))?.classList.contains("is-set") === true');
    assert.equal(evaluate('[...document.querySelectorAll("#pi-settings-view .set-row")].find(r=>r.querySelector("#g-compact"))?.querySelector(".set-src").textContent'), "Set here");
    assert.equal((await api("/api/pi-settings")).global.has.compactionEnabled, true);
    browser("find", "role", "button", "click", "--name", "Use inherited", "--exact");
    browser("wait", "--fn", '[...document.querySelectorAll("#pi-settings-view .set-row")].find(r=>r.querySelector("#g-compact"))?.classList.contains("is-set") === false');
    const afterReset = (await api("/api/pi-settings")).global;
    assert.equal(afterReset.has?.compactionEnabled ?? false, false);
    assert.equal(evaluate('[...document.querySelectorAll("#pi-settings-view .set-row")].find(r=>r.querySelector("#g-compact"))?.querySelector(".set-src").textContent'), "Pi default");
    await capture(app + "-reset", true);
    results.push(app + ": provenance marker and Use inherited round trip");

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

    // An agent in the route opens that agent's layer, and the switcher offers
    // exactly the layers this context has.
    navigate((app === "desktop" ? "#/settings" : "#/more/settings") + "?agentId=" + encodeURIComponent(id));
    ready();
    assert.equal(evaluate("location.hash"), canonicalContext);
    assert.equal(layerOf(), "agent");
    assert.deepEqual(evaluate('[...document.querySelectorAll("#pi-settings-view .settings-layer .pkg-scope-btn")].map(b=>b.textContent)'), ["This machine", "picode", "Atlas"]);
    await capture(app + "-context", true);
    if (!(await api("/api/pi-settings?agentId=" + encodeURIComponent(id))).writable.project) {
      browser("find", "role", "radio", "click", "--name", "picode", "--exact");
      browser("wait", "--fn", '!!document.querySelector("#pi-settings-view .cli-notice")');
      assert.equal(evaluate('document.querySelectorAll("#w-steer").length'), 0);
      browser("find", "role", "radio", "click", "--name", "Atlas", "--exact");
      browser("wait", "--fn", '!!document.querySelector("#ag-set-thinking")');
      results.push(app + ": untrusted workspace blocks project controls");
    }
    const layerBeforeReload = layerOf();
    browser("reload"); ready();
    assert.equal(evaluate("location.hash").startsWith(canonicalContext), true);
    assert.equal(layerOf(), layerBeforeReload);
    browser("wait", "--fn", '[...document.querySelector("#ag-set-thinking").options].some(o=>o.value==="medium")');
    browser("select", "#ag-set-thinking", "medium"); ready();
    assert.equal((await api("/api/pi-settings?agentId=" + encodeURIComponent(id))).agent.thinking, "medium");
    results.push(app + ": legacy redirect, reload and correct agent write");
    await api("/api/agents/" + encodeURIComponent(id), "PATCH", { thinking: "high" });
    browser("wait", "--fn", 'document.querySelector("#ag-set-thinking")?.value==="high"');
    results.push(app + ": external agent changes arrive through feed");

    // Switching layer rewrites the route; a reload lands on the same layer.
    browser("find", "role", "radio", "click", "--name", "This machine", "--exact");
    // The hash moves first; wait for the rendered layer, not the URL.
    browser("wait", "--fn", 'document.querySelector("#pi-settings-view .settings-section")?.dataset.layer === "global"');
    assert.equal(evaluate('location.hash.includes("layer=global")'), true);
    browser("reload"); ready();
    assert.equal(layerOf(), "global");
    assert.equal(evaluate('!!document.querySelector("#pi-settings-view .settings-file")'), true);
    results.push(app + ": the switcher writes the layer onto the route and survives a reload");

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

    // Stay inside Agent CLIs: unmounting it with the invention down would
    // (correctly) replace the whole detail with its outage notice, and this
    // row is about the settings editor, not the shell's error reporting.
    evaluate('window.qaFetch=window.fetch;window.fetch=(url,opts)=>/^\\/api\\/(clis|terminals|cli-jobs)(?:[/?]|$)/.test(String(url))?Promise.resolve(new Response(JSON.stringify({error:"QA: terminal manager unavailable"}),{status:503,headers:{"Content-Type":"application/json"}})):window.qaFetch(url,opts)');
    navigate("#/clis/pi/launch");
    browser("wait", "--fn", '!document.querySelector("#pi-settings-view")');
    navigate(globalHash); ready();
    // The settings editor is the acceptance: it renders and stays editable
    // while the terminal/CLI inventory is down. The Agent CLIs shell may
    // report that outage above it (an honest notice, not a settings failure).
    assert.equal(evaluate('document.querySelectorAll("#pi-settings-view [role=alert]").length'), 0);
    assert.equal(evaluate('!!document.querySelector("#g-compact")'), true);
    assert.equal(evaluate('document.querySelector(".pi-settings-fields").disabled'), false);
    evaluate('window.fetch=window.qaFetch');
    results.push(app + ": independent from terminal manager requests");

    // The Palette shortcut opens the layer that holds the row it names.
    navigate(contextHash + "&focus=scoped-models");
    ready();
    assert.equal(layerOf(), "global");
    browser("wait", "--fn", '(()=>{const r=document.querySelector("#scoped-models")?.getBoundingClientRect();return r&&r.top>=0&&r.bottom<=innerHeight})()');
    results.push(app + ": scoped models revealed after load, on the layer that owns it");
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
  browser("open", new URL("/mobile/?theme=dark" + globalHash + "?agentId=" + encodeURIComponent(untrustedAgent.id) + "&layer=project", base).href); ready();
  assert.equal(evaluate('document.querySelectorAll("#w-steer").length'), 0);
  // A plain anchor: the shell's own notices can sit over it on a phone, so
  // navigate it directly and assert the route the app lands on.
  browser("wait", "--fn", '!!document.querySelector(\'[data-layer="project"] a\')');
  evaluate('document.querySelector(\'[data-layer="project"] a\').click(); true');
  browser("wait", "--fn", `location.hash===${JSON.stringify("#/agent/" + untrustedAgent.id)}`);
  browser("back"); ready();
  evaluate('document.querySelector("[data-layer=project]").scrollIntoView({block:"center"})');
  await capture("mobile-untrusted-workspace");
  const rejected = await fetch(new URL("/api/pi-settings", base), { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ agentId: untrustedAgent.id, layer: "project", patch: { steeringMode: "all" } }) });
  assert.equal(rejected.status, 409);
  results.push("untrusted workspace: Trust action targets its agent; server refuses project write");
  // An unknown reset name is refused instead of quietly leaving the override.
  const badReset = await fetch(new URL("/api/pi-settings", base), { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ layer: "global", patch: { reset: ["model"] } }) });
  assert.equal(badReset.status, 400);
  results.push("unknown reset key refused");
  // Trust acceptance uses the real endpoint on this disposable workspace.
  await api("/api/agents/" + encodeURIComponent(id) + "/trust", "POST");
  browser("open", new URL("/mobile/?theme=light" + contextHash + "&layer=project", base).href); ready();
  browser("wait", "#w-steer");
  browser("select", "#w-steer", "all"); ready();
  assert.equal((await api("/api/pi-settings?agentId=" + encodeURIComponent(id))).project.steeringMode, "all");
  await capture("mobile-trusted-workspace");
  results.push("trusted workspace: project round-trip");
  const free = await api("/api/agents", "POST", { name: "Settings QA free" });
  assert.ok(free.id);
  browser("open", new URL("/mobile/" + globalHash + "?agentId=" + encodeURIComponent(free.id), base).href); ready();
  assert.equal(evaluate('document.querySelectorAll("[data-layer=project]").length'), 0);
  assert.equal(evaluate('document.querySelectorAll("[data-layer=agent]").length'), 1);
  assert.equal(evaluate('document.querySelectorAll("#pi-settings-view .settings-layer .pkg-scope-btn").length'), 2);
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
