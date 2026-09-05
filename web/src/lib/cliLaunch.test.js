import test from "node:test";
import assert from "node:assert/strict";
import { cliLocation, cliTerminals, launchDraft, launchConfig, resolveLaunch, launchOverrides, terminalLaunchCLI, defaultLaunchConfig, profileOverrides, editLaunchOverrides, cliWorkspaceList, launchChanged } from "./cliLaunch.js";
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

test("restoring defaults retains reporting choice and automatic executable", () => {
  for (const integration of [true, false]) {
    const c = defaultLaunchConfig(integration);
    assert.equal(c.executable, ""); assert.equal(c.integration, integration);
    assert.deepEqual(launchConfig(launchDraft(c)), c);
  }
});

test("profile copies preserve empty pins and ignore later profile edits", () => {
  const base = { ...defaultLaunchConfig(true), args: ["base"], env: { DROP: "x" } };
  const profile = { ...defaultLaunchConfig(false), env: { ADD: "secret" } };
  const overrides = profileOverrides(base, profile);
  profile.args.push("changed"); profile.env.ADD = "new";
  assert.deepEqual(resolveLaunch(base, overrides), { ...defaultLaunchConfig(false), env: { ADD: "secret" } });
  assert.deepEqual(editLaunchOverrides(base, overrides, resolveLaunch(base, overrides)), overrides);
});

test("editing an unrelated field retains an override equal to current defaults", () => {
  const base = defaultLaunchConfig(false), previous = { args: [], integration: false };
  assert.deepEqual(editLaunchOverrides(base, previous, { ...base, executable: "/bin/pi" }), { args: [], integration: false, executable: "/bin/pi" });
});

test("argument editor preserves empty and literal quoted arguments", () => {
  const c = { ...defaultLaunchConfig(false), args: ["", "  ", '"quoted"', "'single'", "$literal", "two words"] };
  assert.deepEqual(launchConfig(launchDraft(c)), c);
});

test("profile and workspace routes carry launch context", () => {
  assert.deepEqual(cliLocation("#/clis/new/pi?profile=review&workspace=project"), { view: "new", id: "pi", profile: "review", workspace: "project" });
  assert.deepEqual(cliLocation("#/clis/profile/new/pi"), { view: "profile", id: "new", cli: "pi" });
});

test("workspace picker accepts the API's direct array response", () => {
  const rows = [{ id: "project", name: "Project" }];
  assert.deepEqual(cliWorkspaceList(rows), rows);
  assert.deepEqual(cliWorkspaceList({ workspaces: rows }), rows);
  assert.deepEqual(cliWorkspaceList(null), []);
});

test("launch comparison detects binary replacement without marking legacy snapshots stale", () => {
  const applied = { cli: "pi", fingerprint: "config", executable: "/bin/pi", identity: "old" };
  assert.equal(launchChanged(applied, { ...applied }), false);
  for (const patch of [{ cli: "codex" }, { fingerprint: "changed" }, { executable: "/other/pi" }, { identity: "replaced" }]) {
    assert.equal(launchChanged(applied, { ...applied, ...patch }), true);
  }
  assert.equal(launchChanged({ ...applied, identity: "" }, applied), false);
});
