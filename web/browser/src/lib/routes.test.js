import assert from "node:assert/strict";
import { test } from "node:test";
import { appPath, isAgentTab, inboxHash, inboxPath, legacyInboxHash } from "./routes.js";
import { workspaceOverviewHash, workspaceOverviewRoute, workspaceAgentsHash, workspaceAgentsRoute } from "./routes.js";

test("Inbox is a core page with stable item links", () => {
  assert.equal(parseRoute("#/inbox"), "inbox");
  assert.equal(parseRoute("#/inbox/i-1"), "inbox");
  assert.equal(inboxHash("item/i-1"), "#/inbox/i-1");
  assert.equal(inboxPath("#/inbox/i-1"), "item/i-1");
  assert.equal(inboxPath("#/inbox/done"), "done");
  assert.equal(legacyInboxHash("#/app/inbox/item/i-1"), "#/inbox/i-1");
  assert.equal(legacyInboxHash("#/app/docker"), "");
});

test("workspace overview is a page route, not an editor tab", () => {
  const hash = workspaceOverviewHash("ws/one");
  assert.equal(workspaceOverviewRoute(hash), "ws/one");
  assert.equal(parseRoute(hash), "workspaceOverview");
  assert.equal(workspaceOverviewRoute("#/workspaces/%ZZ/overview"), null);
  const agentsHash = workspaceAgentsHash("ws/one");
  assert.equal(workspaceAgentsRoute(agentsHash), "ws/one");
  assert.equal(parseRoute(agentsHash), "workspaceAgents");
  assert.equal(workspaceAgentsRoute("#/workspaces/%ZZ/agents"), null);
});

test("integrations deep links remain reload-safe", () => {
  for (const hash of ["#/integrations", "#/integrations/webhooks"]) assert.equal(parseRoute(hash), "integrations");
  // `#/mcps` and `#/integrations/connectors` stopped meaning Pi's connectors on 2026-09-25.
  assert.notEqual(parseRoute("#/mcps"), "clis");
});
import { isWebTab, tabWebId, webTabId, webHash, webRoute, boundWorkTab } from "./routes.js";
import { parseRoute, ROUTES, go, providersLlama, pinRoute, prefSection, agentRoute, workspaceHash, termRoute, termHash, sessionsHash, sessionsRoute, isTermTab, termTabId, tabTermId, fileTabId, isFileTab, parseFileTab, fileHash, fileRoute, gitHash, gitRoute, gitTabId, isGitTab, gitTabKey, treeHash, treeRoute, treeTabId, isTreeTab, treeTabRoot, appTabId, isAppTab, tabAppId, appHash, appRoute, renamedAppId, renamedAppHash, renamedTabId, snippetRoute, snippetsHash } from "./routes.js";

test("preferences and settings are distinct", () => {
  assert.equal(parseRoute("#/preferences"), "preferences");
  assert.equal(parseRoute("#/preferences/backup"), "preferences");
  assert.equal(prefSection("#/preferences"), "appearance");
  // terminal appearance moved to #/termset; the old link degrades gracefully
  assert.equal(prefSection("#/preferences/terminal"), "appearance");
  assert.equal(prefSection("#/preferences/backup"), "backup");
  assert.equal(prefSection("#/preferences/landing"), "landing");
  assert.equal(parseRoute("#/clis/settings"), "clis");
  assert.equal(ROUTES.preferences, "/preferences");
  assert.equal(ROUTES.settings, undefined, "no Pi default: go() resolves the CLI");
  assert.equal(providersLlama("#/providers/llama"), true);
  assert.equal(parseRoute("#/pins/new"), "pins");
  assert.deepEqual(pinRoute("#/pins/new"), { mode: "new", id: "" });
  assert.deepEqual(pinRoute("#/pins/hello-abc"), { mode: "edit", id: "hello-abc" });
  assert.equal(parseRoute("#/snippets"), "snippets");
  assert.equal(parseRoute("#/snippets/new"), "snippets");
  assert.equal(parseRoute("#/outcomes"), "outcomes");
  assert.equal(parseRoute("#/history"), "history");
  assert.equal(snippetRoute("#/snippets"), "");
  assert.equal(snippetRoute("#/snippets/new"), "new");
  assert.equal(snippetsHash("new"), "#/snippets/new");
});

test("agent hash is still the workspace shell", () => {
  assert.equal(parseRoute("#/agent/grok-c87aca"), "workspace");
  assert.equal(parseRoute("#/"), "workspace");
  assert.equal(agentRoute("#/agent/grok-c87aca"), "grok-c87aca");
  assert.equal(agentRoute("#/agent/a%2Fb"), "a/b");
  assert.equal(agentRoute("#/"), null);
  assert.equal(agentRoute("#/mcps"), null);
  assert.equal(workspaceHash("grok-c87aca"), "#/agent/grok-c87aca");
  assert.equal(workspaceHash(null), "#/");
  assert.equal(parseRoute("#/term/term-abc"), "workspace");
  assert.equal(termRoute("#/term/term-abc"), "term-abc");
  assert.equal(termHash("term-abc"), "#/term/term-abc");
  assert.equal(isTermTab("t:term-abc"), true);
  assert.equal(tabTermId("t:term-abc"), "term-abc");
  assert.equal(termTabId("term-abc"), "t:term-abc");
  assert.equal(isTermTab("grok-c87aca"), false);
});

test("file tabs encode path in the hash and the tab id", () => {
  const id = fileTabId("term", "term-abc", "web/src/a.js");
  assert.equal(isFileTab(id), true);
  assert.deepEqual(parseFileTab(id), { kind: "term", id: "term-abc", path: "web/src/a.js" });
  assert.equal(fileHash("term", "term-abc", "web/src/a.js"), "#/file/t/term-abc/web%2Fsrc%2Fa.js");
  assert.deepEqual(fileRoute("#/file/t/term-abc/web%2Fsrc%2Fa.js"), { kind: "term", id: "term-abc", path: "web/src/a.js" });
  assert.equal(parseRoute("#/file/t/term-abc/web%2Fsrc%2Fa.js"), "workspace");
  assert.equal(isFileTab("t:term-abc"), false);
});

test("the git hash names the owner, the tab id names the repository", () => {
  // ADR-0022: two agents in two worktrees of one repo ask by different hashes
  // and land on the same tab.
  assert.equal(parseRoute("#/git/a/opus"), "workspace");
  assert.deepEqual(gitRoute("#/git/a/opus"), { kind: "agent", id: "opus" });
  assert.deepEqual(gitRoute("#/git/t/sh1"), { kind: "term", id: "sh1" });
  assert.equal(gitRoute("#/agent/opus"), null);
  assert.equal(gitRoute("#/git/x/opus"), null);

  assert.equal(gitHash("agent", "opus"), "#/git/a/opus");
  assert.equal(gitHash("term", "sh1"), "#/git/t/sh1");
  assert.deepEqual(gitRoute(gitHash("agent", "a/b")), { kind: "agent", id: "a/b" });

  // A folder with nobody in it is an owner too (ADR-0027/ADR-0030): the
  // workspace asks for its own repository, exactly as the file tree does.
  assert.equal(gitHash("workspace", "ws1"), "#/git/w/ws1");
  assert.deepEqual(gitRoute("#/git/w/ws1"), { kind: "workspace", id: "ws1" });

  const key = "/home/goat/picode/.git";
  assert.equal(gitTabId(key), "g:" + key);
  assert.ok(isGitTab(gitTabId(key)));
  assert.equal(gitTabKey(gitTabId(key)), key);
  assert.equal(gitTabId(""), "");
  assert.ok(!isGitTab("a:opus"));
  assert.ok(!isGitTab(""));
  assert.equal(gitTabKey("t:sh1"), "");
});

test("git tabs are distinct from file and terminal tabs", () => {
  const git = gitTabId("/repo/.git");
  assert.ok(!isFileTab(git) && !isTermTab(git));
  assert.ok(!isGitTab(fileTabId("agent", "opus", "a.js")));
  assert.ok(!isGitTab(termTabId("sh1")));
});

test("sessions live under Agent CLIs (ADR-0079)", () => {
  // The Pi-era #/sessions* and #/clis/sessions* addresses were retired on 2026-09-25.
  assert.notEqual(parseRoute("#/sessions"), "clis");
  assert.notEqual(parseRoute("#/sessions/ws-9"), "clis");
  assert.equal(sessionsRoute("#/clis/pi/sessions/ws-9"), "ws-9");
  assert.equal(sessionsRoute("#/clis/sessions/ws-9"), null);
  assert.equal(sessionsRoute("#/clis/pi/sessions"), null);
  assert.equal(sessionsRoute("#/clis/sessions"), null);
  assert.equal(sessionsRoute("#/agent/opus"), null);
  assert.equal(sessionsHash("ws-9"), "#/clis/pi/sessions/ws-9");
  assert.equal(sessionsHash(""), "#/clis/pi/sessions");
  assert.equal(sessionsHash("ws-9", "codex"), "#/clis/codex/sessions/ws-9");
  assert.equal(sessionsHash("", "grok"), "#/clis/grok/sessions");
});

test("tree hash names the owner, tab id names the root folder", () => {
  assert.equal(parseRoute("#/tree/a/opus"), "workspace");
  assert.equal(treeHash("agent", "opus"), "#/tree/a/opus");
  assert.equal(treeHash("term", "sh1"), "#/tree/t/sh1");
  assert.equal(treeHash("workspace", "ws-9"), "#/tree/w/ws-9");
  assert.deepEqual(treeRoute("#/tree/w/ws-9"), { kind: "workspace", id: "ws-9" });
  assert.deepEqual(treeRoute("#/tree/t/sh1"), { kind: "term", id: "sh1" });
  assert.deepEqual(treeRoute("#/tree/a/opus"), { kind: "agent", id: "opus" });
  assert.equal(treeRoute("#/git/a/opus"), null);
  assert.equal(treeTabId("/home/u/proj"), "d:/home/u/proj");
  assert.ok(isTreeTab("d:/home/u/proj"));
  assert.equal(treeTabRoot("d:/home/u/proj"), "/home/u/proj");
  assert.equal(treeTabRoot("g:/repo/.git"), "");
});

test("tree tabs are distinct from every other tab family", () => {
  const tree = treeTabId("/home/u/proj");
  assert.ok(!isFileTab(tree) && !isTermTab(tree) && !isGitTab(tree));
  assert.ok(!isTreeTab(gitTabId("/repo/.git")));
  assert.ok(!isTreeTab(termTabId("sh1")));
});

test("file tabs carry the workspace owner since ADR-0030", () => {
  assert.equal(fileHash("workspace", "ws-9", "src/a.go"), "#/file/w/ws-9/src%2Fa.go");
  assert.deepEqual(fileRoute("#/file/w/ws-9/src%2Fa.go"), { kind: "workspace", id: "ws-9", path: "src/a.go" });
  assert.deepEqual(parseFileTab(fileTabId("workspace", "ws-9", "a.go")), { kind: "workspace", id: "ws-9", path: "a.go" });
  // the old two-owner ids still parse
  assert.deepEqual(parseFileTab("f:t:sh1:a.go"), { kind: "term", id: "sh1", path: "a.go" });
});

test("app tabs (ADR-0036) are self-describing", () => {
  assert.equal(appTabId("demo"), "x:demo");
  assert.ok(isAppTab("x:demo"));
  assert.equal(tabAppId("x:demo"), "demo");
  assert.equal(tabAppId("t:demo"), "");
  assert.equal(appHash("demo"), "#/app/demo");
  assert.equal(appRoute("#/app/demo"), "demo");
  assert.equal(appRoute("#/app/a%20b"), "a b");
  assert.equal(appRoute("#/agent/demo"), null);
  assert.equal(parseRoute("#/app/demo"), "workspace");
  const review = appHash("docker", "plan/qa review");
  assert.equal(appRoute(review), "docker");
  assert.equal(appPath(review), "plan/qa review");
  assert.equal(appPath("#/app/docker"), "");
  assert.equal(appPath("#/agent/qa"), "");
});

// ADR-0118: the Matrix app became Canvas. One map answers the hash redirect
// and the tab restore, so a bookmark and a saved tab strip agree.
test("a renamed app keeps its old deep links and its old tab id working", () => {
  assert.equal(renamedAppId("matrix"), "canvas");
  assert.equal(renamedAppId("canvas"), "", "the current id is not renamed to itself");
  assert.equal(renamedAppId("inbox"), "");
  assert.equal(renamedAppHash("#/app/matrix"), "#/app/canvas");
  assert.equal(renamedAppHash("#/app/matrix/m-42"), "#/app/canvas/m-42");
  assert.equal(renamedAppHash("#/app/matrix/a%20b"), "#/app/canvas/a%20b", "the path survives the round trip");
  assert.equal(renamedAppHash("#/app/canvas/m-42"), "", "already canonical: nothing to replace");
  assert.equal(renamedAppHash("#/agent/matrix"), "", "only #/app/* is ours");
  assert.equal(renamedAppHash("#/clis/sessions"), "");
  assert.equal(renamedTabId("x:matrix"), "x:canvas");
  assert.equal(renamedTabId("x:canvas"), "");
  assert.equal(renamedTabId("t:matrix"), "", "a terminal called matrix is a terminal");
  assert.equal(renamedTabId(""), "");
});

test("app tabs are distinct from every other tab family", () => {
  const app = appTabId("demo");
  assert.ok(!isFileTab(app) && !isTermTab(app) && !isGitTab(app) && !isTreeTab(app));
  assert.ok(!isAppTab(termTabId("demo")));
  assert.ok(!isAppTab(gitTabId("/repo/.git")));
  assert.ok(!isAppTab(treeTabId("/home/u/proj")));
});

test("llama manager owns its routes and the legacy link", () => {
 for (const hash of ["#/llama", "#/llama/models", "#/llama/server", "#/llama/activity", "#/providers/llama"]) assert.equal(parseRoute(hash), "llama");
 assert.notEqual(parseRoute("#/providers/new"), "clis");
});

test("packages config lives under Agent CLIs; the Pi-era address is retired", () => {
  assert.equal(parseRoute("#/clis/pi/packages/config/pi-roles"), "clis");
  assert.notEqual(parseRoute("#/packages"), "clis");
});

test("native provider navigation; the Pi-era aliases are retired", () => {
  for (const hash of ["#/clis/pi/providers", "#/clis/pi/providers/new", "#/clis/codex/providers"]) assert.deepEqual(parseRoute(hash), "clis");
  for (const hash of ["#/providers", "#/providers/new", "#/more/providers", "#/more/providers/new"]) assert.notEqual(parseRoute(hash), "clis", hash);
  assert.deepEqual(parseRoute("#/more/providers/llama?tab=models"), "llama");
});

// | agent in context | its cli   | a CLI pane command opens        |
// | none             | —         | the catalog, #/clis             |
// | yes              | empty     | Pi's pane (agentIsPi)           |
// | yes              | <cli>     | that CLI's pane (ADR-0179)      |
test("CLI pane commands open the context's CLI, or the catalog with none", () => {
  const previous = globalThis.location;
  globalThis.location = { hash: "" };
  try {
    for (const name of ["settings", "packages", "skills", "mcps", "connectors", "providers", "providers-new", "providers-custom"]) {
      location.hash = ""; go(name); assert.equal(location.hash, "#/clis", name);
      location.hash = ""; go(name, "", { workspaceId: "W" }); assert.equal(location.hash, "#/clis", name + " with a workspace only");
    }
    go("providers", "A"); assert.equal(location.hash, "#/clis/pi/providers");
    go("providers-new", "A"); assert.equal(location.hash, "#/clis/pi/providers/new");
    go("providers-custom", "A", { cli: "omp" }); assert.equal(location.hash, "#/clis/omp/providers/custom");
    go("providers-custom", "A", { cli: "pi", customId: "cheap" }); assert.equal(location.hash, "#/clis/pi/providers/custom/cheap");
    go("mcps", "A", { workspaceId: "W" });
    assert.equal(location.hash, "#/clis/pi/connectors?workspaceId=W&agentId=A");
    go("connectors", "A", { workspaceId: "W" });
    assert.equal(location.hash, "#/clis/pi/connectors?workspaceId=W&agentId=A");
    go("packages", "A", { workspaceId: "W" });
    assert.equal(location.hash, "#/clis/pi/packages?workspaceId=W&agentId=A");
    go("settings", "A", { workspaceId: "W" });
    assert.equal(location.hash, "#/clis/pi/settings?workspaceId=W&agentId=A");
    // A selected agent of another CLI opens that CLI's pane (ADR-0179).
    go("packages", "C", { workspaceId: "W", cli: "claude-code" });
    assert.equal(location.hash, "#/clis/claude-code/packages?workspaceId=W&agentId=C");
    go("connectors", "C", { workspaceId: "W", cli: "codex" });
    assert.match(location.hash, /^#\/clis\/codex\/connectors/);
    go("providers", "C", { cli: "omp" }); assert.equal(location.hash, "#/clis/omp/providers");
    // The catalog view follows the same rule: a selected agent opens its own
    // CLI's page; no agent in context opens the catalog.
    go("clis", "C", { workspaceId: "W", cli: "claude-code" });
    assert.equal(location.hash, "#/clis/claude-code");
    go("clis", "P", {});
    assert.equal(location.hash, "#/clis/pi");
    go("clis", "", {});
    assert.equal(location.hash, "#/clis");
  }
  finally { globalThis.location = previous; }
});

test("web tabs encode with the w: prefix", () => {
  const id = webTabId("abc123");
  assert.equal(id, "w:abc123");
  assert.ok(isWebTab(id));
  assert.equal(tabWebId(id), "abc123");
  assert.ok(!isWebTab("t:abc123"));
  assert.ok(!isWebTab(""));
});

test("isAgentTab is true only for a bare agent id — the fetches that ask this", () => {
  // The regression: role-state and slash were guarded with isTermTab alone, so
  // a file/tree/git/app tab asked the API for its role as an agent (2 x 404 per
  // selection).
  assert.equal(isAgentTab("workspace-agent-id"), true);
  assert.equal(isAgentTab(""), false);
  assert.equal(isAgentTab(null), false);
  assert.equal(isAgentTab("t:desktop-51c42d"), false);
  assert.equal(isAgentTab(fileTabId("w", "w1", "src/App.jsx")), false);
  assert.equal(isAgentTab(treeTabId("/home/goat/picode")), false);
  assert.equal(isAgentTab(gitTabId("/home/goat/picode/.git")), false);
  assert.equal(isAgentTab(appTabId("canvas")), false);
  assert.equal(isAgentTab(webTabId("3")), false);
});

test("webHash/webRoute round-trip — the router can hold a work-browser tab", () => {
  // 2026-09-14 regression: web tabs owned no address, so the hash router kept
  // re-resolving the previous tab's route and yanked the selection back.
  assert.equal(webHash("2"), "#/web/2");
  assert.equal(webRoute("#/web/2"), "2");
  assert.equal(webHash("") || webHash(null), "#/");
  assert.equal(webRoute("#/"), null);
  assert.equal(webRoute("#/term/desktop-51c42d"), null);
  assert.equal(webRoute("#/agent/abc"), null);
  // A web id is not a path — one segment only.
  assert.equal(webRoute("#/web/2/extra"), null);
  // And it parses as a workspace-scope route so the write side may replace it.
  assert.equal(parseRoute("#/web/2"), "workspace");
});

test("boundWorkTab — the channel reads the selected tab's binding (ADR-0135)", () => {
  assert.equal(boundWorkTab("w:2", {}), "w:2");
  assert.equal(boundWorkTab("t:desktop-51c42d", { "t:desktop-51c42d": "3" }), "w:3");
  assert.equal(boundWorkTab("agent-1", { "agent-1": "7" }), "w:7");
  assert.equal(boundWorkTab("t:desktop-51c42d", {}), null);
  assert.equal(boundWorkTab(null, { "": "3" }), null);
});

test("delivery links keep owner identity separate from view",()=>{
 assert.deepEqual(gitRoute(gitHash("workspace","project","delivery")),{kind:"workspace",id:"project",view:"delivery"});
});

import { instructionsHash, instructionsRoute, instructionsTabId, isInstructionsTab, instructionsTabWorkspace } from "./routes.js";

test("instructions: one tab per workspace, round-tripped through the hash", () => {
  assert.equal(parseRoute("#/instructions/ws_1"), "instructions");
  assert.equal(instructionsHash("ws 1"), "#/instructions/ws%201");
  assert.equal(instructionsRoute("#/instructions/ws%201"), "ws 1");
  assert.equal(instructionsRoute("#/instructions/"), null);
  assert.equal(instructionsRoute("#/instructions/a/b"), null);
  assert.equal(instructionsRoute("#/tree/w/ws_1"), null);
  assert.equal(instructionsTabId("ws_1"), "i:ws_1");
  assert.equal(instructionsTabId(""), "");
  assert.equal(isInstructionsTab("i:ws_1"), true);
  assert.equal(isInstructionsTab("d:/x"), false);
  assert.equal(instructionsTabWorkspace("i:ws_1"), "ws_1");
  assert.equal(instructionsTabWorkspace("g:x"), "");
});

test("an Instructions tab is not an agent (it carried i:<workspace> into ?agentId=, 2026-09-23)", () => {
  assert.equal(isAgentTab("i:ws_1"), false);
  assert.equal(isAgentTab("ag_1"), true);
});
