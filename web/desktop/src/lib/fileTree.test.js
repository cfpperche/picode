import { test } from "node:test";
import assert from "node:assert/strict";
import {
  treeApiBase,
  provisionalTreeKey,
  mergeLevel,
  flattenTree,
  changeKinds,
  changedDirs,
  fitTreeWidth,
  treeKeyAction,
} from "./fileTree.js";

test("treeApiBase picks the owner's route family", () => {
  assert.equal(treeApiBase("term"), "/api/terminals/");
  assert.equal(treeApiBase("workspace"), "/api/workspaces/");
  assert.equal(treeApiBase("agent"), "/api/agents/");
});

test("tree width reserves space for the file on narrow desktops", () => {
  assert.equal(fitTreeWidth(320, 1000), 320);
  assert.equal(fitTreeWidth(720, 700), 440);
  assert.equal(fitTreeWidth(320, 450), 190);
  assert.equal(fitTreeWidth(320, 300), 160);
});

test("tree keyboard follows the visible hierarchy, not hidden descendants", () => {
  const rows = [
    { path: "src", isDir: true, open: true, depth: 0 },
    { path: "src/a.go", depth: 1 },
    { path: "test", isDir: true, open: false, depth: 0 },
    { path: "README.md", depth: 0 },
  ];
  for (const [index, key, want] of [
    [0, "ArrowDown", { focus: "src/a.go" }], [0, "ArrowUp", { focus: "src" }],
    [3, "Home", { focus: "src" }], [0, "End", { focus: "README.md" }],
    [0, "ArrowRight", { focus: "src/a.go" }], [0, "ArrowLeft", { toggle: "src" }],
    [1, "ArrowLeft", { focus: "src" }], [2, "ArrowRight", { toggle: "test" }],
    [3, "ArrowRight", null], [3, "ArrowLeft", null],
  ]) assert.deepEqual(treeKeyAction(rows, index, key), want, `${index} ${key}`);
});

test("provisionalTreeKey names the owner until the root arrives", () => {
  assert.equal(provisionalTreeKey("term", "t1"), "@t:t1");
  assert.equal(provisionalTreeKey("workspace", "w1"), "@w:w1");
  assert.equal(provisionalTreeKey("agent", "a1"), "@a:a1");
});

test("mergeLevel caches one browse answer per dir, root under empty key", () => {
  let levels = mergeLevel({}, { dir: "", dirs: [{ name: "src", path: "src" }], files: [] });
  levels = mergeLevel(levels, { dir: "src", dirs: [], files: [{ name: "a.js", path: "src/a.js" }] });
  assert.deepEqual(Object.keys(levels).sort(), ["", "src"]);
  assert.equal(levels["src"].files[0].path, "src/a.js");
});

test("flattenTree walks expanded dirs only, dirs before files", () => {
  const levels = {
    "": { dirs: [{ name: "src", path: "src" }, { name: "web", path: "web" }], files: [{ name: "go.mod", path: "go.mod" }] },
    src: { dirs: [], files: [{ name: "a.go", path: "src/a.go" }] },
  };
  const rows = flattenTree(levels, new Set(["src"]));
  assert.deepEqual(rows.map((r) => r.path), ["src", "src/a.go", "web", "go.mod"]);
  assert.equal(rows[0].open, true);
  assert.equal(rows[0].loaded, true);
  assert.equal(rows[1].depth, 1);
  assert.equal(rows[2].open, false);
});

test("flattenTree marks an expanded-but-unfetched dir as not loaded", () => {
  const levels = { "": { dirs: [{ name: "src", path: "src" }], files: [] } };
  const rows = flattenTree(levels, new Set(["src"]));
  assert.equal(rows[0].loaded, false);
  assert.equal(rows.length, 1);
});

test("an expanded empty directory is distinguishable from a pending read", () => {
  const levels = { "": { dirs: [{ name: "src", path: "src" }], files: [] }, src: { dirs: [], files: [] } };
  assert.equal(flattenTree(levels, new Set(["src"]))[0].empty, true);
  assert.equal(flattenTree(levels, new Set())[0].empty, false);
});

test("changeKinds maps path to kind with a modified fallback", () => {
  const kinds = changeKinds([{ path: "a", kind: "untracked" }, { path: "b" }]);
  assert.equal(kinds.get("a"), "untracked");
  assert.equal(kinds.get("b"), "modified");
});

test("changedDirs yields every ancestor of a changed file", () => {
  const dirs = changedDirs([{ path: "a/b/c.go" }, { path: "top.go" }]);
  assert.deepEqual([...dirs].sort(), ["a", "a/b"]);
});
