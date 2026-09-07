import { sniffImage, planDeviceImages, MAX_IMAGE_BYTES, MAX_IMAGES, sceneHasInk } from "./composerImage.js";
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

test("planDeviceImages decision table", () => {
  const png = { type: "image/png", name: "a.png", size: 10 };
  const jpg = { type: "image/jpeg", name: "b.jpg", size: 20 };
  const txt = { type: "text/plain", name: "c.txt", size: 10 };
  const huge = { type: "image/png", name: "big.png", size: MAX_IMAGE_BYTES + 1 };
  const rows = [
    { files: [png], already: 0, want: { n: 1, notImage: 0, tooLarge: 0, tooMany: false } },
    { files: [txt], already: 0, want: { n: 0, notImage: 1, tooLarge: 0, tooMany: false } },
    { files: [png, txt], already: 0, want: { n: 1, notImage: 1, tooLarge: 0, tooMany: false } },
    { files: [huge], already: 0, want: { n: 0, notImage: 0, tooLarge: 1, tooMany: false } },
    { files: [png, jpg, png, jpg, png], already: 0, want: { n: MAX_IMAGES, notImage: 0, tooLarge: 0, tooMany: true } },
    { files: [png, jpg], already: 3, want: { n: 1, notImage: 0, tooLarge: 0, tooMany: true } },
    { files: [png], already: 4, want: { n: 0, notImage: 0, tooLarge: 0, tooMany: true } },
    { files: [], already: 0, want: { n: 0, notImage: 0, tooLarge: 0, tooMany: false } },
  ];
  for (const row of rows) {
    const got = planDeviceImages(row.files, row.already);
    assert.equal(got.files.length, row.want.n, JSON.stringify(row));
    assert.equal(got.notImage, row.want.notImage, JSON.stringify(row));
    assert.equal(got.tooLarge, row.want.tooLarge, JSON.stringify(row));
    assert.equal(got.tooMany, row.want.tooMany, JSON.stringify(row));
  }
});

test("sceneHasInk ignores empty and deleted", () => {
  assert.equal(sceneHasInk([]), false);
  assert.equal(sceneHasInk(null), false);
  assert.equal(sceneHasInk([{ id: "a", isDeleted: true }]), false);
  assert.equal(sceneHasInk([{ id: "a", type: "rectangle" }]), true);
});
