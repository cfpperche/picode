import test from "node:test";
import assert from "node:assert/strict";
import { folderSections } from "./folders.js";

test("folderSections groups by top-level folder with descendant sums", () => {
  const changes = [
    { path: "web/src/b.js", add: 10, del: 2 },
    { path: "web/src/a.js", add: 5, del: 0 },
    { path: "web/img/logo.png", add: 0, del: 0, binary: true },
    { path: "README.md", add: 1, del: 1 },
    { path: "cmd/main.go", add: 3, del: 3 },
  ];
  const sections = folderSections(changes);
  assert.deepEqual(sections.map((s) => s.dir), ["", "cmd", "web"]);
  assert.equal(sections[0].files.length, 1);
  assert.deepEqual(sections[2].add, 15);
  assert.deepEqual(sections[2].del, 2);
  assert.deepEqual(sections[2].files.map((f) => f.path), ["web/src/a.js", "web/src/b.js", "web/img/logo.png"]);
});

test("folderSections keeps an empty list empty and sums one-file folders", () => {
  assert.deepEqual(folderSections([]), []);
  const single = folderSections([{ path: "main.go", add: 2, del: 1 }]);
  assert.equal(single.length, 1);
  assert.equal(single[0].dir, "");
  assert.equal(single[0].add, 2);
});
