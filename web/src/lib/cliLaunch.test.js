import test from "node:test";
import assert from "node:assert/strict";
import { cliLocation, cliTerminals, launchDraft, launchConfig, resolveLaunch, launchOverrides, terminalLaunchCLI } from "./cliLaunch.js";
import { cliLaunchSchema, parseForm } from "./schemas.js";
import { parseRoute } from "./routes.js";
import { mobileRoute } from "./mobileRoutes.js";

test("CLI routes and old preferences reach the same manager", () => {
  for (const hash of ["#/clis", "#/clis/codex", "#/clis/terminals", "#/preferences/status"]) {
    assert.equal(parseRoute(hash), "clis"); assert.equal(mobileRoute(hash).section, "clis");
  }
  assert.deepEqual(cliLocation("#/clis/new/codex"), { view: "new", id: "codex" });
  assert.deepEqual(cliLocation("#/clis/terminal/a%20b"), { view: "terminal", id: "a b" });
});

test("launch overrides inherit untouched fields and preserve explicit clearing", () => {
  const base = { executable: "", args: ["--flag"], path: ["/base"], env: { KEEP: "one", DROP: "two" }, integration: true };
  const next = { ...base, args: [], env: { KEEP: "one", ADD: "three" }, integration: false };
  const patch = launchOverrides(base, next);
  assert.deepEqual(patch, { args: [], integration: false, env: { DROP: null, ADD: "three" } });
  assert.deepEqual(resolveLaunch(base, patch), next);
  assert.deepEqual(launchConfig(launchDraft(next)), next);
});

test("launch form rejects ambiguous or reserved environment settings", () => {
  const base = launchDraft();
  for (const envText of ["PICODE_TERM_ID=x", "PATH=/other", "HOME=/other", "bad-name=x", "NAME", "NAME=x\nNAME=y"]) {
    assert.equal(parseForm(cliLaunchSchema, { ...base, envText }).ok, false, envText);
  }
  assert.equal(parseForm(cliLaunchSchema, { ...base, envText: "NAME=a=b", pathText: "/tools with spaces" }).ok, true);
});

test("terminal inventory includes manual CLIs and saved launches, not ordinary shells", () => {
  const terms = [{ id: "manual", tui: { cli: "pi" } }, { id: "saved", launchCli: "codex", running: false }, { id: "changed", launchCli: "codex", tui: { cli: "grok" } }, { id: "shell" }];
  assert.deepEqual(cliTerminals(terms).map((t) => t.id), ["manual", "saved", "changed"]);
  assert.deepEqual(cliTerminals(terms, "grok").map((t) => t.id), ["changed"]);
  assert.deepEqual(cliTerminals(terms, "codex").map((t) => t.id), ["saved", "changed"]);
});

test("adopting a manual terminal proposes its observed CLI without replacing saved identity", () => {
  assert.equal(terminalLaunchCLI({ tui: { cli: "claude-code" } }, "terminal-id"), "claude-code");
  assert.equal(terminalLaunchCLI({ cli: "grok" }, "terminal-id"), "grok");
  assert.equal(terminalLaunchCLI({ launchCli: "codex", tui: { cli: "pi" } }), "codex");
  assert.equal(terminalLaunchCLI(null, "pi"), "pi");
});
