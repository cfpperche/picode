import { fileChangeFromTool, normalizeEdits, parseOfficialDiff } from "./diff.js";
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

test("non-file tools return null", () => {
  assert.equal(fileChangeFromTool("bash", { command: "ls" }, null), null);
});
