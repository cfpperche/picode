import assert from "node:assert/strict";
import test from "node:test";
import { containerIds, moveId, movePhrase, optimisticFleet, reorderIds, sameIds, sortIdsBy } from "./sidebarOrder.js";

test("moveId steps one place and refuses the ends", () => {
  assert.deepEqual(moveId(["a", "b", "c"], "b", -1), ["b", "a", "c"]);
  assert.deepEqual(moveId(["a", "b", "c"], "b", 1), ["a", "c", "b"]);
  const ids = ["a", "b"];
  assert.equal(moveId(ids, "a", -1), ids);
  assert.equal(moveId(ids, "b", 1), ids);
  assert.equal(moveId(ids, "missing", 1), ids);
});

test("reorderIds drops active onto over", () => {
  assert.deepEqual(reorderIds(["a", "b", "c"], "c", "a"), ["c", "a", "b"]);
  const ids = ["a", "b"];
  assert.equal(reorderIds(ids, "a", "a"), ids);
  assert.equal(reorderIds(ids, "a", null), ids);
});

test("movePhrase names the neighbour", () => {
  const label = (id) => id.toUpperCase();
  assert.equal(movePhrase(["a", "b"], ["b", "a"], "b", label), "Moved B to the top");
  assert.equal(movePhrase(["a", "b"], ["b", "a"], "a", label), "Moved A below B");
  assert.equal(movePhrase(["a"], ["a"], "a", label), "");
});

// "Sort agents by name" (ADR-0173): case-insensitive, numeric, and a no-op
// keeps the stored order — the same ids written nothing and emitted nothing.
test("sortIdsBy orders by display key and keeps ties in place", () => {
  assert.deepEqual(sortIdsBy(["c", "a", "b"], (id) => id), ["a", "b", "c"]);
  assert.deepEqual(sortIdsBy(["a10", "A2", "a1"], (id) => id), ["a1", "A2", "a10"]);
  assert.deepEqual(sortIdsBy(["x", "b", "a"], () => ""), ["x", "b", "a"]);
  const ids = ["c", "a", "b"];
  assert.deepEqual(sortIdsBy(ids, (id) => id), ["a", "b", "c"]);
  assert.deepEqual(ids, ["c", "a", "b"], "input stays untouched");
  assert.deepEqual(sortIdsBy([], (id) => id), []);
  assert.deepEqual(sortIdsBy(undefined, (id) => id), []);
});

test("optimisticFleet appends the permutation and ignores a no-op", () => {
  const fleet = {
    workspaces: [{ id: "b", agents: [{ id: "b1" }, { id: "b2" }] }, { id: "a", agents: [] }],
    freeAgents: [{ id: "f1" }, { id: "f2" }],
    terminals: [{ id: "t1", workspaceId: "a" }, { id: "t2", workspaceId: "ws_free" }, { id: "t3", workspaceId: "a" }],
  };
  assert.equal(optimisticFleet(fleet, "workspaces", null, ["b", "a"]), fleet);
  const swapped = optimisticFleet(fleet, "workspaces", null, ["a", "b"]);
  assert.deepEqual(swapped.workspaces.map((w) => w.id), ["a", "b"]);
  assert.deepEqual(containerIds(swapped, "agents", "b"), ["b1", "b2"]);
  const agents = optimisticFleet(fleet, "agents", "b", ["b2", "b1"]);
  assert.deepEqual(containerIds(agents, "agents", "b"), ["b2", "b1"]);
  const terms = optimisticFleet(fleet, "terminals", "a", ["t3", "t1"]);
  assert.deepEqual(containerIds(terms, "terminals", "a"), ["t3", "t1"]);
  assert.deepEqual(containerIds(terms, "terminals", "ws_free"), ["t2"]);
  assert.equal(optimisticFleet(fleet, "workspaces", null, ["a"]), null);
  assert.equal(sameIds(["a"], ["a"]), true);
});
