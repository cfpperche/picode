import assert from "node:assert/strict";
import { test } from "node:test";
import { shellInvoke, shellClipboardFiles, clipboardFileFromRow } from "./shellClipboard.js";

test("shellInvoke is null without the Tauri bridge", () => {
  assert.equal(shellInvoke(), null);
});

test("clipboardFileFromRow round-trips bytes", async () => {
  const bytes = new Uint8Array([137, 80, 78, 71]);
  let bin = "";
  for (const b of bytes) bin += String.fromCharCode(b);
  const f = clipboardFileFromRow({ name: "a.png", mime: "image/png", data: btoa(bin) });
  assert.equal(f.name, "a.png");
  assert.equal(f.type, "image/png");
  assert.deepEqual(new Uint8Array(await f.arrayBuffer()), bytes);
});

test("clipboardFileFromRow throws on rows without data", () => {
  assert.throws(() => clipboardFileFromRow(null));
  assert.throws(() => clipboardFileFromRow({}));
  assert.throws(() => clipboardFileFromRow({ name: "a.png" }));
});

test("shellClipboardFiles converts rows and skips bad ones", async () => {
  const invoke = async () => [
    { name: "a.png", mime: "image/png", data: btoa("x") },
    null,
    { name: "b.txt", mime: "text/plain", data: btoa("hi") },
  ];
  const files = await shellClipboardFiles(invoke);
  assert.deepEqual(files.map((f) => f.name), ["a.png", "b.txt"]);
});

test("shellClipboardFiles answers null outside the shell or on refusal", async () => {
  assert.equal(await shellClipboardFiles(null), null);
  assert.equal(await shellClipboardFiles(async () => { throw new Error("denied"); }), null);
});
