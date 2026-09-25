#!/usr/bin/env node
// Smoke-test the Providers pane (ADR-0169: one pane, CliCredentials, for all
// nine CLIs) against an owned synthetic fixture: its canonical route, the Add
// door as a route, the custom-endpoint sub-route, the blocked states and the
// llama.cpp alias. Nothing signs in and nothing is written: every door here is
// a read or a dialog that is closed again.
//
// The suite this replaces drove Pi's own provider editor (#providers-view),
// which ADR-0169 removed on 2026-09-21; the account flows (pause, activate,
// OAuth completion) of the new pane have no browser suite yet — recorded in
// docs/handoff/open/legacy-compat.md.
//
// Usage: node scripts/qa-cli-providers.mjs http://127.0.0.1:18861 [output-dir]
import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { mkdirSync, writeFileSync } from "node:fs";
import { resolve, join } from "node:path";

const base = new URL(process.argv[2]);
assert.ok(base.protocol === "http:" && ["localhost", "127.0.0.1"].includes(base.hostname));
const out = resolve(process.argv[3] || "var/screenshots/cli-providers");
mkdirSync(out, { recursive: true });
const workspaces = await (await fetch(new URL("/api/workspaces", base))).json();
const workspace = workspaces.find(w => w.name === "picode" && /\/picode-docs-fixture-[^/]+\/work\/picode$/.test(w.path));
assert.ok(workspace, "Refuse non-synthetic fixture");
const agent = workspace.agents.find(a => a.name === "Atlas");

let session = "cli-providers-" + process.pid;
const ab = (...args) => execFileSync("agent-browser", ["--session", session, ...args], { encoding: "utf8", maxBuffer: 4 * 1024 * 1024 }).trim();
const ev = code => JSON.parse(ab("eval", code));
const wait = code => ab("wait", "--fn", code);
const nav = hash => ev(`location.hash=${JSON.stringify(hash)}`);
const list = "#/clis/pi/providers";
// The pane is ready when its accounts have loaded: the frame is there and the
// skeleton is gone (an unreadable vault or a missing store is a notice, and
// the fixture has neither).
const ready = () => wait('!!document.querySelector("#cli-credentials-view") && !document.querySelector("#cli-credentials-view .cred-loading")');
const results = [];
async function capture(name, audit = false) {
  await new Promise(done => setTimeout(done, 250));
  if (audit) {
    const report = ev("window.__picodeOverlayAudit()");
    writeFileSync(join(out, name + "-audit.json"), JSON.stringify(report, null, 2));
    assert.equal(report.ok, true, name);
  }
  ab("screenshot", join(out, name + ".png"));
}

try {
  for (const app of ["desktop", "mobile"]) {
    try { ab("close"); } catch {}
    session = "cli-providers-" + process.pid + "-" + app;
    ab("open", new URL(`/${app}/?theme=dark#/more`, base).href);
    ab("set", "viewport", app === "mobile" ? "390" : "1365", app === "mobile" ? "844" : "1000");
    wait('!!document.querySelector("#m-app, #app")');

    nav(list); wait(`location.hash===${JSON.stringify(list)}`); ready();
    await capture(app + "-roster", true);
    results.push(app + ": the canonical route renders Pi's accounts");

    // The Add door is an address: opening it shows the dialog, Escape closes
    // it and puts the list address back.
    nav(list + "/new"); wait('!!document.querySelector("[role=dialog]")');
    // A phone's sheet slides up: audit it once it has settled, not mid-way.
    wait('(() => { const d = document.querySelector("[role=dialog]"); return !!d && d.getBoundingClientRect().bottom <= innerHeight + 1 && !d.getAnimations({ subtree: true }).some(a => a.playState === "running"); })()');
    await capture(app + "-add", true);
    ab("press", "Escape");
    wait(`location.hash===${JSON.stringify(list)} && !document.querySelector('[role=dialog]')`);
    results.push(app + ": /new opens Add provider and Escape returns to the list");

    // Pi's custom endpoint is a page, not a dialog step.
    nav(list + "/custom"); wait('[...document.querySelectorAll("#cli-providers-view h1, #cli-providers-view h2, #cli-providers-view h3")].some(h=>h.textContent.trim()==="Custom provider") && !document.querySelector("#cli-providers-view .cred-loading")');
    await capture(app + "-custom");
    results.push(app + ": /custom opens the custom provider page");

    // Blocked addresses name what is wrong and never open the editor.
    for (const [name, hash] of [["unknown", "#/clis/unknown-cli/providers"], ["malformed", "#/clis/%ZZ/providers"], ["invalid", list + "/extra"], ["scoped", list + "?agentId=" + agent.id]]) {
      nav(hash);
      wait('!!document.querySelector("#cli-providers-view .cli-notice")');
      assert.equal(ev('!!document.querySelector("#cli-credentials-view")'), false, name);
      await capture(app + "-" + name);
    }
    results.push(app + ": unknown, malformed, invalid and scoped addresses block the editor");

    // llama.cpp keeps its own manager under the old provider alias.
    nav("#/providers/llama"); wait('!!document.querySelector(".llama-nav")');
    results.push(app + ": the llama.cpp alias opens its manager");
  }
  writeFileSync(join(out, "results.json"), JSON.stringify({ ok: true, results }, null, 2));
  console.log(JSON.stringify({ ok: true, results }, null, 2));
} catch (error) {
  try { ab("screenshot", join(out, "failure.png")); } catch {}
  throw error;
} finally {
  try { ab("close"); } catch {}
}
