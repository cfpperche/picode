import assert from "node:assert/strict";
import { test } from "node:test";
import { parseRoute, ROUTES, go, providersNew, providersLlama, pinRoute, prefSection, agentRoute, workspaceHash, termRoute, termHash, sessionsHash, sessionsRoute, isTermTab, termTabId, tabTermId, fileTabId, isFileTab, parseFileTab, fileHash, fileRoute, gitHash, gitRoute, gitTabId, isGitTab, gitTabKey, treeHash, treeRoute, treeTabId, isTreeTab, treeTabRoot } from "./routes.js";

test("preferences and settings are distinct", () => {
  assert.equal(parseRoute("#/preferences"), "preferences");
  assert.equal(parseRoute("#/preferences/backup"), "preferences");
  assert.equal(prefSection("#/preferences"), "appearance");
  // terminal appearance moved to #/termset; the old link degrades gracefully
  assert.equal(prefSection("#/preferences/terminal"), "appearance");
  assert.equal(prefSection("#/preferences/backup"), "backup");
  assert.equal(parseRoute("#/settings"), "settings");
  assert.equal(ROUTES.preferences, "/preferences");
  assert.equal(ROUTES.settings, "/settings");
  assert.equal(parseRoute("#/providers/new"), "providers");
  assert.equal(providersNew("#/providers/new"), true);
  assert.equal(providersLlama("#/providers/llama"), true);
  assert.equal(parseRoute("#/pins/new"), "pins");
  assert.deepEqual(pinRoute("#/pins/new"), { mode: "new", id: "" });
  assert.deepEqual(pinRoute("#/pins/hello-abc"), { mode: "edit", id: "hello-abc" });
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

test("sessions route parses workspace id", () => {
  assert.equal(parseRoute("#/sessions"), "sessions");
  assert.equal(parseRoute("#/sessions/ws-9"), "sessions");
  assert.equal(sessionsRoute("#/sessions/ws-9"), "ws-9");
  assert.equal(sessionsRoute("#/sessions"), null);
  assert.equal(sessionsRoute("#/agent/opus"), null);
  assert.equal(sessionsHash("ws-9"), "#/sessions/ws-9");
  assert.equal(sessionsHash(""), "#/");
});

test("go(sessions) lands on the machine-wide view, not the :id template", () => {
  const orig = globalThis.location;
  globalThis.location = { hash: "" };
  try {
    go("sessions");
    assert.equal(globalThis.location.hash, "#/sessions");
  } finally {
    globalThis.location = orig;
  }
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
