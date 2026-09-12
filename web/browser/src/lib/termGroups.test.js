import assert from "node:assert/strict";
import { test } from "node:test";
import { termWorkspaceId, freeTerminals, workspaceTerminals, workspaceForTerminal, sortTermsByName, FREE_WS } from "./termGroups.js";

const terms = [
  { id: "t1", name: "zsh", workspaceId: "ws_free" },
  { id: "t2", name: "Build", workspaceId: "w1" },
  { id: "t3", name: "api", workspaceId: "w1" },
  { id: "t4", name: "Deploy" },
];

test("splits free terminals from workspace terminals", () => {
  assert.deepEqual(freeTerminals(terms).map((t) => t.id), ["t4", "t1"]);
  assert.deepEqual(workspaceTerminals(terms, "w1").map((t) => t.id), ["t3", "t2"]);
  assert.deepEqual(workspaceTerminals(terms, "w2"), []);
});

test("treats missing workspaceId as free", () => {
  assert.equal(termWorkspaceId({ id: "x" }), FREE_WS);
  assert.equal(termWorkspaceId(null), FREE_WS);
  assert.ok(freeTerminals(terms).some((t) => t.id === "t4"));
});

test("sorts by name, base sensitivity", () => {
  const out = sortTermsByName([{ name: "b" }, { name: "A" }, { name: "a2" }]);
  assert.deepEqual(out.map((t) => t.name), ["A", "a2", "b"]);
});

test("workspaceForTerminal returns the owning workspace, not a free one", () => {
  const workspaces = [{ id: "w1", name: "PiCode" }, { id: "w2", name: "Other" }];
  assert.equal(workspaceForTerminal(terms, workspaces, "t2").id, "w1");
  assert.equal(workspaceForTerminal(terms, workspaces, "t1"), null);
  assert.equal(workspaceForTerminal(terms, workspaces, "t4"), null);
  assert.equal(workspaceForTerminal(terms, workspaces, "gone"), null);
});
