// Disposable scratch only. Lifecycle responses are simulated: no model runs.
// Usage: node scripts/qa-agent-menu.mjs http://localhost:8473
import { execFileSync } from "node:child_process";
import { mkdirSync } from "node:fs";
import { resolve } from "node:path";
import assert from "node:assert/strict";

const base = process.argv[2];
assert.match(base || "", /^http:\/\/(localhost|127\.0\.0\.1):\d+$/);
const out = resolve("var/screenshots/agent-menu");
mkdirSync(out, { recursive: true });
const browser = (...args) => execFileSync("agent-browser", ["--session", "agent-menu", ...args], { encoding: "utf8" }).trim();
const evaluate = (js) => JSON.parse(browser("eval", js));
const click = (name) => browser("find", "role", "button", "click", "--name", name, "--exact");
const item = (name) => browser("find", "role", "menuitem", "click", "--name", name, "--exact");
const capture = (name) => {
  browser("wait", "350");
  const audit = evaluate("window.__picodeOverlayAudit()");
  assert.equal(audit.ok, true, name + " overlay audit: " + JSON.stringify(audit));
  browser("screenshot", resolve(out, name + ".png"));
};
const agents = [
  { id: "pi-stopped", name: "Pi stopped", cli: "pi", mode: "stopped" },
  { id: "pi-interactive", name: "Pi interactive", cli: "pi", mode: "interactive" },
  { id: "pi-managed", name: "Pi managed", cli: "pi", mode: "managed" },
  { id: "codex", name: "Codex running", cli: "codex", terminalId: "tc", mode: "stopped" },
  { id: "claude", name: "Claude stopped", cli: "claude-code", terminalId: "tl", mode: "stopped" },
  { id: "muse", name: "Muse no settings", cli: "muse", terminalId: "tm", mode: "stopped" },
].map((a) => ({ ...a, workspaceId: "qa-menu", cwd: "/tmp", status: a.mode === "stopped" ? "stopped" : "running" }));
const terminals = [
  { id: "tc", name: "Codex running", cli: "codex", running: true, state: "idle", lastSession: "qa-session" },
  { id: "tl", name: "Claude stopped", cli: "claude-code", running: false },
  { id: "tm", name: "Muse no settings", cli: "muse", running: false },
].map((t) => ({ ...t, workspaceId: "qa-menu", cwd: "/tmp" }));
const route = (path, data) => {
  browser("network", "unroute", base + path);
  browser("network", "route", base + path, "--body", JSON.stringify(data));
};
const menu = (name) => { click("Actions for " + name); browser("wait", '[role="menu"]'); };
try {
  browser("open", base + "/desktop/?theme=dark");
  browser("set", "viewport", "1366", "1000");
  route("/api/workspaces", []);
  route("/api/terminals", { terminals: [] });
  browser("reload");
  browser("wait", "--text", "No agents yet");
  capture("empty");
  route("/api/workspaces", [{ id: "qa-menu", name: "Menu QA", path: "/tmp", agents }]);
  route("/api/terminals", { terminals });
  route("/api/clis", { clis: ["pi", "codex", "claude-code", "muse"].map((id) => ({ id, name: id, installed: true, integrationCapable: id !== "muse" })) });
  browser("reload");
  browser("wait", '[aria-label="Actions for Pi stopped"]');
  for (const a of agents) {
    menu(a.name);
    capture(a.id);
    const labels = evaluate('Array.from(document.querySelectorAll("[role=menuitem]")).map(e=>e.textContent.trim())');
    assert.equal(labels.at(-1), "Remove agent");
    assert.equal(labels.includes("Open chat"), a.cli === "pi");
    assert.equal(labels.includes("Launch settings"), a.cli !== "muse");
    assert.ok(labels.includes(a.mode === "stopped" && a.cli !== "codex" ? "Start agent" : "Restart agent"));
    browser("press", "Escape");
  }
  // Intercept mutations inside this browser only; record exact dispatch and
  // deliberately keep the fixture state stable across mode-specific checks.
  evaluate(`(()=>{const original=window.fetch;window.qaCalls=[];window.qaFail="";window.fetch=async(input,opts={})=>{const url=String(input);if(opts.method==="POST"&&(url.includes("/api/agents/")||url.includes("/api/terminals/"))){window.qaCalls.push(url);return new Response(JSON.stringify(url.includes(window.qaFail)&&window.qaFail?{error:"QA restart unavailable"}:{}),{status:url.includes(window.qaFail)&&window.qaFail?503:200,headers:{"Content-Type":"application/json"}})}return original(input,opts)};return true})()`);
  for (const [name, expected] of [
    ["Pi interactive", ["/api/agents/pi-interactive/open?restart=1"]],
    ["Pi managed", ["/api/agents/pi-managed/close", "/api/agents/pi-managed/managed/start"]],
    ["Codex running", ["/api/terminals/tc/launch/restart"]],
  ]) {
    evaluate("window.qaCalls=[]");
    menu(name); item("Restart agent");
    browser("wait", '[role="alertdialog"]');
    capture(name.toLowerCase().replaceAll(" ", "-") + "-confirm");
    click("Cancel");
    assert.deepEqual(evaluate("window.qaCalls"), []);
    menu(name); item("Restart agent"); click("Restart agent");
    browser("wait", "--text", "Agent restarted.");
    assert.deepEqual(evaluate("window.qaCalls"), expected);
    evaluate('document.querySelectorAll(".notice-x").forEach(e=>e.click());true');
  }
  for (const [name, expected] of [["Pi interactive", "/api/agents/pi-interactive/close"], ["Pi managed", "/api/agents/pi-managed/close"], ["Codex running", "/api/terminals/tc/launch/stop"]]) {
    evaluate("window.qaCalls=[]");
    menu(name); item("Stop agent"); click("Stop agent");
    browser("wait", "--text", "Agent stopped.");
    assert.deepEqual(evaluate("window.qaCalls"), [expected]);
    evaluate('document.querySelectorAll(".notice-x").forEach(e=>e.click());true');
  }
  for (const [name, expected] of [
    ["Pi stopped", ["/api/agents/pi-stopped/managed/start"]],
    ["Claude stopped", ["/api/terminals/tl/launch/start", "/api/terminals/tl/open"]],
  ]) {
    evaluate("window.qaCalls=[]");
    menu(name); item("Start agent");
    browser("wait", "--fn", `window.qaCalls.length>=${expected.length}`);
    assert.deepEqual(evaluate("window.qaCalls"), expected);
  }
  // A failed managed stop must not start a replacement run.
  evaluate('window.qaCalls=[];window.qaFail="/close"');
  menu("Pi managed"); item("Restart agent"); click("Restart agent");
  browser("wait", "--text", "QA restart unavailable");
  assert.deepEqual(evaluate("window.qaCalls"), ["/api/agents/pi-managed/close"]);
  capture("restart-error");
  menu("Pi interactive"); item("Launch settings");
  assert.ok(browser("get", "url").endsWith("#/clis/pi/settings?agentId=pi-interactive"));
  browser("open", base + "/desktop/?theme=light");
  browser("wait", '[aria-label="Actions for Codex running"]');
  menu("Codex running");
  browser("find", "role", "menuitem", "hover", "--name", "Restart agent", "--exact");
  capture("codex-light");
  item("Launch settings");
  assert.ok(browser("get", "url").endsWith("#/clis/terminal/tc"));
  console.log("PASS: menu matrix, scoped settings, stop/restart dispatch, cancellation, failure, dark/light overlays. " + out);
} catch (error) {
  console.error(browser("snapshot", "-i"));
  throw error;
} finally { browser("close"); }
