#!/usr/bin/env node
// Run against a disposable picode-docs-fixture, never the installed service.
// Usage: node scripts/qa-mobile-settings.mjs http://localhost:18777 [output-dir]
import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { mkdirSync, writeFileSync } from "node:fs";
import { resolve } from "node:path";

const base = new URL(process.argv[2] || "http://localhost:18777");
assert.ok(["localhost", "127.0.0.1"].includes(base.hostname) && base.protocol === "http:", "Use a loopback HTTP fixture");
const out = resolve(process.argv[3] || "var/mobile-settings-qa");
mkdirSync(out, { recursive: true });
const session = "mobile-settings-qa-" + process.pid;
const browser = (...args) => execFileSync("agent-browser", ["--session", session, ...args], { encoding: "utf8", maxBuffer: 4 * 1024 * 1024 }).trim();
const evaluate = code => JSON.parse(browser("eval", code));
const settle = () => new Promise(done => setTimeout(done, 800));
const readFleet = async () => {
  const response = await fetch(new URL("/api/workspaces", base));
  assert.ok(response.ok);
  return response.json();
};
const fleet = await readFleet();
const workspace = fleet.find(w => w.name === "picode" && /[\\/]picode-docs-fixture-[^\\/]+[\\/]work[\\/]picode$/.test(w.path));
assert.ok(workspace, "Refuse to mutate a non-synthetic workspace");
const agent = workspace.agents.find(a => a.name === "Atlas");
assert.equal(agent?.mode, "stopped", "QA must not start or restart a real agent");
const id = agent.id;
const saved = async () => (await readFleet()).flatMap(w => w.agents).find(a => a.id === id);
const waitSaved = () => browser("wait", "--fn", '!document.querySelector(".m-agent-config-fields").disabled');
const click = name => browser("find", "role", "button", "click", "--name", name);
async function capture(name, audit = false) {
  await settle();
  if (audit) assert.equal(evaluate("window.__picodeOverlayAudit().ok"), true, name + " overlay geometry");
  browser("screenshot", resolve(out, name + ".png"));
}
try {
  const seed = await fetch(new URL("/api/agents/" + encodeURIComponent(id), base), {
    method: "PATCH", headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ provider: "anthropic", model: "claude-sonnet-4-6", thinking: "low", opMode: "full", checklist: "changes" }),
  });
  assert.ok(seed.ok);
  browser("set", "viewport", "390", "844");
  browser("open", new URL("/mobile/?theme=light#/agent/" + encodeURIComponent(id), base).href);
  browser("wait", "textarea");
  browser("fill", "textarea", "Keep this draft while I adjust settings");
  evaluate('(()=>{window.qaComposer=document.querySelector("textarea");const d=new DataTransfer();const bytes=Uint8Array.from(atob("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+jp1sAAAAASUVORK5CYII="),c=>c.charCodeAt(0));d.items.add(new File([bytes],"qa.png",{type:"image/png"}));window.qaComposer.dispatchEvent(new ClipboardEvent("paste",{clipboardData:d,bubbles:true,cancelable:true}));return true})()');
  browser("wait", ".composer-pics");
  click("Settings");
  browser("wait", "#ag-set-thinking");
  await capture("settings", true);
  browser("select", "#ag-set-thinking", "high");
  waitSaved();
  assert.equal((await saved()).thinking, "high");
  click("Done");
  await settle();
  assert.equal(evaluate('window.qaComposer===document.querySelector("textarea")'), true);
  assert.equal(evaluate('document.querySelector("textarea").value'), "Keep this draft while I adjust settings");
  assert.equal(evaluate('document.querySelectorAll(".composer-pics img").length'), 1);
  await capture("retained-draft");

  click("Settings");
  await settle();
  evaluate('(()=>{const real=window.fetch;window.fetch=(url,opts)=>opts?.method==="PATCH"?Promise.resolve(new Response(JSON.stringify({error:"QA: server unavailable"}),{status:503,headers:{"Content-Type":"application/json"}})):real(url,opts);window.qaRestoreFetch=()=>{window.fetch=real};return true})()');
  browser("select", "#ag-set-thinking", "off");
  browser("wait", '[role="alert"]');
  assert.equal(evaluate('document.querySelector("#ag-set-thinking").value'), "high");
  assert.equal((await saved()).thinking, "high");
  await capture("save-error", true);
  evaluate('(()=>{window.qaRestoreFetch();return true})()');
  click("Dismiss");
  browser("select", "#ag-set-thinking", "off");
  waitSaved();
  assert.equal((await saved()).thinking, "off");
  evaluate('(()=>{document.querySelector("#ag-set-model").add(new Option("claude-opus-4-6","claude-opus-4-6"));return true})()');
  browser("select", "#ag-set-model", "claude-opus-4-6");
  waitSaved();
  assert.equal((await saved()).model, "claude-opus-4-6");
  browser("click", "#agent-mode");
  await capture("tool-selector", true);
  browser("find", "role", "option", "click", "--name", "Read-only read, grep, find, ls");
  waitSaved();
  assert.equal((await saved()).opMode, "readonly");
  browser("click", "#agent-mode");
  browser("find", "role", "option", "click", "--name", "Full All tools");
  waitSaved();
  assert.equal((await saved()).opMode || "full", "full");
  browser("click", "#agent-checklist");
  browser("find", "role", "option", "click", "--name", "Always Every task, read-only answers too");
  waitSaved();
  assert.equal((await saved()).checklist, "always");
  click("Done");
  await settle();
  browser("open", new URL("/mobile/#/clis/pi/settings?agentId=" + encodeURIComponent(id), base).href);
  browser("wait", "#ag-set-thinking");
  browser("select", "#ag-set-thinking", "medium");
  waitSaved();
  assert.equal((await saved()).thinking, "medium");
  assert.equal((await saved()).mode, "stopped");
  writeFileSync(resolve(out, "result.json"), JSON.stringify({ ok: true, draft: true, attachment: true, rollback: true, model: true, tools: true, checklist: true, more: true, noModelTurns: true }, null, 2) + "\n");
  console.log("Mobile Settings browser regressions: PASS");
} finally {
  browser("close");
}
