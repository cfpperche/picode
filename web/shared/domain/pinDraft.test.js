import assert from "node:assert/strict";
import { test } from "node:test";
import {
  autoTitle, bodyLimit, clearDraft, draftToRestore, missingBackgroundId, normalizeTag,
  pinFileSrc, readDraft, sameDraft, stripBackgroundFiles, utf8Bytes, writeDraft,
} from "./pinDraft.js";

function memStore() {
  const m = new Map();
  return { getItem: (k) => (m.has(k) ? m.get(k) : null), setItem: (k, v) => m.set(k, String(v)), removeItem: (k) => m.delete(k) };
}

test("utf8Bytes and bodyLimit measure what the server measures", () => {
  assert.equal(utf8Bytes("abc"), 3);
  assert.equal(utf8Bytes("é"), 2);
  assert.equal(utf8Bytes("日本"), 6);
  assert.equal(utf8Bytes("\u{1F600}"), 4);
  const ok = bodyLimit("x".repeat(1000));
  assert.deepEqual([ok.over, ok.near], [false, false]);
  const near = bodyLimit("x".repeat(95_000));
  assert.deepEqual([near.over, near.near], [false, true]);
  const over = bodyLimit("x".repeat(100_001));
  assert.equal(over.over, true);
});

test("normalizeTag folds like the store", () => {
  assert.equal(normalizeTag("#Foo"), "foo");
  assert.equal(normalizeTag("  a b   c "), "a-b-c");
  assert.equal(normalizeTag("   "), "");
});

test("autoTitle prefers typed, then the file name, never Untitled", () => {
  assert.equal(autoTitle(" Deploy ", [{ name: "shot.png" }]), "Deploy");
  assert.equal(autoTitle("", [{ name: "screen shot.PNG" }]), "screen shot");
  assert.equal(autoTitle("", [{ name: ".png" }]), "Sketch");
  assert.equal(autoTitle("", []), "Sketch");
  assert.equal(autoTitle("", [], "Note"), "Note");
});

test("retained drafts round-trip and restore only when they add something", () => {
  const s = memStore();
  const server = { title: "T", tags: ["a"], body: "b", updatedAt: "v1" };
  assert.equal(readDraft(s, "p1"), null);
  writeDraft(s, "p1", { title: "T", tags: ["a"], body: "b changed" }, "v1");
  const d = readDraft(s, "p1");
  assert.equal(d.body, "b changed");
  assert.equal(d.base, "v1");
  assert.deepEqual(draftToRestore(d, server), d);
  // Same as the server: nothing to restore.
  writeDraft(s, "p1", { title: "T", tags: ["a"], body: "b" }, "v1");
  assert.equal(draftToRestore(readDraft(s, "p1"), server), null);
  // Taken from an older version than the server now has: stale, dropped.
  writeDraft(s, "p1", { title: "T", tags: ["a"], body: "older edit" }, "v0");
  assert.equal(draftToRestore(readDraft(s, "p1"), server), null);
  // A new pin restores anything non-blank.
  writeDraft(s, "", { title: "", tags: [], body: "" }, "");
  assert.equal(draftToRestore(readDraft(s, ""), null), null);
  writeDraft(s, "", { title: "x", tags: [], body: "" }, "");
  assert.equal(draftToRestore(readDraft(s, ""), null).title, "x");
  clearDraft(s, "");
  assert.equal(readDraft(s, ""), null);
  assert.equal(sameDraft({ title: "a", tags: ["x", "y"], body: "" }, { title: "a", tags: ["x", "y"], body: "" }), true);
  assert.equal(sameDraft({ title: "a", tags: ["x"], body: "" }, { title: "a", tags: ["y"], body: "" }), false);
});

test("background files leave the scene and are found missing on open", () => {
  const scene = {
    elements: [{ type: "image", fileId: "bg:/api/pins/p/files/f" }, { type: "image", fileId: "pasted1" }],
    files: { "bg:/api/pins/p/files/f": { dataURL: "data:..." }, pasted1: { dataURL: "data:..." } },
  };
  const out = stripBackgroundFiles(scene);
  assert.deepEqual(Object.keys(out.files), ["pasted1"]);
  assert.equal(Object.keys(scene.files).length, 2, "input untouched");
  assert.equal(missingBackgroundId(out), "bg:/api/pins/p/files/f");
  assert.equal(missingBackgroundId(scene), "");
  assert.equal(missingBackgroundId({ elements: [{ type: "image", fileId: "legacy123" }], files: {} }), "", "legacy embedded ids are not ours");
  assert.equal(stripBackgroundFiles(null), null);
});

test("pinFileSrc carries the version", () => {
  assert.equal(pinFileSrc("p", { id: "f", createdAt: "c" }), "/api/pins/p/files/f?v=c");
  assert.equal(pinFileSrc("p", { id: "f", createdAt: "c", updatedAt: "2026-09-08T10:00:00Z" }), "/api/pins/p/files/f?v=2026-09-08T10%3A00%3A00Z");
  assert.equal(pinFileSrc("p", { id: "f" }), "/api/pins/p/files/f");
  assert.equal(pinFileSrc("", { id: "f" }), "");
});
