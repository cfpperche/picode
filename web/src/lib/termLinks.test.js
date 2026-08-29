import assert from "node:assert/strict";
import { test } from "node:test";
import {
  hasOpenModifier, stripLineCol, classify, findLinks, underCwd, relPath,
} from "./termLinks.js";

const cwd = "/home/goat/picode";

test("hasOpenModifier is ctrl or meta only", () => {
  assert.equal(hasOpenModifier({ ctrlKey: true }), true);
  assert.equal(hasOpenModifier({ metaKey: true }), true);
  assert.equal(hasOpenModifier({ altKey: true }), false);
  assert.equal(hasOpenModifier({}), false);
});

test("stripLineCol drops :line and :line:col", () => {
  assert.equal(stripLineCol("foo.go:12"), "foo.go");
  assert.equal(stripLineCol("foo.go:1-120"), "foo.go");
  assert.equal(stripLineCol("foo.go:12:3"), "foo.go");
  assert.equal(stripLineCol("foo.go."), "foo.go");
});

test("classify decision table", () => {
  const rows = [
    { raw: "https://example.com/a", want: { kind: "http", href: "https://example.com/a" } },
    { raw: "http://localhost:8445/", want: { kind: "http" } },
    { raw: "javascript:alert(1)", want: null },
    { raw: "/etc/passwd", want: null },
    { raw: cwd + "/web/src/a.js", want: { kind: "file", path: "web/src/a.js" } },
    { raw: "~/picode/web/a.js", want: { kind: "file", path: "~/picode/web/a.js" } },
    { raw: "web/src/a.js", want: { kind: "file", path: "web/src/a.js" } },
    { raw: "./web/a.js", want: { kind: "file", path: "web/a.js" } },
    { raw: "file://" + cwd + "/README.md", want: { kind: "file", path: "README.md" } },
    { raw: "web/src/a.js:12", want: { kind: "file", path: "web/src/a.js" } },
    { raw: "../secret", want: null },
    { raw: "/", want: null },
  ];
  for (const r of rows) {
    const got = classify(r.raw, cwd);
    if (r.want === null) {
      assert.equal(got, null, r.raw);
      continue;
    }
    assert.ok(got, r.raw);
    assert.equal(got.kind, r.want.kind, r.raw);
    if (r.want.path) assert.equal(got.path, r.want.path, r.raw);
    if (r.want.href) assert.equal(got.href, r.want.href, r.raw);
  }
});

test("findLinks skips http when scanning paths and keeps tool-call paths", () => {
  const line = "read ~/picode/web/src/a.js  see https://example.com/x  and /etc/passwd";
  const hits = findLinks(line, cwd);
  assert.equal(hits.some((h) => h.kind === "http"), true);
  assert.equal(hits.some((h) => h.kind === "file" && h.path.startsWith("~/")), true);
  assert.equal(hits.some((h) => String(h.raw).includes("/etc/passwd")), false);
});

test("underCwd / relPath", () => {
  assert.equal(underCwd(cwd, cwd + "/web"), true);
  assert.equal(underCwd(cwd, "/etc/passwd"), false);
  assert.equal(relPath(cwd, cwd + "/web/a.js"), "web/a.js");
});
