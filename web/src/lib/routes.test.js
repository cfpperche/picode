import assert from "node:assert/strict";
import { test } from "node:test";
import { parseRoute, ROUTES, providersNew, providersLlama, pinRoute, prefSection, agentRoute, workspaceHash, termRoute, termHash, isTermTab, termTabId, tabTermId, fileTabId, isFileTab, parseFileTab, fileHash, fileRoute } from "./routes.js";

test("preferences and settings are distinct", () => {
  assert.equal(parseRoute("#/preferences"), "preferences");
  assert.equal(parseRoute("#/preferences/backup"), "preferences");
  assert.equal(prefSection("#/preferences"), "appearance");
  assert.equal(prefSection("#/preferences/terminal"), "terminal");
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
