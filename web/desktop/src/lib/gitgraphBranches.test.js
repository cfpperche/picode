import assert from "node:assert/strict";
import { test } from "node:test";
import { visibleRefs, groupBranches, triggerLabel, walkParams, resolveSelection } from "./gitgraphBranches.js";

const refs = [
  { name: "main", kind: "head", hash: "a" },
  { name: "feature", kind: "head", hash: "b" },
  { name: "origin/main", kind: "remote", hash: "a" },
  { name: "v1.0", kind: "tag", hash: "a" },
];

test("visibleRefs always drops tags, drops remotes only when showRemotes is false", () => {
  assert.deepEqual(visibleRefs(refs, true).map((r) => r.name), ["main", "feature", "origin/main"]);
  assert.deepEqual(visibleRefs(refs, false).map((r) => r.name), ["main", "feature"]);
  assert.deepEqual(visibleRefs([], true), []);
});

test("groupBranches sorts each section and drops remote when showRemotes is off", () => {
  assert.deepEqual(groupBranches(refs, true), { local: ["feature", "main"], remote: ["origin/main"] });
  assert.deepEqual(groupBranches(refs, false), { local: ["feature", "main"], remote: [] });
  assert.deepEqual(groupBranches([], true), { local: [], remote: [] });
  assert.deepEqual(groupBranches(null, true), { local: [], remote: [] });
});

test("triggerLabel", () => {
  assert.equal(triggerLabel([]), "Show All");
  assert.equal(triggerLabel(["main"]), "main");
  assert.equal(triggerLabel(["main", "feature"]), "main & 1 more");
  assert.equal(triggerLabel(["main", "feature", "origin/main"]), "main & 2 more");
});

test("walkParams: empty selection carries the remotes toggle, a real selection always implies remotes:true", () => {
  assert.deepEqual(walkParams([], true), { branches: [], remotes: true });
  assert.deepEqual(walkParams([], false), { branches: [], remotes: false });
  assert.deepEqual(walkParams(["main"], false), { branches: ["main"], remotes: true });
  assert.deepEqual(walkParams(["main", "origin/main"], false), { branches: ["main", "origin/main"], remotes: true });
});

test("resolveSelection keeps names still present, drops vanished ones and tags, preserves order", () => {
  assert.deepEqual(resolveSelection(["main", "gone", "feature"], refs), ["main", "feature"]);
  assert.deepEqual(resolveSelection(["v1.0"], refs), []);
  assert.deepEqual(resolveSelection([], refs), []);
  assert.deepEqual(resolveSelection(["main"], []), []);
});
