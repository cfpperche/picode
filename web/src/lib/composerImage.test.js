import { sniffImage, MAX_IMAGE_BYTES } from "./composerImage.js";
import assert from "node:assert/strict";
import { test } from "node:test";

test("sniffImage accepts common image types", () => {
  assert.equal(sniffImage({ type: "image/png", name: "a.png", size: 10 }).mime, "image/png");
  assert.equal(sniffImage({ type: "image/jpg", name: "a.jpg", size: 10 }).mime, "image/jpeg");
  assert.equal(sniffImage({ type: "text/plain", name: "a.txt", size: 10 }), null);
  assert.equal(sniffImage(null), null);
});

test("size cap is 4 MB", () => {
  assert.equal(MAX_IMAGE_BYTES, 4 * 1024 * 1024);
});
