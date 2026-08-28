import { fileChangeFromTool, normalizeEdits, parseOfficialDiff, hunksFromDiff, groupHunks, undoHunkInText } from "./diff.js";
import assert from "node:assert/strict";
import { test } from "node:test";

test("edit from edits[]", () => {
  const ch = fileChangeFromTool("edit", {
    path: "/a/b.go",
    edits: [{ oldText: "foo\nbar", newText: "foo\nbaz" }],
  }, null);
  assert.equal(ch.path, "/a/b.go");
  assert.equal(ch.del, 2);
  assert.equal(ch.add, 2);
});

test("legacy top-level oldText/newText", () => {
  assert.deepEqual(normalizeEdits({ oldText: "a", newText: "b" }), [{ oldText: "a", newText: "b" }]);
});

test("write is all additions", () => {
  const ch = fileChangeFromTool("write", { path: "x.md", content: "one\ntwo" }, null);
  assert.equal(ch.add, 2);
  assert.equal(ch.del, 0);
});

test("official patch preferred", () => {
  const ch = fileChangeFromTool("edit", { path: "f", edits: [{ oldText: "a", newText: "b" }] }, {
    details: { patch: "--- a\n+++ b\n@@ -1 +1 @@\n-a\n+b\n" },
  });
  assert.equal(ch.add, 1);
  assert.equal(ch.del, 1);
});

test("parseOfficialDiff ignores headers", () => {
  const p = parseOfficialDiff({ details: { diff: "+++ x\n--- y\n+ok\n-no\n ctx" } });
  assert.equal(p.add, 1);
  assert.equal(p.del, 1);
});

test("hunksFromDiff", () => {
  const p = hunksFromDiff("--- a\n+++ b\n@@ -1 +1 @@\n-old\n+new\n ctx\n");
  assert.equal(p.add, 1);
  assert.equal(p.del, 1);
  assert.equal(p.hunks.filter((h) => h.kind === "ctx" && h.text === "ctx").length, 1);
});

test("non-file tools return null", () => {
  assert.equal(fileChangeFromTool("bash", { command: "ls" }, null), null);
});

test("groupHunks splits on gap and meta", () => {
  const g = groupHunks([
    { kind: "del", text: "old" },
    { kind: "add", text: "new" },
    { kind: "gap", text: "" },
    { kind: "del", text: "a" },
    { kind: "add", text: "b" },
  ]);
  assert.equal(g.length, 2);
  assert.deepEqual(g[0].dels, ["old"]);
  assert.deepEqual(g[0].adds, ["new"]);
});

test("undoHunkInText replaces new with old", () => {
  const g = { dels: ["bar"], adds: ["baz"], ctxBefore: ["foo"], ctxAfter: [] };
  const r = undoHunkInText("foo\nbaz\n", g);
  assert.equal(r.ok, true);
  assert.equal(r.text, "foo\nbar\n");
});

test("undoHunkInText missing new side fails", () => {
  const g = { dels: ["a"], adds: ["zzz"], ctxBefore: [], ctxAfter: [] };
  assert.equal(undoHunkInText("hello\n", g).ok, false);
});

test("undoHunkInText whole write is not applied", () => {
  const g = { dels: [], adds: ["one", "two"], ctxBefore: [], ctxAfter: [] };
  const r = undoHunkInText("one\ntwo", g);
  assert.equal(r.ok, false);
  assert.equal(r.whole, true);
});
