import test from "node:test";
import assert from "node:assert/strict";
import { gitURL, isMoved, rootProblem, worktreeRef } from "./git/model.js";
import { groupBranches, visibleRefs, walkParams, resolveSelection, matchCommits } from "./git/history.js";
import { rowGeometry, measuredBranchPath } from "./git/geometry.js";

const asURL = value => new URL(value, "http://mobile.test");

// Decision table: every owner uses its own resource namespace, while root
// stays an equality precondition on every read after the initial browse.
for (const [kind, namespace] of [["workspace", "workspaces"], ["agent", "agents"], ["term", "terminals"]]) {
  for (const resource of ["browse", "gitstatus", "gitdiff", "git", "git/commit", "git/blob", "blob", "pr"]) {
    test(`${kind} ${resource} preserves owner and canonical root`, () => {
      const owner = { kind, id: "owner/a?b#c" };
      const root = "/repo/space & # plus+ ü";
      const url = asURL(gitURL(owner, resource, { root }));
      assert.equal(url.pathname, `/api/${namespace}/owner%2Fa%3Fb%23c/${resource}`);
      assert.equal(url.searchParams.get("root"), root);
      assert.equal(url.hash, "");
      assert.deepEqual([...url.searchParams.keys()], ["root"]);
    });
  }
}

test("initial browse discovers a root without supplying an alternate directory", () => {
  const url = asURL(gitURL({ kind: "workspace", id: "ws" }, "browse"));
  assert.equal(url.pathname, "/api/workspaces/ws/browse");
  assert.equal(url.search, "");
});

for (const owner of [null, {}, { kind: "workspace" }, { kind: "unknown", id: "a" }, { kind: "term", id: "" }]) {
  test(`invalid owner cannot form a Git endpoint: ${JSON.stringify(owner)}`, () => assert.equal(gitURL(owner, "git", { root: "/repo" }), ""));
}

for (const [name, wt, expected] of [
  ["branch checkout", { path: "/siblings/feature", branch: "feat/mobile", head: "a".repeat(40) }, "feat/mobile"],
  ["detached checkout", { path: "/siblings/detached", head: "a".repeat(40) }, "a".repeat(40)],
  ["SHA-256 detached checkout", { path: "/siblings/detached", head: "b".repeat(64) }, "b".repeat(64)],
  ["bare repository", { path: "/bare", branch: "main", bare: true }, ""],
  ["prunable checkout", { path: "/gone", branch: "main", prunable: true }, ""],
  ["filesystem path only", { path: "/siblings/feature" }, ""],
  ["missing worktree", null, ""],
]) {
  test(`sibling read selector: ${name}`, () => assert.equal(worktreeRef(wt), expected));
}

for (const resource of ["gitstatus", "gitdiff", "git/blob", "blob"]) {
  test(`sibling ${resource} keeps the owner namespace and resolves a ref with its pinned path`, () => {
    const owner = { kind: "agent", id: "original-owner" };
    const wt = { path: "/siblings/feature", branch: "feat/review&root=/outside", head: "a".repeat(40) };
    const path = "images/a #1+ü.png";
    const url = asURL(gitURL(owner, resource, { root: wt.path, worktree: worktreeRef(wt), path, hash: "HEAD" }));
    assert.equal(url.pathname, `/api/agents/original-owner/${resource}`);
    assert.deepEqual(url.searchParams.getAll("root"), [wt.path]);
    assert.equal(url.searchParams.get("worktree"), wt.branch);
    assert.notEqual(url.searchParams.get("worktree"), wt.path);
    assert.equal(url.searchParams.get("path"), path);
    assert.equal(url.searchParams.get("hash"), "HEAD");
    const ownerRead = asURL(gitURL(owner, "gitstatus", { root: "/repo" }));
    assert.equal(ownerRead.searchParams.get("root"), "/repo");
    assert.equal(ownerRead.searchParams.has("worktree"), false);
  });
}

for (const [name, selected, remotes, expectedBranches, expectedRemotes] of [
  ["all branches", [], true, [], null],
  ["local branches only", [], false, [], "0"],
  ["explicit branch", ["main"], false, ["main"], null],
  ["multiple branches", ["main", "origin/feat&root=/elsewhere"], true, ["main", "origin/feat&root=/elsewhere"], null],
  ["empty selections ignored", ["", "main", ""], false, ["main"], null],
]) {
  test(`history branch request: ${name}`, () => {
    const url = asURL(gitURL({ kind: "workspace", id: "ws" }, "git", { root: "/repo", limit: 200, ...walkParams(selected, remotes) }));
    assert.deepEqual(url.searchParams.getAll("branches"), expectedBranches);
    assert.equal(url.searchParams.get("remotes"), expectedRemotes);
    assert.equal(url.searchParams.get("limit"), "200");
    assert.deepEqual(url.searchParams.getAll("root"), ["/repo"]);
  });
}

test("PR cache bypass is explicit and retains the root precondition", () => {
  const owner = { kind: "term", id: "t" };
  const cached = asURL(gitURL(owner, "pr", { root: "/repo" }));
  const fresh = asURL(gitURL(owner, "pr", { root: "/repo", refresh: true }));
  assert.equal(cached.searchParams.has("refresh"), false);
  assert.equal(fresh.searchParams.get("refresh"), "1");
  assert.equal(fresh.searchParams.get("root"), cached.searchParams.get("root"));
});

for (const [name, error, moved] of [
  ["root conflict with current path", { status: 409, body: { cwd: "/new" } }, true],
  ["root conflict message", { status: 409, message: "The root changed" }, true],
  ["moved conflict message", { status: 409, message: "This owner moved" }, true],
  ["busy conflict", { status: 409, message: "Agent is busy" }, false],
  ["network error", { message: "Network error" }, false],
  ["missing owner", { status: 404, message: "owner not found" }, false],
  ["non-conflict mentioning root", { status: 500, message: "Cannot read root", body: { cwd: "/new" } }, false],
]) {
  test(`root recovery classification: ${name}`, () => assert.equal(isMoved(error), moved));
}

test("root recovery exposes the new folder without replacing the pinned root in a URL", () => {
  const error = { status: 409, body: { cwd: "/new" }, message: "stale root" };
  assert.ok(rootProblem(error).includes("/new"));
  assert.equal(asURL(gitURL({ kind: "agent", id: "a" }, "git", { root: "/original" })).searchParams.get("root"), "/original");
});

test("branch options exclude tags, separate remotes, and discard deleted selections", () => {
  const refs = [
    { name: "main", kind: "head" }, { name: "feature", kind: "head" },
    { name: "origin/main", kind: "remote" }, { name: "v1.0", kind: "tag" },
  ];
  assert.deepEqual(groupBranches(refs, true), { local: ["feature", "main"], remote: ["origin/main"] });
  assert.deepEqual(groupBranches(refs, false), { local: ["feature", "main"], remote: [] });
  assert.deepEqual(visibleRefs(refs, false).map(r => r.name), ["main", "feature"]);
  assert.deepEqual(resolveSelection(["main", "deleted", "v1.0", "origin/main"], refs), ["main", "origin/main"]);
});

test("history search identifies loaded rows without removing topology inputs", () => {
  const commits = [
    { hash: "ab1234", subject: "Save mobile edits", author: "Alex" },
    { hash: "cd5678", subject: "Read repository", author: "Morgan" },
    { hash: "ef9012", subject: "Merge changes", author: "Alex" },
  ];
  const before = structuredClone(commits);
  for (const [query, hashes] of [[" mobile ", ["ab1234"]], ["MORGAN", ["cd5678"]], ["ef90", ["ef9012"]], ["Alex", ["ab1234", "ef9012"]], ["a", []], ["not loaded", []]]) {
    assert.deepEqual([...matchCommits(commits, query)], hashes);
  }
  assert.deepEqual(commits, before);
});

test("wrapped commit rows keep dots centered and ancestry edges attached", () => {
  const geometry = rowGeometry([48, 96, 60]);
  assert.deepEqual(geometry, { centers: [24, 96, 174], height: 204 });
  const lines = [{ p1: { x: 0, y: 0 }, p2: { x: 1, y: 1 } }, { p1: { x: 1, y: 1 }, p2: { x: 1, y: 2 } }];
  const path = measuredBranchPath(lines, geometry);
  assert.ok(path.startsWith("M10,24"));
  assert.ok(path.includes("22,96"));
  assert.ok(path.endsWith("L22,174"));
  assert.equal((path.match(/M/g) || []).length, 1, "connected ancestry stays one continuous path");
  const resized = measuredBranchPath(lines, rowGeometry([48, 144, 60]));
  assert.ok(resized.includes("22,120"));
  assert.ok(resized.endsWith("L22,222"));
});

test("history continuation extends below the loaded rows and disconnected edges do not join", () => {
  const geometry = rowGeometry([40, 80]);
  const lines = [{ p1: { x: 0, y: 0 }, p2: { x: 0, y: 1 } }, { p1: { x: 2, y: 1 }, p2: { x: 2, y: 2 } }];
  const path = measuredBranchPath(lines, geometry);
  assert.equal(path, "M10,20L10,80M34,80L34,132");
  assert.deepEqual(rowGeometry([]), { centers: [], height: 0 });
  assert.equal(measuredBranchPath([], geometry), "");
});
