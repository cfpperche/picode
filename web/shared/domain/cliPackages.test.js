import assert from "node:assert/strict";
import { test } from "node:test";
import { cliPackagesHash, cliPackagesLocation, supportsCliPackages, loadPiPackagesContext, packageContextKey, GUEST_PACKAGES, usesGuestPackages, guestPackagesApi, guestPackagesNotes, catalogRowAction, refusalCommand } from "./cliPackages.js";
import { cliLocation } from "./cliLaunch.js";

test("canonical links round-trip CLI, package, scope and explicit context", () => {
  const context = { workspaceId: "w /&", agentId: "a /?", scope: "agent", pkg: "@scope/roles" };
  const route = cliPackagesLocation(cliPackagesHash("pi", context));
  for (const [key, value] of Object.entries(context)) assert.equal(route[key], value);
  assert.equal(route.legacy, false);
  assert.equal(route.redirect, "");
  assert.equal(cliLocation(cliPackagesHash("pi", context)).pane, "packages");
});

test("legacy links adopt a pane only when they have no explicit context", () => {
  for (const prefix of ["#/packages", "#/more/packages"]) {
    const route = cliPackagesLocation(prefix + "/config/pi-roles", { workspaceId: "w", agentId: "a" });
    assert.equal(route.adoptPane, true);
    assert.equal(route.redirect, "#/clis/pi/packages/config/pi-roles?workspaceId=w&agentId=a");
    assert.equal(cliPackagesLocation(prefix + "?workspaceId=x", { agentId: "a" }).agentId, "");
    assert.equal(cliPackagesLocation(prefix + "?agentId=", { agentId: "a" }).agentId, "");
  }
  assert.equal(cliPackagesLocation("#/clis/pi/packages", { workspaceId: "w", agentId: "a" }).adoptPane, undefined);
});

test("canonical machine links never inherit the current pane", () => {
  const route = cliPackagesLocation("#/clis/pi/packages", { workspaceId: "w", agentId: "a" });
  assert.equal(route.workspaceId, ""); assert.equal(route.agentId, "");
  assert.equal(cliPackagesLocation("#/clis/packages").redirect, "#/clis/pi/packages");
  assert.equal(cliPackagesLocation("#/clis/packages/pi").redirect, "#/clis/pi/packages");
  assert.equal(cliPackagesLocation("#/providers"), null);
});

test("unsupported CLIs and malformed links never become Pi package targets", () => {
  assert.equal(supportsCliPackages("pi"), true);
  assert.equal(supportsCliPackages("codex"), false);
  assert.equal(cliPackagesLocation("#/clis/codex/packages").id, "codex");
  for (const hash of ["#/packages/nope", "#/clis/pi/packages?scope=everyone"]) {
    assert.equal(cliPackagesLocation(hash).invalid, true, hash);
  }
  assert.equal(cliLocation("#/clis/pi/packages/config/").invalid, true);
});

const rows = [{ id: "w", name: "Workspace", agents: [{ id: "a", workspaceId: "w" }] }, { id: "other", agents: [] }];
const request = async path => {
  if (path === "/api/workspaces") return rows;
  if (path === "/api/agents?free=1") return [{ id: "free", workspaceId: "ws_free" }];
  throw new Error("Unexpected API: " + path);
};

test("machine packages need no native settings, fleet or terminal reads", async () => {
  assert.deepEqual(await loadPiPackagesContext({}, () => assert.fail("unexpected API")), { workspace: null, agent: null });
});

test("workspace and attached agent targets resolve independently of the selected pane", async () => {
  for (const input of [{ workspaceId: "w" }, { agentId: "a" }, { workspaceId: "w", agentId: "a", scope: "agent" }]) {
    const result = await loadPiPackagesContext(input, request);
    assert.equal(result.workspace.id, "w");
    assert.equal(result.agent?.id || "", input.agentId || "");
  }
  const free = await loadPiPackagesContext({ agentId: "free", scope: "agent" }, request);
  assert.equal(free.workspace, null); assert.equal(free.agent.id, "free");
});

test("missing identities and incompatible scopes block instead of falling back", async () => {
  for (const input of [{ workspaceId: "missing" }, { agentId: "missing" }, { workspaceId: "other", agentId: "a" },
    { workspaceId: "w", agentId: "free" }, { agentId: "free", scope: "project" }, { scope: "agent" }, { scope: "project" }, { scope: "invalid" }]) {
    await assert.rejects(loadPiPackagesContext(input, request), error => error.status === 404, JSON.stringify(input));
  }
});

test("transient read failure stays distinguishable from a deleted target", async () => {
  const error = Object.assign(new Error("offline"), { status: 503 });
  await assert.rejects(loadPiPackagesContext({ agentId: "a" }, async () => { throw error; }), err => err === error);
});

test("draft targets ignore display changes but include workspace and agent file locations", () => {
  const value = { workspace: { id: "w", path: "/w", name: "Old" }, agent: { id: "a", workPath: "/w/tree", mode: "stopped" } };
  assert.equal(packageContextKey(value), packageContextKey({ workspace: { ...value.workspace, name: "New" }, agent: { ...value.agent, mode: "managed" } }));
  assert.notEqual(packageContextKey(value), packageContextKey({ ...value, workspace: { ...value.workspace, path: "/other" } }));
  assert.notEqual(packageContextKey(value), packageContextKey({ ...value, agent: { ...value.agent, workPath: "/other" } }));
});

// --- guest CLI plugins (ADR-0167) --------------------------------------------
//
// The guest pane's own contract, tested where a browser is not: the eight ids
// that route to it, the request each control sends, and the copy shown where a
// verb is absent. No snapshots — shapes and sentences are asserted directly.

test("the guest list is the eight declared CLIs and never pi", () => {
  assert.deepEqual(GUEST_PACKAGES, ["claude-code", "codex", "grok", "hermes", "opencode", "muse", "agy", "omp"]);
  for (const id of GUEST_PACKAGES) {
    assert.equal(usesGuestPackages(id), true, id);
    assert.equal(supportsCliPackages(id), false, id);
  }
  assert.equal(usesGuestPackages("pi"), false);
  assert.equal(usesGuestPackages(""), false);
  assert.equal(usesGuestPackages(undefined), false);
});

test("a guest read carries the CLI, the workspace and the scope as a query", () => {
  const api = guestPackagesApi("claude-code", { workspaceId: "w /&", scope: "project" });
  assert.deepEqual(api.roster(), { method: "GET", path: "/api/cli-packages?cli=claude-code&workspace=w+%2F%26&scope=project", body: null });
  assert.equal(api.roster({ refresh: true }).path, "/api/cli-packages?cli=claude-code&workspace=w+%2F%26&scope=project&refresh=1");
  assert.equal(api.available().path, "/api/cli-packages/available?cli=claude-code&workspace=w+%2F%26&scope=project");
  assert.equal(api.marketplaces().path, "/api/cli-packages/marketplaces?cli=claude-code&workspace=w+%2F%26&scope=project");
  assert.equal(api.marketplaces().method, "GET");
  assert.equal(api.marketplaces().body, null);
  // The machine scope is the contract default: it stays out of the query and
  // the body, and a bare CLI still reads the machine roster.
  const machine = guestPackagesApi("grok");
  assert.equal(machine.roster().path, "/api/cli-packages?cli=grok");
  assert.equal(machine.marketplaces().path, "/api/cli-packages/marketplaces?cli=grok");
  assert.deepEqual(machine.install({ source: "owner/repo" }).body, { cli: "grok", scope: "user", source: "owner/repo" });
});

test("a guest mutation names its target and keeps one request key across a retry", () => {
  const api = guestPackagesApi("muse", { workspaceId: "w", scope: "project" });
  assert.deepEqual(api.install({ source: "plugins/demo", requestKey: "k1" }), {
    method: "POST",
    path: "/api/cli-packages/install",
    body: { cli: "muse", scope: "project", workspace: "w", source: "plugins/demo", requestKey: "k1" },
  });
  assert.deepEqual(api.remove({ name: "demo", source: "plugins/demo", requestKey: "k1" }).body,
    { cli: "muse", scope: "project", workspace: "w", name: "demo", source: "plugins/demo", requestKey: "k1" });
  assert.deepEqual(api.update({ name: "demo", confirmTerminals: true }).body,
    { cli: "muse", scope: "project", workspace: "w", name: "demo", confirmTerminals: true });
  assert.equal(api.toggle({ name: "demo", source: "plugins/demo" }, false).body.on, false);
  assert.equal(api.toggle({ name: "demo" }, true).body.on, true);
  assert.deepEqual(api.inspect({ name: "demo" }), { method: "POST", path: "/api/cli-packages/inspect", body: { cli: "muse", target: "demo" } });
});

test("a marketplace action passes the scope as its ref, and never for the machine scope", () => {
  assert.deepEqual(guestPackagesApi("claude-code", { workspaceId: "w", scope: "project" })
    .marketplace("add", { source: "owner/repo", name: "team" }).body,
    { cli: "claude-code", scope: "project", workspace: "w", source: "owner/repo", name: "team", action: "add", ref: "project" });
  assert.deepEqual(guestPackagesApi("codex").marketplace("remove", { name: "openai-curated" }).body,
    { cli: "codex", scope: "user", action: "remove", name: "openai-curated" });
  assert.equal(guestPackagesApi("omp", { scope: "project", workspaceId: "w" })
    .marketplace("update", { name: "core", ref: "main" }).body.ref, "main");
});

test("absence notes speak once per missing verb, then for the concepts left over", () => {
  // Codex: install and remove exist, enable/disable and update do not. The two
  // vendor words for the toggle are one sentence, not two.
  assert.deepEqual(
    guestPackagesNotes({ install: true, remove: true }, { enable: "No toggle.", disable: "No toggle.", update: "No update." }),
    ["No toggle.", "No update."],
  );
  // A pane renders before its first report: `report && report.notes` hands
  // over null, and a default parameter does not cover null. Regression for the
  // boot crash this shipped with (guest packages route, 2026-09-20).
  assert.deepEqual(guestPackagesNotes(null, null), []);
  assert.deepEqual(guestPackagesNotes(undefined, undefined), []);
  // Hermes: a catalog without marketplace sources, plus a concept note.
  assert.deepEqual(
    guestPackagesNotes({ install: true, remove: true, toggle: true, update: true, available: true }, {
      "marketplace-add": "No sources.",
      capabilities: "Capabilities are granted per plugin.",
    }),
    ["No sources.", "Capabilities are granted per plugin."],
  );
  // A verb that exists keeps its note out of the pane; a note with no missing
  // verb behind it is still one sentence of chrome.
  assert.deepEqual(guestPackagesNotes({ update: true }, { update: "Never shown." }), []);
  assert.deepEqual(guestPackagesNotes({ install: true }, { app: "Marketplaces can run a command." }), ["Marketplaces can run a command."]);
  // A CLI with every verb and no notes says nothing.
  assert.deepEqual(guestPackagesNotes({ install: true, remove: true, toggle: true, update: true, inspect: true, marketplace: true }, {}), []);
});

test("a note missing for an absent verb invents nothing", () => {
  assert.deepEqual(guestPackagesNotes({ install: true, remove: true }, {}), []);
  assert.deepEqual(guestPackagesNotes({}, { install: "" }), []);
});

test("a catalog row offers Install only when its CLI names the spec", () => {
  // Muse: the row carries name@marketplace, so the pane may install from it.
  assert.equal(catalogRowAction({ catalogInstall: true }, { source: "picode-spare@mp" }), "install");
  assert.equal(catalogRowAction({ catalogInstall: true }, { installed: true, source: "x@mp" }), "installed");
  // Omp: discover prints a name and a version and never the marketplace, so an
  // install from the row would run `omp plugin install <name>` — which resolves
  // through npm. The row stays information.
  assert.equal(catalogRowAction({ catalogInstall: false }, { source: "" }), "none");
  assert.equal(catalogRowAction({}, { source: "x@mp" }), "none");
});

test("a refusal carries the command a person has to run", () => {
  const ex = Object.assign(new Error("grok plugin install failed: refusing without --trust"), {
    status: 409,
    body: { error: "grok plugin install failed: refusing without --trust", command: "grok plugin install owner/repo" },
  });
  assert.deepEqual(refusalCommand(ex), {
    message: "grok plugin install failed: refusing without --trust",
    command: "grok plugin install owner/repo",
  });
  assert.deepEqual(refusalCommand(new Error("plain")), { message: "plain", command: "" });
  assert.deepEqual(refusalCommand(null), { message: "", command: "" });
});
