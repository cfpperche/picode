import assert from "node:assert/strict";
import { test } from "node:test";
import { effectiveKeys, fromEvent, isOverride, matchKeys } from "./piKey.js";

test("fromEvent maps chords Pi understands", () => {
  assert.equal(fromEvent({ key: "Backspace", ctrlKey: true }), "ctrl+backspace");
  assert.equal(fromEvent({ key: "Enter", shiftKey: true }), "shift+enter");
  assert.equal(fromEvent({ key: "ArrowLeft", altKey: true }), "alt+left");
  assert.equal(fromEvent({ key: "f" , ctrlKey: true, shiftKey: true }), "ctrl+shift+f");
  assert.equal(fromEvent({ key: "a" }), null);
  assert.equal(fromEvent({ key: "Control", ctrlKey: true }), null);
});

test("effectiveKeys uses user override including empty", () => {
  const a = { id: "x", defaults: ["ctrl+w"] };
  assert.deepEqual(effectiveKeys(a, {}), ["ctrl+w"]);
  assert.deepEqual(effectiveKeys(a, { x: ["ctrl+backspace"] }), ["ctrl+backspace"]);
  assert.deepEqual(effectiveKeys(a, { x: [] }), []);
  assert.equal(isOverride(a, { x: [] }), true);
  assert.equal(isOverride(a, {}), false);
});

test("matchKeys filters label and chord", () => {
  const a = { id: "tui.editor.deleteWordBackward", group: "Delete", label: "Delete word", defaults: ["ctrl+w"] };
  assert.equal(matchKeys(a, {}, ""), true);
  assert.equal(matchKeys(a, {}, "word"), true);
  assert.equal(matchKeys(a, {}, "ctrl+w"), true);
  assert.equal(matchKeys(a, {}, "paste"), false);
});
