import assert from "node:assert/strict";
import { test } from "node:test";
import { previewKind, isBlobKind, svgDataUrl, previewEmpty } from "./filePreview.js";

test("previewKind covers track 1", () => {
  assert.equal(previewKind("a.svg"), "svg");
  assert.equal(previewKind("a.mmd"), "mermaid");
  assert.equal(previewKind("a.md"), "markdown");
  assert.equal(previewKind("a.png"), "image");
  assert.equal(previewKind("a.pdf"), "pdf");
  assert.equal(previewKind("a.mp3"), "audio");
  assert.equal(previewKind("a.webm"), "video");
  assert.equal(previewKind("a.glb"), "model3d");
  assert.equal(previewKind("a.js"), "");
  assert.equal(isBlobKind("image"), true);
  assert.equal(isBlobKind("markdown"), false);
  assert.equal(svgDataUrl("<svg></svg>").startsWith("data:image/svg+xml"), true);
  assert.equal(previewEmpty("  "), true);
});
