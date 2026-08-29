import assert from "node:assert/strict";
import { test } from "node:test";
import { previewKind, svgDataUrl, previewEmpty } from "./filePreview.js";

test("previewKind svg and mermaid only", () => {
  assert.equal(previewKind("a.svg"), "svg");
  assert.equal(previewKind("x.MMD"), "mermaid");
  assert.equal(previewKind("d.mermaid"), "mermaid");
  assert.equal(previewKind("a.go"), "");
  assert.equal(previewKind("a.md"), "");
  assert.equal(previewKind("a.png"), "");
});

test("svgDataUrl", () => {
  assert.equal(svgDataUrl(""), "");
  assert.equal(svgDataUrl("not svg"), "");
  const u = svgDataUrl('<svg xmlns="http://www.w3.org/2000/svg"></svg>');
  assert.ok(u.startsWith("data:image/svg+xml"));
  assert.equal(previewEmpty("  \n"), true);
  assert.equal(previewEmpty("<svg></svg>"), false);
});
