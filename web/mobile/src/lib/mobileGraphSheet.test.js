import test from "node:test";
import assert from "node:assert/strict";
import { targetFor, sheetMenu, sheetGroups, doors, doorHint, gate } from "./git/graphSheet.js";

const catalog = {};
for (const [id, tier, needs] of [
  ["fetch", "A", []], ["pull", "B", []], ["push", "C", []], ["pr", "B", []],
  ["create-branch", "A", ["target", "name"]], ["merge", "B", ["target"]], ["reset-hard", "C", ["target"]],
  ["checkout", "A", ["target"]], ["create-worktree", "A", ["target", "name"]], ["cherry-pick", "B", ["target"]],
  ["revert", "B", ["target"]], ["rebase", "B", ["target"]], ["reset-soft", "B", ["target"]], ["reset-mixed", "B", ["target"]],
  ["checkout-detach", "A", ["target"]], ["create-tag", "A", ["target", "name"]], ["rename-branch", "B", ["target", "name"]],
  ["delete-branch", "B", ["target"]], ["commit", "B", ["message"]],
]) catalog[id] = { id, tier, needs };

const graph = {
  head: "aaa", remotes: ["origin"],
  refs: [
    { name: "main", kind: "head", hash: "aaa", worktree: "/repo", merged: true },
    { name: "feat/x", kind: "head", hash: "bbb", worktree: "/repo/.worktrees/x" },
    { name: "v1", kind: "tag", hash: "bbb" },
  ],
  worktrees: [
    { path: "/repo", head: "aaa", branch: "main", self: true, agents: [] },
    { path: "/repo/.worktrees/x", head: "bbb", branch: "feat/x", agents: [{ id: "t1", name: "Pi", kind: "terminal", live: true }] },
  ],
};
const ctx = { workspaces: [], freeAgents: [], terminals: [{ id: "t1", name: "Pi", state: "working" }], catalog };

test("a commit target carries only the refs drawn on that commit", () => {
  const t = targetFor({ commit: { hash: "bbb", subject: "s" }, refs: graph.refs });
  assert.equal(t.kind, "commit");
  assert.deepEqual(t.refs.map((r) => r.name), ["feat/x", "v1"]);
  assert.equal(targetFor({}), null);
  assert.equal(targetFor({ repo: true }).kind, "repo");
  assert.deepEqual(targetFor({ worktree: graph.worktrees[1], uncommitted: true }), { kind: "worktree", worktree: graph.worktrees[1], uncommitted: true });
});

test("the sheet's groups follow the shared menu's sections, in order", () => {
  const menu = sheetMenu(targetFor({ commit: { hash: "bbb", subject: "s" }, refs: graph.refs }), graph, ctx);
  const groups = sheetGroups(menu);
  assert.deepEqual(groups.map((g) => g.label), ["", "Create", "Apply", "Move this branch", "Branch feat/x", "Tag v1"]);
  assert.deepEqual(groups[0].items.map((i) => i.id), ["copy-hash", "copy-subject"]);
  assert.equal(groups.find((g) => g.label === "Branch feat/x").state, "Checked out in x.");
  assert.ok(groups.find((g) => g.label === "Branch feat/x").items.some((i) => i.kind === "open-terminal"), "the pi terminal's door is on the branch group");
  // No group is empty, and nothing nests.
  for (const g of groups) assert.ok(g.items.length && g.items.every((i) => i.kind !== "sub"), g.label);
});

test("the header target lists the repository's own actions", () => {
  const menu = sheetMenu({ kind: "repo" }, graph, ctx);
  assert.deepEqual(menu.items.map((i) => i.action), ["fetch", "pull", "push", "pr", "create-branch"]);
});

test("without the catalog the sheet shows tier 0 only", () => {
  const menu = sheetMenu(targetFor({ commit: { hash: "bbb", subject: "s" } }), graph, { ...ctx, catalog: null });
  assert.ok(menu.items.every((i) => i.kind !== "action"));
});

test("doors are the two terminal ones plus one ask per running occupant, by name", () => {
  const menu = sheetMenu(targetFor({ commit: { hash: "bbb" } }), graph, ctx);
  const list = doors(menu.occupants);
  assert.deepEqual(list.map((d) => d.value), ["prepare", "run", "ask:t1"]);
  assert.equal(list[2].label, "Ask Pi");
  assert.equal(list[2].who.kind, "terminal");
  // A stopped occupant has no door.
  assert.deepEqual(doors([{ id: "x", name: "Old", running: false }]).map((d) => d.value), ["prepare", "run"]);
});

test("the gate is the shared one: a typed phrase only for tier C through run", () => {
  assert.equal(gate({ tier: "C", door: "run", action: "reset-hard", target: "bbb" }).typed, "discard");
  assert.equal(gate({ tier: "C", door: "prepare", action: "reset-hard" }).typed, "");
  assert.equal(gate({ tier: "B", door: "run", action: "merge" }).typed, "");
});

test("the hint under the select says who presses Enter", () => {
  assert.match(doorHint("prepare"), /you to press Enter/);
  assert.match(doorHint("run"), /no agent is working/);
  assert.match(doorHint("ask:t1", { name: "Pi" }), /^Pi does it in its own turn/);
});
