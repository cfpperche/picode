import assert from "node:assert/strict";
import { test } from "node:test";
import { isImageFile, planAttachFiles, clipboardFiles, termHasPromptDoor, promptDoorFor, clipboardText, MAX_ATTACH, MAX_ATTACH_BYTES } from "./termPrompt.js";

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

test("clipboardFiles", () => {
  assert.deepEqual(clipboardFiles(null), []);
  assert.deepEqual(clipboardFiles({}), []);
  assert.deepEqual(clipboardFiles({ files: [] }), []);
  const f = { name: "shot.png" };
  assert.deepEqual(clipboardFiles({ files: [f, null] }), [f]);
});

test("termHasPromptDoor", () => {
  assert.equal(termHasPromptDoor(null), false);
  assert.equal(termHasPromptDoor({}), false);
  assert.equal(termHasPromptDoor({ launchCli: "codex", running: false }), false);
  assert.equal(termHasPromptDoor({ launchCli: "codex", running: true }), true);
  assert.equal(termHasPromptDoor({ launchCli: "pi", running: true }), true);
});

test("promptDoorFor mirrors the context menu rule", () => {
  const cli = { launchCli: "codex", running: true };
  assert.equal(promptDoorFor({ kind: "term", term: cli }), true);
  assert.equal(promptDoorFor({ kind: "term", term: { launchCli: "", running: true } }), false);
  assert.equal(promptDoorFor({ kind: "term", term: { launchCli: "codex", running: false } }), false);
  // An interactive agent pane has the door even when its bound record
  // carries no launch fields (store struct, untracked binding).
  assert.equal(promptDoorFor({ kind: "agent", agentMode: "interactive", term: { id: "t1" } }), true);
  assert.equal(promptDoorFor({ kind: "agent", agentMode: "interactive", term: cli }), true);
  assert.equal(promptDoorFor({ kind: "agent", agentMode: "managed", term: { id: "t1" } }), false);
  assert.equal(promptDoorFor({ kind: "agent", agentMode: "managed", term: cli }), true);
  assert.equal(promptDoorFor({}), false);
  assert.equal(promptDoorFor(), false);
});

test("clipboardText", () => {
  assert.equal(clipboardText(null), "");
  assert.equal(clipboardText({}), "");
  assert.equal(clipboardText({ getData: () => "hello" }), "hello");
  assert.equal(clipboardText({ getData: () => null }), "");
  assert.equal(clipboardText({ getData: () => { throw new Error("denied"); } }), "");
});
