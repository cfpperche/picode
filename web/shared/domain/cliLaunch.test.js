import test from "node:test";
import assert from "node:assert/strict";
import { cliLocation, cliPaneHash, cliPaneSetupContext, cliCapabilities, cliPanes, cliTerminals, launchDraft, launchConfig, resolveLaunch, launchOverrides, terminalLaunchCLI, defaultLaunchConfig, profileOverrides, editLaunchOverrides, cliWorkspaceList, launchChanged, PICODE_TOOL_FAMILIES, adoptOffer } from "./cliLaunch.js";
import { cliLaunchSchema, parseForm } from "../contracts/schemas.js";

test("CLI manager parses launch routes", () => {
  assert.deepEqual(cliLocation("#/clis/new/codex"), { view: "new", id: "codex" });
  assert.deepEqual(cliLocation("#/clis/terminal/a%20b"), { view: "terminal", id: "a b" });
});

test("the retired general Terminals address is not a pane (retired 2026-09-25)", () => {
  assert.notEqual(cliLocation("#/clis/terminals").redirect, "#/clis");
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
  assert.deepEqual(cliLocation("#/clis/pi/settings"), { view: "clis", pane: "settings", id: "pi", workspaceId: "", agentId: "", focus: "", layer: "", redirect: "" });
  assert.deepEqual(cliLocation("#/clis/pi/packages"), { view: "clis", pane: "packages", id: "pi", pkg: "", workspaceId: "", agentId: "", scope: "user", invalid: false, redirect: "" });
  assert.equal(cliLocation("#/clis/pi/connectors").pane, "connectors");
  assert.equal(cliPaneHash("pi", "settings"), "#/clis/pi/settings");
  assert.equal(cliPaneHash("pi", "packages"), "#/clis/pi/packages");
  assert.equal(cliPaneHash("pi", "connectors"), "#/clis/pi/connectors");
  assert.equal(cliLocation("#/clis/pi/settings/extra").invalid, true);
  assert.equal(cliLocation("#/clis/pi/connectors/extra").invalid, true);
  assert.equal(cliLocation("#/clis/pi/keyboard").pane, "keyboard");
});

test("setup panes fall back to the selected sidebar pane when the hash has no identity", () => {
  const legacy = { workspaceId: "w", agentId: "a" };
  assert.deepEqual(cliPaneSetupContext({ pane: "packages" }, legacy), { workspaceId: "w", agentId: "a", scope: "", focus: "", layer: "" });
  // The context carries the shared scope word the address named (scopes.js).
  assert.deepEqual(cliPaneSetupContext({ workspaceId: "x", agentId: "y", scope: "project", scopeKind: "workspace", focus: "scoped-models" }, legacy), { workspaceId: "x", agentId: "y", scope: "workspace", focus: "scoped-models", layer: "project" });
  assert.deepEqual(cliPaneSetupContext({}, {}), { workspaceId: "", agentId: "", scope: "", focus: "", layer: "" });
});

test("retired package and connector addresses no longer name Pi's pane", () => {
  for (const hash of ["#/packages", "#/mcps", "#/integrations", "#/clis/packages", "#/clis/connectors"]) {
    assert.notEqual(cliLocation(hash).pane, "packages", hash);
    assert.notEqual(cliLocation(hash).pane, "connectors", hash);
  }
});

test("Sessions live on the CLI's pane; the Pi-era addresses are retired (ADR-0079)", () => {
  assert.deepEqual(cliLocation("#/clis/codex/sessions/ws-9"), { view: "clis", id: "codex", pane: "sessions", workspace: "ws-9" });
  // #/sessions* and #/clis/sessions* stopped naming Pi's pane on 2026-09-25.
  for (const hash of ["#/sessions", "#/sessions/ws-9", "#/clis/sessions", "#/clis/sessions?cli=codex"]) {
    assert.notEqual(cliLocation(hash).pane, "sessions", hash);
  }
});

test("launch overrides inherit untouched fields and preserve explicit clearing", () => {
  const base = { executable: "", args: ["--flag"], path: ["/base"], env: { KEEP: "one", DROP: "two" }, integration: true, tools: [] };
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
  // Memory joined the setup group with ADR-0163: every CLI carries the tab,
  // and the ones with no native memory answer it in one line.
  assert.deepEqual(cliPanes({ id: "muse", integrationCapable: false, launchable: true, sessions: { list: true } }), ["launch", "terminals", "sessions", "providers", "settings", "keyboard", "memory", "packages", "skills", "connectors"]);
  assert.deepEqual(cliPanes({ id: "pi", integrationCapable: true, launchable: true, sessions: { list: true } }), ["launch", "terminals", "sessions", "providers", "settings", "keyboard", "memory", "packages", "skills", "connectors"]);
});

// ADR-0154: a CLI agent's per-agent scope for PiCode tools is the launch. A
// config saved before the key existed reads as no tools and never diffs.
test("launch tools ride the draft, the config and the overrides", () => {
  const base = defaultLaunchConfig(false);
  const legacy = { executable: "", args: [], path: [], env: {}, integration: false };
  assert.deepEqual(launchDraft(legacy).tools, []);
  assert.deepEqual(launchOverrides(legacy, base), {});
  const next = { ...base, tools: ["computer", "browser"] };
  assert.deepEqual(launchOverrides(base, next), { tools: ["computer", "browser"] });
  assert.deepEqual(launchConfig(launchDraft(next)), next);
  assert.deepEqual(resolveLaunch(legacy, { tools: ["browser"] }).tools, ["browser"]);
  assert.deepEqual(profileOverrides(base, next).tools, ["computer", "browser"]);
  assert.deepEqual(editLaunchOverrides(base, { tools: ["computer"] }, { ...base, tools: [] }), { tools: [] });
  assert.equal(parseForm(cliLaunchSchema, { ...launchDraft(base), tools: ["Computer Use"] }).ok, false);
  assert.equal(parseForm(cliLaunchSchema, { ...launchDraft(base), tools: ["computer"] }).ok, true);
  assert.deepEqual(parseForm(cliLaunchSchema, launchDraft(legacy)).value.tools, []);
});

// ADR-0154: the form is what a person can switch on, so a family the daemon
// serves but the form omits is reachable only by hand — which is how
// delivery (ADR-0171) shipped. TestToolFamiliesMatchTheForm holds the Go
// side of this list, in order.
test("the PiCode tools form offers every family the daemon serves", () => {
  assert.deepEqual(PICODE_TOOL_FAMILIES.map((f) => f.id), ["computer", "browser", "inbox", "checklist", "delivery", "mission"]);
  const ids = PICODE_TOOL_FAMILIES.map((f) => f.id);
  const parsed = parseForm(cliLaunchSchema, { ...launchDraft(defaultLaunchConfig(false)), tools: ids });
  assert.equal(parsed.ok, true);
  assert.deepEqual(parsed.value.tools, ids);
});

// ADR-0181: only a CLI PiCode can ask for its catalog carries a Models pane.
test("the Models pane is offered where the CLI answers, and its address round-trips", async () => {
  const { cliPanes, cliLocation, cliModelsHash } = await import("./cliLaunch.js");
  const omp = { id: "omp", launchable: true, integrationCapable: true, sessions: { list: true } };
  const codex = { id: "codex", launchable: true, integrationCapable: true, sessions: { list: true } };
  assert.ok(cliPanes(omp).includes("models"));
  assert.equal(cliPanes(omp).indexOf("models"), cliPanes(omp).indexOf("providers") + 1);
  assert.equal(cliPanes(codex).includes("models"), false);
  const hash = cliModelsHash("omp", { workspaceId: "w1", layer: "project" });
  assert.equal(hash, "#/clis/omp/models?workspaceId=w1&scope=workspace");
  const loc = cliLocation(hash);
  assert.equal(loc.pane, "models");
  assert.equal(loc.workspaceId, "w1");
  assert.equal(loc.layer, "project");
  assert.equal(cliLocation("#/clis/omp/models/extra").invalid, true);
  assert.equal(cliLocation("#/clis/omp/models?layer=elsewhere").layer, "");
});

test("adoptOffer: only a shell running a detected CLI, not yet an agent (ADR-0184)", () => {
  const shell = { id: "t", tui: { cli: "claude-code" } };
  assert.equal(adoptOffer(shell, null), "claude-code");
  assert.equal(adoptOffer({ id: "t" }, null), "");
  assert.equal(adoptOffer(shell, { id: "a" }), "");
  assert.equal(adoptOffer({ ...shell, launchCli: "claude-code" }, null), "");
  assert.equal(adoptOffer({ ...shell, kind: "signin" }, null), "");
  assert.equal(adoptOffer(null, null), "");
});

// One scope vocabulary for every setup pane's address (scopes.js): each pane
// writes global/workspace/agent, reads its own older words as aliases and
// rewrites such a link; a scope a pane lacks is invalid or dropped, never
// adopted as another.
import { readScope, writeScope, PACKAGE_WORDS, SKILL_WORDS } from "./scopes.js";

test("every setup pane writes and reads the shared scope words", () => {
  const rows = [
    // [hash, pane's own scope, redirect]
    ["#/clis/pi/packages?scope=workspace", "project", ""],
    ["#/clis/pi/packages?scope=project", "project", "#/clis/pi/packages?scope=workspace"],
    ["#/clis/pi/connectors?scope=agent", "agent", ""],
    ["#/clis/pi/connectors?scope=user", "user", "#/clis/pi/connectors"],
    ["#/clis/pi/skills?scope=global", "machine", ""],
    ["#/clis/pi/skills?scope=machine", "machine", "#/clis/pi/skills"],
    ["#/clis/pi/memory?scope=workspace", "workspace", undefined],
  ];
  for (const [hash, scope, redirect] of rows) {
    const loc = cliLocation(hash);
    assert.equal(loc.invalid || false, false, hash);
    assert.equal(loc.scope, scope, hash);
    if (redirect !== undefined) assert.equal(loc.redirect || "", redirect, hash);
  }
  assert.equal(cliLocation("#/clis/pi/settings?scope=agent").layer, "agent");
  assert.equal(cliLocation("#/clis/pi/settings?layer=project").redirect, "#/clis/pi/settings?scope=workspace");
  assert.equal(cliLocation("#/clis/omp/models?layer=global").redirect, "#/clis/omp/models?scope=global");
  assert.equal(cliLocation("#/clis/pi/packages?scope=bogus").invalid, true);
  assert.equal(readScope(new URLSearchParams("scope=agent"), PACKAGE_WORDS).value, "agent");
  assert.equal(readScope(new URLSearchParams("scope=agent"), { global: "g" }).invalid, true);
  const q = new URLSearchParams();
  writeScope(q, "project");
  assert.equal(q.toString(), "scope=workspace");
  writeScope(q, "machine");
  assert.equal(q.get("scope"), "workspace", "Global is left out unless kept");
  assert.equal(SKILL_WORDS.global, "machine");
});

test("a tab link carries the scope the address chose into the next pane's words", () => {
  const from = cliLocation("#/clis/claude-code/skills?workspaceId=w&scope=workspace");
  const ctx = cliPaneSetupContext(from, {});
  assert.equal(ctx.scope, "workspace");
  const fromDefault = cliPaneSetupContext(cliLocation("#/clis/pi/packages"), {});
  assert.equal(fromDefault.scope, "", "a pane's default is not carried as if chosen");
});
