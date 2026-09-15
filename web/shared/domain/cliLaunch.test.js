import test from "node:test";
import assert from "node:assert/strict";
import { cliLocation, cliPaneHash, cliPaneSetupContext, cliCapabilities, cliPanes, cliTerminals, launchDraft, launchConfig, resolveLaunch, launchOverrides, terminalLaunchCLI, defaultLaunchConfig, profileOverrides, editLaunchOverrides, cliWorkspaceList, launchChanged } from "./cliLaunch.js";
import { cliLaunchSchema, parseForm } from "../contracts/schemas.js";

test("CLI manager parses launch routes", () => {
  assert.deepEqual(cliLocation("#/clis/new/codex"), { view: "new", id: "codex" });
  assert.deepEqual(cliLocation("#/clis/terminal/a%20b"), { view: "terminal", id: "a b" });
});

test("the general Terminals address lands on the CLI catalog (2026-09-11)", () => {
  assert.deepEqual(cliLocation("#/clis/terminals"), { view: "clis", id: "", pane: "launch", redirect: "#/clis" });
});

test("a CLI page names its pane in the path (ADR-0079 amendment 2026-09-11)", () => {
  assert.deepEqual(cliLocation("#/clis"), { view: "clis", id: "", pane: "launch" });
  assert.deepEqual(cliLocation("#/clis/pi"), { view: "clis", id: "pi", pane: "launch" });
  assert.deepEqual(cliLocation("#/clis/codex/terminals"), { view: "clis", id: "codex", pane: "terminals" });
  assert.deepEqual(cliLocation("#/clis/pi/sessions"), { view: "clis", id: "pi", pane: "sessions" });
  assert.deepEqual(cliLocation("#/clis/claude-code/sessions/ws-9"), { view: "clis", id: "claude-code", pane: "sessions", workspace: "ws-9" });
  assert.deepEqual(cliLocation("#/clis/pi/launch"), { view: "clis", id: "pi", pane: "launch", redirect: "#/clis/pi" });
  assert.equal(cliPaneHash("pi"), "#/clis/pi");
  assert.equal(cliPaneHash("pi", "launch"), "#/clis/pi");
  assert.equal(cliPaneHash("codex", "terminals"), "#/clis/codex/terminals");
  assert.equal(cliPaneHash("pi", "sessions"), "#/clis/pi/sessions");
  assert.equal(cliPaneHash("claude-code", "sessions", "ws-9"), "#/clis/claude-code/sessions/ws-9");
  assert.deepEqual(cliLocation("#/clis/pi/providers"), { view: "clis", id: "pi", pane: "providers" });
  assert.deepEqual(cliLocation("#/clis/pi/providers/new"), { view: "clis", id: "pi", pane: "providers", add: true });
  assert.deepEqual(cliLocation("#/clis/codex/providers"), { view: "clis", id: "codex", pane: "providers" });
  assert.deepEqual(cliLocation("#/clis/pi/providers/custom"), { view: "clis", id: "pi", pane: "providers", custom: "new" });
  assert.deepEqual(cliLocation("#/clis/pi/providers/custom/cheap"), { view: "clis", id: "pi", pane: "providers", custom: "edit", customId: "cheap" });
  assert.deepEqual(cliLocation("#/clis/pi/providers/custom/a%20b"), { view: "clis", id: "pi", pane: "providers", custom: "edit", customId: "a b" });
  assert.equal(cliLocation("#/clis/pi/providers/custom/cheap/extra").invalid, true);
  assert.equal(cliLocation("#/clis/pi/providers/nope").invalid, true);
  assert.equal(cliPaneHash("pi", "providers"), "#/clis/pi/providers");
  assert.deepEqual(cliLocation("#/clis/pi/settings"), { view: "clis", pane: "settings", id: "pi", agentId: "", focus: "", layer: "", keysTab: false, legacy: false, redirect: "" });
  assert.deepEqual(cliLocation("#/clis/pi/packages"), { view: "clis", pane: "packages", id: "pi", pkg: "", workspaceId: "", agentId: "", scope: "user", legacy: false, invalid: false, redirect: "" });
  assert.equal(cliLocation("#/clis/pi/connectors").pane, "connectors");
  assert.equal(cliPaneHash("pi", "settings"), "#/clis/pi/settings");
  assert.equal(cliPaneHash("pi", "packages"), "#/clis/pi/packages");
  assert.equal(cliPaneHash("pi", "connectors"), "#/clis/pi/connectors");
  assert.equal(cliLocation("#/clis/connectors").pane, "connectors");
  assert.equal(cliLocation("#/clis/connectors").id, "pi");
  assert.equal(cliLocation("#/clis/connectors").redirect, "#/clis/pi/connectors");
  assert.equal(cliLocation("#/clis/pi/settings/extra").invalid, true);
  assert.equal(cliLocation("#/clis/pi/connectors/extra").invalid, true);
  assert.equal(cliLocation("#/clis/pi/keyboard").pane, "keyboard");
});

test("setup panes fall back to the selected sidebar pane when the hash has no identity", () => {
  const legacy = { workspaceId: "w", agentId: "a" };
  assert.deepEqual(cliPaneSetupContext({ pane: "packages" }, legacy), { workspaceId: "w", agentId: "a", scope: "user", focus: "", layer: "" });
  assert.deepEqual(cliPaneSetupContext({ workspaceId: "x", agentId: "y", scope: "project", focus: "scoped-models", layer: "agent" }, legacy), { workspaceId: "x", agentId: "y", scope: "project", focus: "scoped-models", layer: "agent" });
  assert.deepEqual(cliPaneSetupContext({}, {}), { workspaceId: "", agentId: "", scope: "user", focus: "", layer: "" });
});

test("legacy package and settings hashes adopt pane context through cliLocation", () => {
  const packages = cliLocation("#/packages", { packageContext: { workspaceId: "w", agentId: "a" }, agentId: "a" });
  assert.equal(packages.adoptPane, true);
  assert.equal(packages.redirect, "#/clis/pi/packages?workspaceId=w&agentId=a");
  const settings = cliLocation("#/settings", { agentId: "a" });
  assert.equal(settings.adoptPane, true);
  assert.equal(settings.redirect, "#/clis/pi/settings?agentId=a");
  assert.equal(cliLocation("#/packages").redirect, "#/clis/pi/packages");
  const connectors = cliLocation("#/mcps", { packageContext: { workspaceId: "w", agentId: "a" } });
  assert.equal(connectors.pane, "connectors");
  assert.equal(connectors.redirect, "#/clis/pi/connectors?workspaceId=w&agentId=a");
});

test("legacy Sessions tab addresses rewrite onto the selected CLI's pane (ADR-0079)", () => {
  assert.deepEqual(cliLocation("#/clis/sessions"), { view: "clis", id: "pi", pane: "sessions", redirect: "#/clis/pi/sessions" });
  assert.deepEqual(cliLocation("#/clis/sessions/ws-9"), { view: "clis", id: "pi", pane: "sessions", workspace: "ws-9", redirect: "#/clis/pi/sessions/ws-9" });
  assert.deepEqual(cliLocation("#/clis/sessions?cli=codex"), { view: "clis", id: "codex", pane: "sessions", redirect: "#/clis/codex/sessions" });
  assert.deepEqual(cliLocation("#/clis/sessions/ws-9?cli=claude-code"), { view: "clis", id: "claude-code", pane: "sessions", workspace: "ws-9", redirect: "#/clis/claude-code/sessions/ws-9" });
  assert.deepEqual(cliLocation("#/sessions"), { view: "clis", id: "pi", pane: "sessions", redirect: "#/clis/pi/sessions" });
  assert.deepEqual(cliLocation("#/sessions/ws-9"), { view: "clis", id: "pi", pane: "sessions", workspace: "ws-9", redirect: "#/clis/pi/sessions/ws-9" });
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
  for (const envText of ["PICODE_TERM_ID=x", "PATH=/other", "HOME=/other", "GROK_HOME=/other", "HERMES_HOME=/other", "bad-name=x", "NAME", "NAME=x\nNAME=y"]) {
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

test("catalog capabilities decide what a CLI's surface shows", () => {
  // A full row keeps every pane; Muse Code and Antigravity open a terminal
  // with no adapter, so they get Launch and Terminals only.
  assert.deepEqual(cliCapabilities({ id: "pi", integrationCapable: true, launchable: true, sessions: { list: true } }), { launch: true, integration: true, sessions: true });
  assert.deepEqual(cliCapabilities({ id: "muse", integrationCapable: false, launchable: true, sessions: { list: false } }), { launch: true, integration: false, sessions: false });
  assert.deepEqual(cliCapabilities(null), { launch: false, integration: false, sessions: false });
  assert.deepEqual(cliPanes({ id: "muse", integrationCapable: false, launchable: true }), ["launch", "terminals"]);
  assert.deepEqual(cliPanes({ id: "pi", integrationCapable: true, launchable: true, sessions: { list: true } }), ["launch", "terminals", "sessions", "providers", "settings", "keyboard", "packages", "connectors"]);
});
