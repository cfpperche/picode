import assert from "node:assert/strict";
import { test } from "node:test";
import { termWorkspaceId, freeTerminals, workspaceTerminals, workspaceForTerminal, FREE_WS } from "./termGroups.js";

const terms = [
  { id: "t1", name: "zsh", workspaceId: "ws_free" },
  { id: "t2", name: "Build", workspaceId: "w1" },
  { id: "t3", name: "api", workspaceId: "w1" },
  { id: "t4", name: "Deploy" },
];

test("splits free terminals from workspace terminals", () => {
  assert.deepEqual(freeTerminals(terms).map((t) => t.id), ["t1", "t4"]);
  assert.deepEqual(workspaceTerminals(terms, "w1").map((t) => t.id), ["t2", "t3"]);
  assert.deepEqual(workspaceTerminals(terms, "w2"), []);
});

test("treats missing workspaceId as free", () => {
  assert.equal(termWorkspaceId({ id: "x" }), FREE_WS);
  assert.equal(termWorkspaceId(null), FREE_WS);
  assert.ok(freeTerminals(terms).some((t) => t.id === "t4"));
});

test("keeps the server order inside a group", () => {
  assert.deepEqual(workspaceTerminals(terms, "w1").map((t) => t.name), ["Build", "api"]);
});

test("workspaceForTerminal returns the owning workspace, not a free one", () => {
  const workspaces = [{ id: "w1", name: "PiCode" }, { id: "w2", name: "Other" }];
  assert.equal(workspaceForTerminal(terms, workspaces, "t2").id, "w1");
  assert.equal(workspaceForTerminal(terms, workspaces, "t1"), null);
  assert.equal(workspaceForTerminal(terms, workspaces, "t4"), null);
  assert.equal(workspaceForTerminal(terms, workspaces, "gone"), null);
});
