import assert from "node:assert/strict";
import { test } from "node:test";
import { cliPackagesHash, cliPackagesLocation, loadPiPackagesContext, packageContextKey, packagesApi, packagesNotes, packagesSurface, paneWords, PANE_WORDS, behindFor, catalogRowAction, refusalCommand, matchParts, groupInstalledRows, sourceGroupKey, directMutation, laneMutation, anyLaneMutation, paneTabs, rowToggle, rowInspect } from "./cliPackages.js";
import { cliLocation } from "./cliLaunch.js";

// The transport declarations these rows re-table, as a report carries them
// (`Caps.Lane`): Pi's, a guest's, and the mixed CLI's — OpenCode, whose install
// is its own `plugin` command and whose removal is a write of its own config
// file. Every `{ async: true }` row below became GUEST_LANE, every
// `{ async: false }` row PI_LANE.
const PI_LANE = { install: false, remove: false, update: false, marketplace: false };
const GUEST_LANE = { install: true, remove: true, update: true, marketplace: true };
const MIXED_LANE = { install: true, remove: false, update: false, marketplace: false };

// The transport rule: which mutation is one of PiCode's own calls and which is
// a verb the CLI's own surface takes. A wrong answer here runs a vendor's
// command for a layer the vendor does not own, or reserves a job that can never
// carry it out.
test("the agent scope is PiCode's own write for every CLI", () => {
  // A CLI whose mutations are all its own calls (Pi's own pipkg, nothing on the
  // lane) is always direct.
  assert.equal(directMutation({ lane: PI_LANE }, {}), true);
  assert.equal(directMutation({ lane: PI_LANE }, { scope: "project" }), true);
  // A vendor's own layers are the vendor's route — whether the declaration puts
  // the verb on the lane or, like the mixed CLI's removal, makes it a write.
  assert.equal(directMutation({ lane: GUEST_LANE }, { scope: "user" }), false);
  assert.equal(directMutation({ lane: GUEST_LANE }, { scope: "project" }), false);
  assert.equal(directMutation({ lane: MIXED_LANE }, { scope: "user" }), false);
  assert.equal(directMutation({ lane: MIXED_LANE }, { row: { scope: "workspace" } }), false);
  // The agent layer is PiCode's list on the agent row, so writing it is direct
  // whichever CLI the pane is showing...
  assert.equal(directMutation({ lane: GUEST_LANE }, { scope: "agent" }), true);
  assert.equal(directMutation({ lane: MIXED_LANE }, { scope: "agent" }), true);
  // ...and so is a row that lives there, whatever the pane's own scope is.
  assert.equal(directMutation({ lane: GUEST_LANE }, { scope: "agent", row: { scope: "agent" } }), true);
  assert.equal(directMutation({ lane: GUEST_LANE }, { scope: "user", row: { scope: "agent" } }), true);
  assert.equal(directMutation({ lane: GUEST_LANE }, { row: { scope: "workspace" } }), false);
  // Before a report arrives nothing may be sent to a lane on a guess.
  assert.equal(directMutation(null, { scope: "user" }), true);
  assert.equal(directMutation(undefined, { scope: "project" }), true);
});

// The per-verb half of that declaration: a mixed CLI answers for each verb on
// its own, which is what a removal read as PiCode's own write depends on.
test("the transport declaration is read one verb at a time", () => {
  assert.equal(laneMutation({ lane: GUEST_LANE }, "remove"), true);
  assert.equal(laneMutation({ lane: MIXED_LANE }, "install"), true);
  assert.equal(laneMutation({ lane: MIXED_LANE }, "remove"), false);
  assert.equal(laneMutation({ lane: PI_LANE }, "install"), false);
  assert.equal(laneMutation(undefined, "remove"), false);
  // A verb the lane never carries is false however the declaration is spelled.
  assert.equal(laneMutation({ lane: GUEST_LANE }, "toggle"), false);
  // The lane subscription is the aggregate the single `Async` bool answered: a
  // CLI with any verb on the lane follows its events, and Pi follows none.
  assert.equal(anyLaneMutation({ lane: MIXED_LANE }), true);
  assert.equal(anyLaneMutation({ lane: PI_LANE }), false);
  assert.equal(anyLaneMutation(undefined), false);
});

// The pair of tabs is offered where there is something to switch to: a report
// with no catalog holds one list, and the agent layer has no vendor catalog —
// it is PiCode's own — while Pi's gallery stays PiCode's own at every layer.
test("the marketplace tab is offered only where a catalog exists for that layer", () => {
  assert.equal(paneTabs(null, "user"), false);
  assert.equal(paneTabs({ catalog: "" }, "user"), false);
  assert.equal(paneTabs({ catalog: "vendor" }, "user"), true);
  assert.equal(paneTabs({ catalog: "vendor" }, "project"), true);
  assert.equal(paneTabs({ catalog: "vendor" }, "agent"), false);
  assert.equal(paneTabs({ catalog: "gallery" }, "agent"), true);
});

// The on/off and inspection controls belong to the CLI's own verbs, and a row in
// PiCode's own agent list is one those commands do not know.
test("the vendor's row controls are offered only where its own verbs can act", () => {
  assert.equal(rowToggle({ toggle: true }, { scope: "user" }), true);
  assert.equal(rowToggle({ toggle: true }, { scope: "workspace" }), true);
  assert.equal(rowToggle({ toggle: true }, { scope: "agent" }), false);
  assert.equal(rowToggle({ toggle: false }, { scope: "user" }), false);
  assert.equal(rowToggle(null, { scope: "user" }), false);
  assert.equal(rowToggle({ toggle: true }, null), true);
  assert.equal(rowInspect({ inspect: true }, { scope: "user" }), true);
  assert.equal(rowInspect({ inspect: true }, { scope: "agent" }), false);
  assert.equal(rowInspect({ inspect: false }, { scope: "workspace" }), false);
  assert.equal(rowInspect(null, null), false);
});

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

test("a package link carries any CLI, and a malformed one never becomes a target", () => {
  // Which CLIs exist is the engine's answer now (the report route refuses an
  // unknown id by naming the drivers); the route only carries the id.
  assert.equal(cliPackagesLocation("#/clis/codex/packages").id, "codex");
  assert.equal(cliPackagesLocation("#/clis/pi/packages").id, "pi");
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

// --- one pane, every CLI (ADR-0176) ------------------------------------------
//
// The pane's own contract, tested where a browser is not: the request each
// control sends, the surface a report declares, and the words each surface
// speaks. No snapshots — shapes and sentences are asserted directly.

test("the surface follows the catalog the report declares, never the CLI", () => {
  assert.equal(packagesSurface({ catalog: "gallery" }), "gallery");
  assert.equal(packagesSurface({ catalog: "vendor" }), "vendor");
  // A CLI with no installable list still holds its own plugins.
  assert.equal(packagesSurface({ catalog: "" }), "vendor");
  assert.equal(packagesSurface({}), "vendor");
  assert.equal(packagesSurface(null), "vendor");
});

test("each surface keeps the words its own pane always used", () => {
  // Pi's gallery pane (ADR-0102): the words the merged pane may not lose.
  const gallery = paneWords("gallery");
  assert.equal(gallery.scopeLabel, "Install to");
  assert.equal(gallery.sourcePlaceholder, "npm:pkg  ·  git:github.com/user/repo  ·  ./path");
  assert.equal(gallery.access, "Packages run with full access. Only install what you review.");
  assert.equal(gallery.emptyTitle, "Nothing installed yet");
  assert.equal(gallery.fallback, "Machine packages");
  assert.equal(gallery.noMatch("x"), "No installed package matches \u201cx\u201d.");
  assert.equal(PANE_WORDS.gallery.isolation, "Only this agent's packages (skip machine and folder). Restart to apply.");
  // The vendors' pane (ADR-0167): "Plugins go to", its own empty state, its own
  // way back to the machine scope.
  const vendor = paneWords("vendor");
  assert.equal(vendor.scopeLabel, "Plugins go to");
  assert.equal(vendor.access, "Plugins run with full access. Only install what you review.");
  assert.equal(vendor.emptyTitle, "Nothing installed.");
  assert.equal(vendor.fallback, "Use global");
  // An unknown surface is the vendor's, never null: a pane renders before its
  // first report.
  assert.deepEqual(paneWords("nope"), PANE_WORDS.vendor);
  assert.deepEqual(paneWords(undefined), PANE_WORDS.vendor);
});

test("a read carries the CLI, the workspace, the agent and the scope as a query", () => {
  const api = packagesApi("claude-code", { workspaceId: "w /&", scope: "project" });
  // The unified report is the read every pane load starts with; its badge read
  // takes the same target, and both name the vendor word the CLI's own
  // commands take (a read of the class alone cannot say `local`).
  assert.deepEqual(api.report(), { method: "GET", path: "/api/packages/report?cli=claude-code&workspace=w+%2F%26&vendor=project", body: null });
  assert.equal(api.report({ refresh: true }).path, "/api/packages/report?cli=claude-code&workspace=w+%2F%26&vendor=project&refresh=1");
  assert.deepEqual(api.updates(), { method: "GET", path: "/api/packages/updates?cli=claude-code&workspace=w+%2F%26&vendor=project", body: null });
  assert.equal(api.available().path, "/api/packages/available?cli=claude-code&workspace=w+%2F%26&scope=project");
  assert.equal(api.marketplaces().path, "/api/packages/marketplaces?cli=claude-code&workspace=w+%2F%26&scope=project");
  assert.equal(api.marketplaces().method, "GET");
  assert.equal(api.marketplaces().body, null);
  // The machine scope is the contract default: it stays out of the query and
  // the body, and a bare CLI still reads the machine roster.
  const machine = packagesApi("grok");
  assert.equal(machine.report().path, "/api/packages/report?cli=grok");
  assert.equal(machine.updates().path, "/api/packages/updates?cli=grok");
  assert.equal(machine.marketplaces().path, "/api/packages/marketplaces?cli=grok");
  assert.deepEqual(machine.install({ source: "owner/repo" }).body, { cli: "grok", scope: "user", source: "owner/repo" });
});

test("the agent's own target rides the read, and the gallery is PiCode's own route", () => {
  const api = packagesApi("pi", { workspaceId: "w", agentId: "a", scope: "agent" });
  assert.equal(api.report().path, "/api/packages/report?cli=pi&workspace=w&agent=a&vendor=agent");
  assert.equal(api.gallery("pi (web)").path, "/api/packages/gallery?q=pi%20(web)");
  // A direct mutation is the CLI's own package API: the answer is its fresh
  // list, and the target it describes travels with the request.
  assert.deepEqual(api.direct.install({ source: "npm:x", scope: "user" }), { method: "POST", path: "/api/packages", body: { source: "npm:x", scope: "user", workspaceId: "w", agentId: "a" } });
  assert.deepEqual(api.direct.update({ source: "npm:x", scope: "project" }), { method: "POST", path: "/api/packages/update", body: { source: "npm:x", scope: "project", workspaceId: "w", agentId: "a" } });
  assert.deepEqual(api.direct.remove({ source: "npm:x", scope: "project" }), { method: "DELETE", path: "/api/packages?source=npm%3Ax&scope=project&workspace=w&agent=a", body: null });
});

test("a CLI's own mutation names it on the same paths PiCode's own calls use", () => {
  const api = packagesApi("muse", { workspaceId: "w", scope: "project" });
  assert.deepEqual(api.install({ source: "plugins/demo", requestKey: "k1" }), {
    method: "POST",
    path: "/api/packages",
    body: { cli: "muse", scope: "project", workspace: "w", source: "plugins/demo", requestKey: "k1" },
  });
  // The removal is the same verb PiCode's own call sends, with the CLI named in
  // the body instead of the source in the query.
  assert.equal(api.remove({ name: "demo", source: "plugins/demo", requestKey: "k1" }).method, "DELETE");
  assert.equal(api.remove({ name: "demo", source: "plugins/demo" }).path, "/api/packages");
  assert.deepEqual(api.remove({ name: "demo", source: "plugins/demo", requestKey: "k1" }).body,
    { cli: "muse", scope: "project", workspace: "w", name: "demo", source: "plugins/demo", requestKey: "k1" });
  assert.deepEqual(api.update({ name: "demo", confirmTerminals: true }).body,
    { cli: "muse", scope: "project", workspace: "w", name: "demo", confirmTerminals: true });
  assert.equal(api.update({ name: "demo" }).path, "/api/packages/update");
  assert.equal(api.toggle({ name: "demo", source: "plugins/demo" }, false).body.on, false);
  assert.equal(api.toggle({ name: "demo" }, true).body.on, true);
  assert.equal(api.toggle({ name: "demo" }, true).path, "/api/packages/toggle");
  assert.deepEqual(api.inspect({ name: "demo" }), { method: "POST", path: "/api/packages/inspect", body: { cli: "muse", target: "demo" } });
});

test("a marketplace action passes the scope as its ref, and never for the machine scope", () => {
  assert.deepEqual(packagesApi("claude-code", { workspaceId: "w", scope: "project" })
    .marketplace("add", { source: "owner/repo", name: "team" }).body,
    { cli: "claude-code", scope: "project", workspace: "w", source: "owner/repo", name: "team", action: "add", ref: "project" });
  assert.deepEqual(packagesApi("codex").marketplace("remove", { name: "openai-curated" }).body,
    { cli: "codex", scope: "user", action: "remove", name: "openai-curated" });
  assert.equal(packagesApi("omp", { scope: "project", workspaceId: "w" })
    .marketplace("update", { name: "core", ref: "main" }).body.ref, "main");
});

test("absence notes speak once per missing verb, then for the concepts left over", () => {
  // Codex: install and remove exist, enable/disable and update do not. The two
  // vendor words for the toggle are one sentence, not two.
  assert.deepEqual(
    packagesNotes({ install: true, remove: true }, { enable: "No toggle.", disable: "No toggle.", update: "No update." }),
    ["No toggle.", "No update."],
  );
  // A pane renders before its first report: `report && report.notes` hands
  // over null, and a default parameter does not cover null. Regression for the
  // boot crash this shipped with (guest packages route, 2026-09-20).
  assert.deepEqual(packagesNotes(null, null), []);
  assert.deepEqual(packagesNotes(undefined, undefined), []);
  // Hermes: a catalog without marketplace sources, plus a concept note.
  assert.deepEqual(
    packagesNotes({ install: true, remove: true, toggle: true, update: true, available: true }, {
      "marketplace-add": "No sources.",
      capabilities: "Capabilities are granted per plugin.",
    }),
    ["No sources.", "Capabilities are granted per plugin."],
  );
  // A verb that exists keeps its note out of the pane; a note with no missing
  // verb behind it is still one sentence of chrome.
  assert.deepEqual(packagesNotes({ update: true }, { update: "Never shown." }), []);
  assert.deepEqual(packagesNotes({ install: true }, { app: "Marketplaces can run a command." }), ["Marketplaces can run a command."]);
  // A CLI with every verb and no notes says nothing.
  assert.deepEqual(packagesNotes({ install: true, remove: true, toggle: true, update: true, inspect: true, marketplace: true }, {}), []);
});

test("a note missing for an absent verb invents nothing", () => {
  assert.deepEqual(packagesNotes({ install: true, remove: true }, {}), []);
  assert.deepEqual(packagesNotes({}, { install: "" }), []);
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

test("the availability check is one read, and refresh forces it", () => {
  const api = packagesApi("grok", { workspaceId: "w1", scope: "project" });
  assert.equal(api.updates().path, "/api/packages/updates?cli=grok&workspace=w1&vendor=project");
  assert.equal(api.updates({ refresh: true }).path, "/api/packages/updates?cli=grok&workspace=w1&vendor=project&refresh=1");
});

test("an Update control exists only for a row the CLI's catalog says is behind", () => {
  const updates = [{ source: "plugins/demo", scope: "user", current: "0.2.0", latest: "0.4.0" }];
  assert.deepEqual(behindFor(updates, { source: "plugins/demo", vendor: "user" }), updates[0]);
  // A row the check did not name is not behind — and a check that never ran
  // (an empty list) claims nothing at all.
  assert.equal(behindFor(updates, { source: "plugins/other", vendor: "user" }), null);
  assert.equal(behindFor([], { source: "plugins/demo" }), null);
  assert.equal(behindFor(undefined, { source: "plugins/demo" }), null);
  assert.equal(behindFor(null, null), null);
  // Two layers can hold the same source: the row's own layer decides.
  const both = [{ source: "x", scope: "user" }, { source: "x", scope: "project" }];
  assert.equal(behindFor(both, { source: "x", vendor: "project" }).scope, "project");
});

test("the filter's highlights are literal, case-insensitive and keep the text whole", () => {
  assert.deepEqual(matchParts("Browser Use", "use"), [
    { text: "Browser ", hit: false },
    { text: "Use", hit: true },
  ]);
  assert.deepEqual(matchParts("aXa", "x"), [
    { text: "a", hit: false }, { text: "X", hit: true }, { text: "a", hit: false },
  ]);
  // A filter with regex characters is a string, not a pattern.
  assert.deepEqual(matchParts("pi (web)", "(web"), [
    { text: "pi ", hit: false }, { text: "(web", hit: true }, { text: ")", hit: false },
  ]);
  assert.deepEqual(matchParts("anything", "  "), [{ text: "anything", hit: false }]);
  assert.deepEqual(matchParts("anything", "zzz"), [{ text: "anything", hit: false }]);
  // Always at least one part: the renderer never has to special-case an empty box.
  assert.deepEqual(matchParts("", "x"), [{ text: "", hit: false }]);
});

test("installed rows group by where they came from, and one group gets no header", () => {
  const hermes = [
    { name: "picode-native", kind: "user" },
    { name: "chronos", kind: "bundled" },
    { name: "browser-browser-use", kind: "bundled" },
  ];
  const groups = groupInstalledRows(hermes);
  assert.deepEqual(groups.map(g => [g.key, g.rows.length]), [["user", 1], ["bundled", 2]]);
  // A single-kind list is the whole list: no chrome.
  assert.deepEqual(groupInstalledRows([{ kind: "bundled" }, { kind: "bundled" }]), []);
  assert.deepEqual(groupInstalledRows([]), []);
});

test("the vendor's own provenance words decide the group", () => {
  assert.equal(sourceGroupKey({ kind: "bundled", managedByPiCode: false }), "bundled");
  assert.equal(sourceGroupKey({ kind: "synced" }), "synced");
  // Claude Code names a synced plugin's marketplace "synced" as well: the
  // vendor's own word decides, or the whole file was filed under marketplaces.
  assert.equal(sourceGroupKey({ kind: "synced", marketplace: "synced" }), "synced");
  assert.equal(sourceGroupKey({ kind: "bundled", marketplace: "hermes" }), "bundled");
  assert.equal(sourceGroupKey({ kind: "marketplace-user-added" }), "marketplace");
  assert.equal(sourceGroupKey({ kind: "local-file" }), "local");
  assert.equal(sourceGroupKey({ kind: "native-local" }), "path");
  assert.equal(sourceGroupKey({ kind: "import" }), "imported");
  assert.equal(sourceGroupKey({ kind: "npm" }), "npm");
  assert.equal(sourceGroupKey({ kind: "git" }), "git");
  // PiCode's own integration is findable without knowing the vendor's word.
  assert.equal(sourceGroupKey({ kind: "user", managedByPiCode: true }), "picode");
  // A vendor that said nothing falls back on what the row shows.
  assert.equal(sourceGroupKey({ source: "x", installedPath: "/p" }), "path");
  assert.equal(sourceGroupKey({}), "other");
});
