import assert from "node:assert/strict";
import { test } from "node:test";
import { isImageFile, planAttachFiles, MAX_ATTACH, MAX_ATTACH_BYTES } from "./termPrompt.js";

test("isImageFile", () => {
  assert.equal(isImageFile({ type: "image/png", name: "a.png" }), true);
  assert.equal(isImageFile({ type: "", name: "a.PDF" }), false);
  assert.equal(isImageFile({ type: "", name: "x.webp" }), true);
});

test("planAttachFiles", () => {
  const a = { name: "a.png", type: "image/png", size: 10 };
  const big = { name: "b.bin", type: "application/octet-stream", size: MAX_ATTACH_BYTES + 1 };
  assert.equal(planAttachFiles([a], 0).files.length, 1);
  assert.equal(planAttachFiles([big], 0).tooLarge, 1);
  assert.equal(planAttachFiles([a, a, a, a, a], 0).tooMany, true);
  assert.equal(planAttachFiles([a, a, a, a, a], 0).files.length, MAX_ATTACH);
  assert.equal(planAttachFiles([a], 4).tooMany, true);
});
