// Run against a task-owned qa-scratch containing QA pi/claude-code/codex
// launched with scripts/fixtures/agent-tui.mjs; never against production.
import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { mkdirSync, writeFileSync } from "node:fs";
import { resolve } from "node:path";

const backend = process.argv[2];
const ui = process.argv[3] || backend;
for (const base of [backend, ui]) assert.match(base || "", /^http:\/\/(?:localhost|127\.0\.0\.1):\d+$/);
const out = resolve("var/screenshots/agent-tui-complete");
mkdirSync(out, { recursive: true });
const session = "agent-tui-qa";
const ab = (...args) => execFileSync("agent-browser", ["--session", session, ...args], { encoding: "utf8", maxBuffer: 4e6 }).trim();
const ev = code => JSON.parse(ab("eval", code));
const wait = code => ab("wait", "--fn", "Boolean(" + code + ")");
const click = name => { ab("find", "role", "button", "click", "--name", name, "--exact"); ab("wait", "900"); };
const api = async (path, body) => {
  const res = await fetch(backend + path, body ? { method: "POST", headers: { "content-type": "application/json" }, body: JSON.stringify(body) } : {});
  const data = await res.json(); assert.ok(res.ok, path + ": " + JSON.stringify(data)); return data;
};
const workspaces = await api("/api/workspaces");
const ws = workspaces.find(w => w.name === "TUI QA");
assert.match(ws?.path || "", /\/var\/qa\/agent-tui-complete$/);
const agents = ws.agents.filter(a => ["pi", "claude-code", "codex"].includes(a.cli) && a.name === "QA " + a.cli);
assert.equal(agents.length, 3);
const terms = (await api("/api/terminals")).terminals;
for (const a of agents) {
  const t = terms.find(t => t.id === a.terminalId);
  assert.match(t?.launchApplied?.executable || "", /\/scripts\/fixtures\/agent-tui\.mjs$/);
  await api("/api/terminals/" + t.id + "/launch/restart", { confirm: true });
}
const captures = [], checks = [];
function capture(name) {
  ab("wait", "900");
  const audit = ev("window.__picodeOverlayAudit()");
  assert.equal(audit.ok, true, name + ": " + JSON.stringify(audit));
  assert.equal(ev("document.documentElement.scrollWidth <= innerWidth"), true, name + " overflow");
  ab("screenshot", resolve(out, name + ".png")); captures.push(name);
}
function route(hash) { ev("location.hash=" + JSON.stringify(hash)); }
const paneText = key => "Array.from({length:8},(_,i)=>window.__picodeTerms.get(" + JSON.stringify(key) + ")?.term.buffer.active.getLine(i)?.translateToString()||'').join('\\n')";
try {
  ab("open", ui + "/mobile/?theme=dark");
  ab("set", "viewport", "390", "844");
  for (const a of agents) {
    const key = "sh:" + a.terminalId;
    route("#/agent/" + a.id + "?view=terminal");
    wait("!!window.__picodeTerms?.get(" + JSON.stringify(key) + ")?.sock && document.querySelector('.xterm-screen')");
    wait(paneText(key) + ".includes('Agent TUI fixture')");
    assert.ok(ev("!!document.querySelector('[aria-label=\"Attach\"]')"));
    assert.ok(ev("!!document.querySelector('[aria-label=\"Terminal actions\"]')"));
    ev("window.qaEntry=window.__picodeTerms.get(" + JSON.stringify(key) + ");true");
    const before = ev("Number((" + paneText(key) + ".match(/Scroll events: (\\d+)/)||[])[1])");
    ev("(()=>{const host=document.querySelector('.term-pane');for(const [type,y] of [['touchstart',400],['touchmove',320],['touchend',320]]){const e=new Event(type,{bubbles:true,cancelable:true});Object.defineProperty(e,'touches',{value:type==='touchend'?[]:[{clientY:y,clientX:100}]});host.dispatchEvent(e)}return true})()");
    wait("Number((" + paneText(key) + ".match(/Scroll events: (\\d+)/)||[])[1]) > " + before);
    capture(a.cli + "-terminal");
    // Alias from Agent CLIs must resolve the same owning view and xterm.
    route("#/term/" + a.terminalId);
    wait("location.hash.includes('/agent/" + a.id + "')");
    assert.equal(ev("window.qaEntry===window.__picodeTerms.get(" + JSON.stringify(key) + ")"), true);
    click("Terminal actions"); capture(a.cli + "-menu");
    const menu = ev("document.querySelector('.m-term-actions').innerText");
    for (const label of ["Files", "Git", "Send to terminal", "Stop agent", "Remove terminal"]) assert.ok(menu.includes(label), label);
    click("Done");
    click("Attach"); capture(a.cli + "-attach");
    ab("upload", "input[type=file]:not([accept])", resolve("scripts/fixtures/agent-tui.mjs"));
    wait("document.querySelector('.term-attach-chips')");
    ab("fill", "[aria-label='Message the terminal']", "QA-MESSAGE");
    click("Send · Ctrl/⌘ Enter");
    wait("!document.querySelector('[role=dialog]') || !!document.querySelector('[role=alert]')");
    assert.equal(ev("!!document.querySelector('[role=alert]')"), false, "prompt refused");
    if (a.cli !== "pi") wait(paneText(key) + ".includes('QA-MESSAGE received')");
    // Pi may use its native receiver; fake CLIs only prove terminal bytes,
    // not native vendor completion. Capture the exact server receipt too.
    checks.push(a.cli + ": shared screen/menu/attach; touch reached process; route reused xterm");
    // A dropped attach reconnects without allocating a second xterm.
    ev("window.qaEntry.sock.close();true");
    wait("window.qaEntry.sock.readyState===1");
    assert.equal(ev("window.qaEntry===window.__picodeTerms.get(" + JSON.stringify(key) + ")"), true);
    checks.push(a.cli + ": reconnect retained xterm");
  }
  const pi = agents.find(a => a.cli === "pi");
  route("#/agent/" + pi.id + "?view=terminal");
  wait("document.querySelector('[aria-label=\"Chat\"]')");
  ab("check", "[aria-label='Chat']");
  wait("document.querySelector('.m-chat')");
  assert.equal(ev("location.hash.endsWith('view=chat')"), true);
  ab("check", "[aria-label='Terminal']");
  wait("document.querySelector('.xterm-screen')");
  checks.push("Pi Chat/Terminal switch uses icon labels and respects explicit chat");
  // Small screen and keyboard accessory.
  ab("set", "viewport", "320", "700");
  ev("(()=>{const original=window.matchMedia.bind(window);window.matchMedia=q=>q==='(hover: hover) and (pointer: fine)'?{matches:false}:original(q);return true})()");
  click("Show keyboard"); wait("document.querySelector('.m-keybar')");
  capture("pi-keyboard-320");
  click("Hide keyboard"); wait("!document.querySelector('.m-keybar')");
  click("Attach"); capture("pi-attach-320"); click("Close attachments");
  click("Terminal actions"); capture("pi-menu-320"); click("Done");
  // True stopped state through canonical lifecycle; no phantom shell.
  await api("/api/terminals/" + pi.terminalId + "/launch/stop", { confirm: true });
  ev("window.dispatchEvent(new Event('focus'));true");
  wait("document.querySelector('.term-msg')");
  capture("pi-stopped");
  assert.equal(ev("!!document.querySelector('[aria-label=\"Attach\"]')"), false);
  await api("/api/terminals/" + pi.terminalId + "/launch/start", {});
  checks.push("stopped bound agent keeps terminal recovery UI and hides send controls");
  // Network error must keep a Retry, and never mount a guessed legacy pane.
  const claude = agents.find(a => a.cli === "claude-code");
  route("#/work");
  ev("window.qaFetch=window.fetch;window.fetch=(url,opts)=>String(url).endsWith('/open')?Promise.resolve(new Response(JSON.stringify({error:'QA connection unavailable'}),{status:503})):window.qaFetch(url,opts);true");
  route("#/agent/" + claude.id + "?view=terminal");
  wait("document.querySelector('[role=alert]')");
  capture("attach-error");
  ev("window.fetch=window.qaFetch;true"); click("Retry");
  wait("document.querySelector('.xterm-screen')");
  checks.push("attach failure -> visible error -> Retry -> same runtime");
  click("Attach");
  ab("fill", "[aria-label='Message the terminal']", "Keep my draft");
  ev("window.qaFetch=window.fetch;window.fetch=(url,opts)=>String(url).endsWith('/prompt')?Promise.resolve(new Response(JSON.stringify({error:'QA send refused'}),{status:409})):window.qaFetch(url,opts);true");
  click("Send · Ctrl/⌘ Enter"); wait("document.querySelector('[role=alert]')");
  assert.equal(ev("document.querySelector('[aria-label=\"Message the terminal\"]').value"), "Keep my draft");
  capture("send-error-retains-draft"); ev("window.fetch=window.qaFetch;true");
  checks.push("prompt failure remains visible and preserves the draft");
  assert.equal(ab("errors"), "", "browser errors");
  writeFileSync(resolve(out, "result.json"), JSON.stringify({ checks, captures }, null, 2));
  console.log(JSON.stringify({ passed: checks.length, screenshots: captures.length, checks }, null, 2));
} finally { ab("close"); }
