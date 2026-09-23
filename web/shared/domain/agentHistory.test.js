import assert from "node:assert/strict";
import { test } from "node:test";
import { historyWorkspaceChoice, historyWorkspaceOptions } from "./agentHistory.js";

test("options list living workspaces, then the free list once", () => {
  assert.deepEqual(
    historyWorkspaceOptions([{ id: "ws_free", name: "Free" }, { id: "w1", name: "api" }, { id: "w2" }]),
    [["w1", "api"], ["w2", "w2"], ["ws_free", "Free agents"]],
  );
  assert.deepEqual(historyWorkspaceOptions(null), [["ws_free", "Free agents"]]);
});

test("the target is the pick, else its own workspace while it exists, else none", () => {
  const alive = { exit: { workspaceId: "w1" }, workspaceExists: true };
  const gone = { exit: { workspaceId: "w9" }, workspaceExists: false };
  assert.equal(historyWorkspaceChoice(alive, ""), "w1");
  assert.equal(historyWorkspaceChoice(alive, "w2"), "w2");
  assert.equal(historyWorkspaceChoice(gone, ""), "");
  assert.equal(historyWorkspaceChoice(gone, "ws_free"), "ws_free");
});
