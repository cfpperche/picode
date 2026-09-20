import test from "node:test";
import assert from "node:assert/strict";
import {
  groupChanges, changeTotals, scopeChanges, normalizeTouched,
  compactCount, totalsLabel, sessionGroups, resolveSessionView,
} from "./inspector.js";

// The change-shape logic both apps render (the desktop rail since ADR-0078,
// the mobile Inspector since its screen shipped). Tests moved here with the
// module so neither app can drift.

test("groupChanges builds a folder tree with summed counts", () => {
  const changes = [
    { path: "web/src/b.js", kind: "modified", add: 10, del: 2 },
    { path: "web/src/a.js", kind: "added", add: 5, del: 0 },
    { path: "web/img/logo.png", kind: "modified", add: 0, del: 0, binary: true },
    { path: "README.md", kind: "modified", add: 1, del: 1 },
    { path: "cmd/main.go", kind: "modified", add: 3, del: 3 },
  ];
  const { levels, dirs, stats } = groupChanges(changes);
  assert.deepEqual([...dirs].sort(), ["cmd", "web", "web/img", "web/src"]);
  assert.deepEqual(levels[""].dirs.map((d) => d.name), ["cmd", "web"]);
  assert.deepEqual(levels[""].files.map((f) => f.name), ["README.md"]);
  assert.equal(stats.get("web/src").add, 15);
  assert.equal(stats.get("web/src").files, 2);
  assert.equal(stats.get("web/img").binary, false);
  assert.equal(stats.get("web/img/logo.png").binary, true);
  assert.deepEqual(groupChanges([]).levels, { "": { dirs: [], files: [] } });
});

test("changeTotals, scopeChanges and normalizeTouched agree on root-relative paths", () => {
  const changes = [{ path: "a.js", add: 3, del: 1 }, { path: "src/b.js", add: 0, del: 2 }, { path: "bin.png", binary: true }];
  assert.deepEqual(changeTotals(changes), { add: 3, del: 3, files: 3 });
  assert.equal(scopeChanges(changes, null), changes);
  assert.deepEqual(scopeChanges(changes, new Set(["src/b.js"])).map((c) => c.path), ["src/b.js"]);
  assert.deepEqual(scopeChanges(changes, new Set()), []);
  const touched = normalizeTouched(["./a.js", "/home/u/repo/src/b.js", "/elsewhere/c.js", "src/", "", "../up.js", "/home/u/repo"], "/home/u/repo/");
  assert.deepEqual([...touched].sort(), ["a.js", "src", "src/b.js"]);
  assert.deepEqual([...normalizeTouched(["/abs/x.js"], "")], [], "absolute paths need a root to be relative to");
});

test("compactCount and totalsLabel print the benchmark shape", () => {
  assert.equal(compactCount(684), "684");
  assert.equal(compactCount(1900), "1.9k");
  assert.equal(compactCount(1000), "1k");
  assert.equal(compactCount(12345), "12k");
  assert.equal(compactCount(1250000), "1.3M");
  assert.equal(compactCount(-3), "0");
  assert.equal(totalsLabel({ add: 1900, del: 684 }), "+1.9k −684");
  assert.equal(totalsLabel({ add: 0, del: 0 }), "");
  assert.equal(totalsLabel({ add: 4, del: 0 }), "+4");
  assert.equal(totalsLabel(null), "");
});

const ch = (path) => ({ path, kind: "modified", add: 1, del: 0 });
const wtEntry = (over) => ({ path: "/w/side", ref: "side", branch: "side", changes: [ch("a.txt")], totals: { add: 1, del: 0, files: 1 }, ...over });
const rootStatus = (over) => ({ branch: "main", changes: [], totals: { add: 0, del: 0, files: 0 }, ...over });

test("sessionGroups puts the anchor first and keeps worktree order", () => {
  const groups = sessionGroups(rootStatus({ changes: [ch("m.txt")], worktrees: [wtEntry(), wtEntry({ path: "/w/docs", ref: "docs", branch: "docs" })] }), "/w/app");
  assert.deepEqual(groups.map((g) => g.key), ["root", "wt:/w/side", "wt:/w/docs"]);
  assert.equal(groups[0].path, "/w/app");
  assert.equal(groups[0].isRoot, true);
  assert.equal(groups[1].ref, "side");
  // Entries without a path or a ref cannot be read back through ?worktree=.
  assert.equal(sessionGroups(rootStatus({ worktrees: [{ path: "", ref: "x" }, { path: "/w/y" }, null] }), "/w/app").length, 1);
});

test("resolveSessionView follows one dirty sibling from a clean anchor", () => {
  const groups = sessionGroups(rootStatus({ worktrees: [wtEntry()] }), "/w/app");
  const view = resolveSessionView({ groups });
  assert.equal(view.mode, "follow");
  assert.deepEqual(view.shown.map((g) => g.key), ["wt:/w/side"]);
  assert.equal(view.pill.ref, "side");
  assert.deepEqual(view.switcher, []);
});

test("resolveSessionView keeps single-root views untouched", () => {
  assert.equal(resolveSessionView({ groups: sessionGroups(rootStatus(), "/w/app") }).mode, "root");
  const dirty = resolveSessionView({ groups: sessionGroups(rootStatus({ changes: [ch("m.txt")] }), "/w/app") });
  assert.equal(dirty.mode, "root");
  assert.deepEqual(dirty.shown.map((g) => g.key), ["root"]);
  assert.deepEqual(dirty.switcher, []);
});

test("resolveSessionView dismisses to a switcher and re-follows on View", () => {
  const groups = sessionGroups(rootStatus({ worktrees: [wtEntry()] }), "/w/app");
  const dismissed = resolveSessionView({ groups, dismissed: true });
  assert.equal(dismissed.mode, "root");
  assert.deepEqual(dismissed.shown.map((g) => g.key), ["root"]);
  assert.deepEqual(dismissed.switcher.map((g) => g.ref), ["side"]);
  const again = resolveSessionView({ groups, followedRef: "side", dismissed: true });
  assert.equal(again.mode, "follow");
  assert.equal(again.pill.ref, "side");
});

test("resolveSessionView groups several dirty checkouts", () => {
  const both = sessionGroups(rootStatus({ changes: [ch("m.txt")], worktrees: [wtEntry()] }), "/w/app");
  const view = resolveSessionView({ groups: both });
  assert.equal(view.mode, "groups");
  assert.deepEqual(view.shown.map((g) => g.key), ["root", "wt:/w/side"]);
  assert.equal(view.pill, null);
  const two = sessionGroups(rootStatus({ worktrees: [wtEntry(), wtEntry({ path: "/w/docs", ref: "docs", branch: "docs" })] }), "/w/app");
  const multi = resolveSessionView({ groups: two });
  assert.equal(multi.mode, "groups");
  assert.deepEqual(multi.shown.map((g) => g.key), ["wt:/w/side", "wt:/w/docs"]);
});

test("resolveSessionView falls back when the followed checkout goes clean", () => {
  const groups = sessionGroups(rootStatus({ worktrees: [wtEntry({ changes: [] })] }), "/w/app");
  const view = resolveSessionView({ groups, followedRef: "side" });
  assert.equal(view.mode, "root");
  assert.equal(view.pill, null);
  assert.deepEqual(view.switcher, []);
  const gone = resolveSessionView({ groups: sessionGroups(rootStatus(), "/w/app"), followedRef: "side" });
  assert.equal(gone.mode, "root");
});
