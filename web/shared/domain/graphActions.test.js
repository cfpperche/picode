import test from "node:test";
import assert from "node:assert/strict";
import { graphActions, repoActions, repoOccupants, busyLine, trackingLabel, trackingTitle, worktreeSlug, confirmPhrase, gateFor } from "./graphActions.js";

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
// Sections, not submenus: every row is one click, and the label that gives a
// row its meaning stays on screen beside it.
test("the row menu lays each pill out as a labelled section", () => {
  const g = graph();
  const menu = graphActions(
    { kind: "commit", commit: { hash: "b".repeat(40), subject: "work" }, refs: [g.refs[1], g.refs[3], g.refs[4]] },
    g,
    ctx,
  );
  assert.deepEqual(menu.items.map((i) => i.id), [
    "copy-hash", "copy-subject",
    "section:ref:head:feat/x", "open-agent:a1", "copy-branch",
    "section:ref:remote:origin/main", "copy-branch",
    "section:ref:tag:v1", "copy-tag",
  ]);
  const branch = menu.items.find((i) => i.id === "section:ref:head:feat/x");
  assert.equal(branch.kind, "section");
  assert.equal(branch.label, "Branch feat/x");
  assert.equal(branch.state, "Checked out in x.");
  assert.ok(menu.items.every((i) => i.kind !== "sub"), "no menu of this graph nests");
});

test("a row with no pills is just the commit's own actions", () => {
  const menu = graphActions({ kind: "commit", commit: { hash: "c".repeat(40), subject: "s" }, refs: [] }, graph(), ctx);
  assert.deepEqual(menu.items.map((i) => i.id), ["copy-hash", "copy-subject"]);
});

// Nothing in this graph's menus nests: a submenu puts the leaf a hover away
// from the label that explains it, and adds a layer for rows that are one
// click each.
test("no menu this module builds contains a submenu", () => {
  const g = graph();
  for (const t of [
    { kind: "commit", commit: { hash: "z".repeat(40), subject: "s" }, refs: g.refs },
    { kind: "ref", ref: g.refs[0] },
    { kind: "ref", ref: g.refs[3] },
    { kind: "worktree", worktree: g.worktrees[1], uncommitted: true },
  ]) {
    assert.ok(graphActions(t, g, wctx).items.every((i) => i.kind !== "sub"), `${t.kind} nests`);
  }
  assert.ok(repoActions(g, wctx).items.every((i) => i.kind !== "sub"));
});

test("a malformed pill is dropped rather than rendered empty", () => {
  const refs = [null, { name: "", kind: "head" }, { name: "x", kind: "nonsense" }];
  const menu = graphActions({ kind: "commit", commit: { hash: "d".repeat(40) }, refs }, graph(), ctx);
  assert.deepEqual(menu.items.map((i) => i.id), ["copy-hash"]);
});

// --- Write actions (phases 2-4) -------------------------------------------

// The catalog the server serves. Only these three fields matter here.
const catalog = {};
for (const [id, tier, needs] of [
  ["fetch", "A", []], ["fetch-into-local", "A", ["target", "name"]], ["pull", "B", []],
  ["pull-remote", "B", ["target"]], ["checkout", "A", ["target"]], ["checkout-detach", "A", ["target"]],
  ["checkout-remote", "A", ["target", "name"]], ["create-branch", "A", ["target", "name"]],
  ["create-tag", "A", ["target", "name"]], ["create-worktree", "A", ["target", "name"]],
  ["prune-worktrees", "A", []], ["merge", "B", ["target"]], ["rebase", "B", ["target"]],
  ["cherry-pick", "B", ["target"]], ["revert", "B", ["target"]], ["reset-soft", "B", ["target"]],
  ["reset-mixed", "B", ["target"]], ["rename-branch", "B", ["target", "name"]],
  ["delete-branch", "B", ["target"]], ["commit", "B", ["message"]], ["pr", "B", []],
  ["push", "C", []], ["push-force", "C", []], ["commit-push", "C", ["message"]],
  ["push-tag", "C", ["target"]], ["delete-branch-force", "C", ["target"]],
  ["delete-remote-branch", "C", ["target"]], ["delete-tag", "C", ["target"]],
  ["reset-hard", "C", ["target"]], ["discard", "C", []], ["clean", "C", []],
  ["worktree-remove", "C", ["name"]], ["worktree-remove-force", "C", ["name"]],
]) catalog[id] = { id, tier, needs };

const wctx = { ...ctx, catalog };
// A menu's actions may sit one level down in a group; a reader reaches them
// either way, so the assertions look through both.
const ids = (menu) => menu.items.filter((i) => i.kind === "action").map((i) => i.action);

// Fail closed: a client that has not read the catalog cannot render a tier,
// so it renders no write action at all.
test("without the catalog there are no write actions", () => {
  const menu = graphActions({ kind: "commit", commit: { hash: "z".repeat(40), subject: "s" } }, graph(), ctx);
  assert.deepEqual(ids(menu), []);
});

// Walk into the groups: a tier must be the server's wherever the row sits.
const actionRows = (menu) => menu.items.filter((i) => i.kind === "action");

test("the tier on a row is the server's, never invented here", () => {
  const menu = graphActions({ kind: "commit", commit: { hash: "z".repeat(40), subject: "s" } }, graph(), wctx);
  const byAction = Object.fromEntries(actionRows(menu).map((i) => [i.action, i.tier]));
  assert.equal(byAction["reset-hard"], "C");
  assert.equal(byAction["merge"], "B");
  assert.equal(byAction["create-branch"], "A");
  // An action the catalog has never heard of is not rendered at all.
  const thin = graphActions({ kind: "commit", commit: { hash: "z".repeat(40) } }, graph(), { ...ctx, catalog: { merge: { tier: "B", needs: ["target"] } } });
  assert.deepEqual(ids(thin), ["merge"]);
});

// Merging or rebasing onto the commit you are already on is a no-op git
// reports as "Already up to date". A row that can only do nothing is absent.
test("the HEAD commit offers no merge, rebase, cherry-pick or detached checkout", () => {
  const g = graph();
  const head = graphActions({ kind: "commit", commit: { hash: g.head, subject: "s" } }, g, wctx);
  for (const gone of ["merge", "rebase", "cherry-pick", "checkout-detach"]) {
    assert.ok(!ids(head).includes(gone), `${gone} must not be offered on HEAD`);
  }
  assert.ok(ids(head).includes("create-branch"));
  const other = graphActions({ kind: "commit", commit: { hash: "z".repeat(40), subject: "s" } }, g, wctx);
  for (const there of ["merge", "rebase", "cherry-pick", "checkout-detach"]) {
    assert.ok(ids(other).includes(there), `${there} must be offered on another commit`);
  }
});

// git refuses to switch to, or delete, a branch another worktree holds. The
// menu does not offer it in the first place.
test("a branch held by another checkout offers no switch and no delete", () => {
  const menu = graphActions({ kind: "ref", ref: graph().refs[1] }, graph(), wctx);
  for (const gone of ["checkout", "create-worktree", "delete-branch", "delete-branch-force"]) {
    assert.ok(!ids(menu).includes(gone), `${gone} must not be offered for a checked-out branch`);
  }
  assert.ok(ids(menu).includes("merge"));
  assert.ok(ids(menu).includes("rebase"));
});

test("a branch no checkout holds offers switch, worktree and the right delete", () => {
  const g = graph();
  const merged = graphActions({ kind: "ref", ref: g.refs[2] }, g, wctx);
  assert.ok(ids(merged).includes("checkout"));
  assert.ok(ids(merged).includes("create-worktree"));
  assert.ok(ids(merged).includes("delete-branch"), "a merged branch deletes with -d");
  assert.ok(!ids(merged).includes("delete-branch-force"));

  const unmerged = graphActions({ kind: "ref", ref: { ...g.refs[2], merged: false } }, g, wctx);
  assert.ok(ids(unmerged).includes("delete-branch-force"), "an unmerged branch needs the force twin");
  assert.ok(!ids(unmerged).includes("delete-branch"));
});

// `git push` publishes the checked-out branch, so it belongs only to that one.
test("push is offered on the current branch and nowhere else", () => {
  const g = graph();
  const current = graphActions({ kind: "ref", ref: g.refs[0] }, g, wctx);
  assert.ok(ids(current).includes("push"));
  assert.ok(ids(current).includes("push-force"));
  assert.ok(!ids(current).includes("merge"), "a branch cannot merge into itself");
  const other = graphActions({ kind: "ref", ref: g.refs[1] }, g, wctx);
  assert.ok(!ids(other).includes("push"));
});

test("a repository with no remote offers nothing that needs one", () => {
  const g = { ...graph(), remotes: [] };
  const current = graphActions({ kind: "ref", ref: g.refs[0] }, g, wctx);
  assert.ok(!ids(current).includes("push"));
  const tag = graphActions({ kind: "ref", ref: g.refs[4] }, g, wctx);
  assert.ok(!ids(tag).includes("push-tag"));
  assert.ok(ids(tag).includes("delete-tag"), "a local tag can still be deleted");
  const repo = repoActions(g, wctx);
  assert.deepEqual(ids(repo), ["create-branch"]);
});

// A remote branch that already has a local twin has nothing to create.
test("a remote branch offers to create a local one only when there is none", () => {
  const g = graph();
  const fresh = graphActions({ kind: "ref", ref: { name: "origin/feat-new", kind: "remote", hash: "eee" } }, g, wctx);
  assert.ok(ids(fresh).includes("checkout-remote"));
  assert.ok(ids(fresh).includes("fetch-into-local"));
  const twinned = graphActions({ kind: "ref", ref: g.refs[3] }, g, wctx);
  assert.ok(!ids(twinned).includes("checkout-remote"), "main already exists locally");
  assert.ok(ids(twinned).includes("pull-remote"));
  assert.ok(ids(twinned).includes("delete-remote-branch"));
});

// The command runs in a terminal sitting in the reader's own folder, so only
// that checkout can be committed from here. A sibling's row keeps its agent.
test("only the reader's own dirty row can commit, discard or clean", () => {
  const g = graph();
  const own = graphActions({ kind: "worktree", worktree: g.worktrees[0], uncommitted: true }, g, wctx);
  for (const there of ["commit", "commit-push", "discard", "clean"]) {
    assert.ok(ids(own).includes(there), `${there} must be offered on the reader's own row`);
  }
  const sibling = graphActions({ kind: "worktree", worktree: g.worktrees[1], uncommitted: true }, g, wctx);
  for (const gone of ["commit", "commit-push", "discard", "clean"]) {
    assert.ok(!ids(sibling).includes(gone), `${gone} must not act on a sibling worktree`);
  }
  assert.ok(sibling.items.some((i) => i.kind === "open-agent"), "the sibling keeps its own door");
});

// The composer only builds `.worktrees/<name>`, so a checkout anywhere else
// is not offered for removal rather than removed by a guessed path.
test("only a sibling under .worktrees/ can be removed, and never the reader's own", () => {
  const g = graph();
  const sibling = graphActions({ kind: "worktree", worktree: g.worktrees[1] }, g, wctx);
  assert.ok(ids(sibling).includes("worktree-remove"));
  const item = sibling.items.find((i) => i.action === "worktree-remove");
  assert.equal(item.name, "x", "the row carries the folder name the command needs");

  const own = graphActions({ kind: "worktree", worktree: g.worktrees[0] }, g, wctx);
  assert.ok(!ids(own).includes("worktree-remove"), "the checkout you are reading through is not removable from itself");

  const elsewhere = { path: "/somewhere/else/tree", head: "fff", branch: "odd", agents: [] };
  const outside = graphActions({ kind: "worktree", worktree: elsewhere }, g, wctx);
  assert.ok(!ids(outside).includes("worktree-remove"), "a checkout outside .worktrees/ has no composable command");
});

test("worktreeSlug reads the one segment under .worktrees/", () => {
  const g = graph();
  assert.equal(worktreeSlug(g.worktrees[1], g), "x");
  assert.equal(worktreeSlug(g.worktrees[0], g), "");
  assert.equal(worktreeSlug({ path: "/repo/.worktrees/a/b" }, g), "", "a nested path is not a slug");
  assert.equal(worktreeSlug({ path: "/elsewhere/x" }, g), "");
  assert.equal(worktreeSlug(null, g), "");
});

// A graph read from inside a worktree still resolves its siblings, because
// the repository root is one level above .worktrees/.
test("worktreeSlug works when the reader is itself in a worktree", () => {
  const g = {
    worktrees: [
      { path: "/repo/.worktrees/me", self: true },
      { path: "/repo/.worktrees/other" },
    ],
  };
  assert.equal(worktreeSlug(g.worktrees[1], g), "other");
});

test("every write row carries what the caller needs to compose it", () => {
  const menu = graphActions({ kind: "commit", commit: { hash: "z".repeat(40), subject: "s" } }, graph(), wctx);
  for (const item of actionRows(menu)) {
    assert.ok(item.action && item.label && item.tier, `incomplete row: ${JSON.stringify(item)}`);
    assert.ok(Array.isArray(item.needs));
    if (item.needs.includes("target")) assert.equal(item.target, "z".repeat(40));
  }
});

test("the repository header offers what needs no target", () => {
  const menu = repoActions(graph(), wctx);
  assert.deepEqual(ids(menu), ["fetch", "pull", "push", "pr", "create-branch"]);
  assert.equal(menu.busy, "Atlas is mid-turn in this repository.");
});

// --- Gates ----------------------------------------------------------------

// The prepare door's confirmation is the human pressing Enter in their own
// prompt. Only the run door, where PiCode presses it, borrows a typed one.
test("only a tier C action delivered by PiCode itself asks for a typed phrase", () => {
  assert.equal(gateFor({ tier: "C", door: "prepare", action: "reset-hard" }).typed, "");
  assert.equal(gateFor({ tier: "C", door: "ask", action: "reset-hard" }).typed, "");
  assert.equal(gateFor({ tier: "B", door: "run", action: "merge", target: "feat/x" }).typed, "");
  assert.equal(gateFor({ tier: "A", door: "run", action: "fetch" }).typed, "");
  assert.equal(gateFor({ tier: "C", door: "run", action: "reset-hard" }).typed, "discard");
});

test("the phrase is always something already on the dialog", () => {
  assert.equal(confirmPhrase("delete-branch-force", { target: "feat/x" }), "feat/x");
  assert.equal(confirmPhrase("delete-remote-branch", { target: "origin/feat-x" }), "origin/feat-x");
  assert.equal(confirmPhrase("delete-tag", { target: "v1" }), "v1");
  assert.equal(confirmPhrase("worktree-remove-force", { name: "x" }), "x");
  assert.equal(confirmPhrase("push-force", { branch: "main" }), "main");
  assert.equal(confirmPhrase("push", { branch: "main" }), "main");
  assert.equal(confirmPhrase("commit-push", { branch: "main" }), "main");
  assert.equal(confirmPhrase("push-tag", { target: "v1" }), "v1");
  for (const a of ["reset-hard", "discard", "clean"]) {
    assert.equal(confirmPhrase(a, {}), "discard");
  }
  assert.equal(confirmPhrase("merge", { target: "feat/x" }), "", "a tier B action needs no phrase");
});

// Every tier C action must have a phrase, or the gate would be a no-op that
// silently lets PiCode press Enter on a destructive command.
test("every tier C action has a phrase to type", () => {
  const args = { target: "origin/feat-x", name: "x", branch: "main" };
  for (const [id, info] of Object.entries(catalog)) {
    if (info.tier !== "C") continue;
    assert.notEqual(gateFor({ tier: "C", door: "run", action: id, ...args }).typed, "",
      `tier C action ${id} has no typed phrase, so the run gate would be empty`);
  }
});

// The ask door needs the agents that live here, so every menu carries them.
test("every menu carries the occupants the ask door addresses", () => {
  for (const t of [
    { kind: "commit", commit: { hash: "z".repeat(40) } },
    { kind: "ref", ref: graph().refs[0] },
    { kind: "worktree", worktree: graph().worktrees[1] },
    { kind: "nonsense" },
  ]) {
    const menu = graphActions(t, graph(), wctx);
    assert.ok(Array.isArray(menu.occupants), `no occupants on ${t.kind}`);
  }
  assert.deepEqual(graphActions({ kind: "commit", commit: { hash: "z".repeat(40) } }, graph(), wctx).occupants.map((o) => o.name), ["Atlas"]);
  assert.deepEqual(repoActions(graph(), wctx).occupants.map((o) => o.name), ["Atlas"]);
});

// A menu that runs off the bottom of the window is a clipped overlay, which
// the visual gate refuses outright. Grouping is what keeps the top level short.
test("a commit's write actions arrive in labelled sections", () => {
  const menu = graphActions({ kind: "commit", commit: { hash: "z".repeat(40), subject: "s" }, refs: [] }, graph(), wctx);
  assert.deepEqual(menu.items.filter((i) => i.kind === "section").map((i) => i.label), ["Create", "Apply", "Move this branch"]);
  // Every section is followed by at least one row; a label over nothing is
  // the empty well the UI bar refuses.
  const kinds = menu.items.map((i) => i.kind);
  kinds.forEach((kind, i) => {
    if (kind === "section") assert.equal(kinds[i + 1], "action", `${menu.items[i].label} labels nothing`);
  });
});

test("the HEAD commit's groups drop what would be a no-op but keep the rest", () => {
  const g = graph();
  const menu = graphActions({ kind: "commit", commit: { hash: g.head, subject: "s" }, refs: [] }, g, wctx);
  const groups = {};
  let current = "";
  for (const item of menu.items) {
    if (item.kind === "section") { current = item.label; groups[current] = []; }
    else if (item.kind === "action" && current) groups[current].push(item.action);
  }
  assert.deepEqual(groups["Apply"], ["revert"]);
  assert.ok(!groups["Move this branch"].includes("checkout-detach"));
  assert.ok(groups["Create"].includes("create-branch"));
});
