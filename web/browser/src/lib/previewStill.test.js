import { test } from "node:test";
import assert from "node:assert/strict";
import { coverDecision, previewBytes, previewUrl, verifyPreviewUrl } from "./previewStill.js";

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

test("an empty URL never verifies", async () => {
  await assert.rejects(() => verifyPreviewUrl(""), /no image/);
});

// The legacy hide/restore table (pre-ADR-0161 shells). One row per line of
// the comment on coverDecision; "visible" rows are the restore side.
test("the hide/restore decision table", () => {
  const row = (over) => coverDecision({ liveLayers: false, overlayCover: false, menuOpen: false, still: "", previewFailed: false, ...over });

  // A live-layers shell never hides by decision — geometry owns the page.
  assert.equal(coverDecision({ liveLayers: true, overlayCover: true, menuOpen: true, still: "blob:x", previewFailed: true }), false);

  // A floating overlay (dialog, toast, palette) covers with no menu at all.
  assert.equal(row({ overlayCover: true }), true);
  // …and restoring is the same row backwards: the overlay gone, nothing else
  // holding, the page is visible again.
  assert.equal(row({ overlayCover: true, menuOpen: true, still: "blob:x" }), true);

  // Menu open with a verified still: the frozen page sits behind it.
  assert.equal(row({ menuOpen: true, still: "blob:x" }), true);

  // Menu open, capture still in flight: **visible** — the menu waits rather
  // than parking the live page behind gray (2026-09-17).
  assert.equal(row({ menuOpen: true }), false);

  // Menu open, capture failed: covered over the honest host gray.
  assert.equal(row({ menuOpen: true, previewFailed: true }), true);

  // Menu shut restores the page — a stale still alone never covers it.
  assert.equal(row({ still: "blob:x", previewFailed: true }), false);
});
