import { test } from "node:test";
import assert from "node:assert/strict";
import { previewBytes, previewUrl } from "./previewStill.js";

test("empty captures are not bytes worth hiding the page behind", () => {
  assert.equal(previewBytes(null), null);
  assert.equal(previewBytes(undefined), null);
  assert.equal(previewBytes(new ArrayBuffer(0)), null);
  assert.equal(previewBytes(new Uint8Array(0)), null);
  assert.equal(previewBytes([]), null);
  assert.equal(previewBytes("89504e47"), null);
});

test("live captures survive in their binary shape", () => {
  const buf = new Uint8Array([137, 80, 78, 71]).buffer;
  assert.equal(previewBytes(buf), buf);
  const view = new Uint8Array([1, 2, 3]);
  assert.deepEqual(new Uint8Array(previewBytes(view)), view);
  assert.deepEqual(new Uint8Array(previewBytes([1, 2, 3])), view);
});

test("no bytes means no still URL", () => {
  assert.equal(previewUrl(new ArrayBuffer(0)), "");
  assert.equal(previewUrl(null), "");
});

test("real bytes mint a blob URL", () => {
  const url = previewUrl(new Uint8Array([137, 80, 78, 71]).buffer);
  assert.match(url, /^blob:/);
  URL.revokeObjectURL(url);
});
