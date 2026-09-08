import test from "node:test";
import assert from "node:assert/strict";
import { graphActions, repoOccupants, busyLine, trackingLabel, trackingTitle } from "./graphActions.js";

// A repository with two checkouts: the reader's own on main, and a sibling on
// feat/x where an agent lives. This is the shape ADR-0073 draws and the one
// every interesting menu decision turns on.
function graph() {
  return {
    head: "aaa",
    remotes: ["origin"],
    commits: [{ hash: "a".repeat(40), subject: "first", author: "t", at: 1 }],
    refs: [
      { name: "main", kind: "head", hash: "aaa", upstream: "origin/main", ahead: 2, worktree: "/repo", merged: true },
      { name: "feat/x", kind: "head", hash: "bbb", worktree: "/repo/.worktrees/x" },
      { name: "parked", kind: "head", hash: "ccc", merged: true },
      { name: "origin/main", kind: "remote", hash: "aaa" },
      { name: "v1", kind: "tag", hash: "aaa" },
    ],
    worktrees: [
      { path: "/repo", head: "aaa", branch: "main", self: true, agents: [] },
      { path: "/repo/.worktrees/x", head: "bbb", branch: "feat/x", agents: [{ id: "a1", name: "Atlas" }] },
    ],
  };
}

const ctx = { workspaces: [{ agents: [{ id: "a1", name: "Atlas", running: true, streaming: true }] }], freeAgents: [] };

test("a commit offers its clipboard actions and nothing that writes", () => {
  const menu = graphActions({ kind: "commit", commit: { hash: "a".repeat(40), subject: "first" } }, graph(), ctx);
  assert.equal(menu.kind, "commit");
  assert.equal(menu.title, "aaaaaaa");
  assert.deepEqual(menu.items.map((i) => i.id), ["copy-hash", "copy-subject"]);
  assert.equal(menu.items[0].value, "a".repeat(40));
  assert.ok(menu.items.every((i) => i.kind === "copy"));
});

test("a commit without a subject offers only the hash", () => {
  const menu = graphActions({ kind: "commit", commit: { hash: "b".repeat(40) } }, graph(), ctx);
  assert.deepEqual(menu.items.map((i) => i.id), ["copy-hash"]);
});

// The move no other client can make: git refuses a checkout of a branch held
// by a sibling worktree, so the menu names that checkout and opens it.
test("a branch checked out elsewhere names the checkout and offers its agent", () => {
  const menu = graphActions({ kind: "ref", ref: graph().refs[1] }, graph(), ctx);
  assert.equal(menu.kind, "head");
  assert.equal(menu.state, "Checked out in x.");
  assert.deepEqual(menu.items.map((i) => i.id), ["open-agent:a1", "copy-branch"]);
  assert.equal(menu.items[0].label, "Open Atlas");
  assert.equal(menu.items[0].agentId, "a1");
});

test("the reader's own branch says so and offers no way into itself", () => {
  const menu = graphActions({ kind: "ref", ref: graph().refs[0] }, graph(), ctx);
  assert.equal(menu.state, "Checked out here.");
  assert.deepEqual(menu.items.map((i) => i.id), ["copy-branch"]);
});

test("a branch no checkout holds carries no state line", () => {
  const menu = graphActions({ kind: "ref", ref: graph().refs[2] }, graph(), ctx);
  assert.equal(menu.state, "");
  assert.deepEqual(menu.items.map((i) => i.id), ["copy-branch"]);
});

// A payload written before ADR-0096 has no ref.worktree; the worktree list
// still answers the question, so an old client is not a wrong client.
test("a ref without the worktree field falls back to the worktree list", () => {
  const g = graph();
  const ref = { name: "feat/x", kind: "head", hash: "bbb" };
  const menu = graphActions({ kind: "ref", ref }, g, ctx);
  assert.equal(menu.state, "Checked out in x.");
});

test("remote branches and tags offer their name and nothing else", () => {
  const remote = graphActions({ kind: "ref", ref: graph().refs[3] }, graph(), ctx);
  assert.equal(remote.kind, "remote");
  assert.deepEqual(remote.items.map((i) => i.id), ["copy-branch"]);
  const tag = graphActions({ kind: "ref", ref: graph().refs[4] }, graph(), ctx);
  assert.equal(tag.kind, "tag");
  assert.deepEqual(tag.items.map((i) => i.id), ["copy-tag"]);
});

test("a worktree row offers its agents, its branch and its path", () => {
  const menu = graphActions({ kind: "worktree", worktree: graph().worktrees[1] }, graph(), ctx);
  assert.equal(menu.title, "x");
  assert.deepEqual(menu.items.map((i) => i.id), ["open-agent:a1", "copy-branch", "copy-path"]);
});

test("a detached worktree says so and still offers its path", () => {
  const wt = { path: "/repo/.worktrees/d", head: "ddd", detached: true, agents: [] };
  const menu = graphActions({ kind: "worktree", worktree: wt }, graph(), ctx);
  assert.equal(menu.state, "Detached HEAD.");
  assert.deepEqual(menu.items.map((i) => i.id), ["copy-path"]);
});

test("an unknown or empty target yields an empty menu, never a broken one", () => {
  for (const t of [null, {}, { kind: "nonsense" }, { kind: "ref", ref: null }, { kind: "commit", commit: {} }]) {
    const menu = graphActions(t, graph(), ctx);
    assert.deepEqual(menu.items, []);
    assert.equal(menu.title, "");
  }
});

test("occupants join the graph's own mapping with the app's liveness", () => {
  const list = repoOccupants(graph(), ctx);
  assert.deepEqual(list.map((o) => [o.id, o.name, o.streaming]), [["a1", "Atlas", true]]);
});

test("an agent the app has never heard of still appears, just not as live", () => {
  const list = repoOccupants(graph(), { workspaces: [], freeAgents: [] });
  assert.equal(list.length, 1);
  assert.equal(list[0].streaming, false);
});

test("the busy line speaks only when someone is mid-turn", () => {
  assert.equal(busyLine([]), "");
  assert.equal(busyLine([{ name: "Atlas", streaming: false }]), "");
  assert.equal(busyLine([{ name: "Atlas", streaming: true }]), "Atlas is mid-turn in this repository.");
  assert.equal(
    busyLine([{ name: "Atlas", streaming: true }, { name: "Bo", streaming: true }]),
    "Atlas and 1 more are mid-turn in this repository.",
  );
});

test("every menu carries the busy line so the reader sees it before clicking", () => {
  const menu = graphActions({ kind: "commit", commit: { hash: "a".repeat(40) } }, graph(), ctx);
  assert.equal(menu.busy, "Atlas is mid-turn in this repository.");
});

test("tracking decorates only what needs it", () => {
  assert.equal(trackingLabel({ kind: "head", name: "main", upstream: "origin/main", ahead: 2, behind: 1 }), "↑2 ↓1");
  assert.equal(trackingLabel({ kind: "head", name: "main", upstream: "origin/main" }), "");
  assert.equal(trackingLabel({ kind: "head", name: "main", upstream: "origin/main", gone: true }), "upstream gone");
  assert.equal(trackingLabel({ kind: "head", name: "solo" }), "");
  assert.equal(trackingLabel({ kind: "remote", name: "origin/main" }), "");
  assert.equal(trackingLabel(null), "");
});

test("the tracking tooltip spells out what the arrows mean", () => {
  assert.equal(
    trackingTitle({ kind: "head", name: "main", upstream: "origin/main", ahead: 2, behind: 1 }),
    "main is 2 ahead and 1 behind origin/main",
  );
  assert.equal(
    trackingTitle({ kind: "head", name: "main", upstream: "origin/main" }),
    "main is level with origin/main",
  );
  assert.equal(trackingTitle({ kind: "head", name: "solo" }), "solo has no upstream yet");
  assert.equal(
    trackingTitle({ kind: "head", name: "main", upstream: "origin/main", gone: true }),
    "origin/main no longer exists on the remote",
  );
});

// A pill lives inside the row's button, so a keyboard cannot point at one.
// Everything a pill offers has to be reachable from the row (WCAG 2.1.1).
test("the row menu carries each pill as a submenu", () => {
  const g = graph();
  const menu = graphActions(
    { kind: "commit", commit: { hash: "b".repeat(40), subject: "work" }, refs: [g.refs[1], g.refs[3], g.refs[4]] },
    g,
    ctx,
  );
  const ids = menu.items.map((i) => i.id);
  assert.deepEqual(ids, ["copy-hash", "copy-subject", "ref:head:feat/x", "ref:remote:origin/main", "ref:tag:v1"]);
  const branch = menu.items.find((i) => i.id === "ref:head:feat/x");
  assert.equal(branch.kind, "sub");
  assert.equal(branch.label, "Branch feat/x");
  assert.equal(branch.state, "Checked out in x.");
  assert.equal(branch.name, "feat/x");
  assert.deepEqual(branch.items.map((i) => i.id), ["open-agent:a1", "copy-branch"]);
  assert.equal(menu.items.find((i) => i.id === "ref:tag:v1").label, "Tag v1");
  assert.equal(menu.items.find((i) => i.id === "ref:remote:origin/main").label, "Remote branch origin/main");
});

test("a row with no pills is just the commit's own actions", () => {
  const menu = graphActions({ kind: "commit", commit: { hash: "c".repeat(40), subject: "s" }, refs: [] }, graph(), ctx);
  assert.deepEqual(menu.items.map((i) => i.id), ["copy-hash", "copy-subject"]);
});

test("a malformed pill is dropped rather than rendered empty", () => {
  const refs = [null, { name: "", kind: "head" }, { name: "x", kind: "nonsense" }];
  const menu = graphActions({ kind: "commit", commit: { hash: "d".repeat(40) }, refs }, graph(), ctx);
  assert.deepEqual(menu.items.map((i) => i.id), ["copy-hash"]);
});
