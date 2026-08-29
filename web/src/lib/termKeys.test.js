import assert from "node:assert/strict";
import { test } from "node:test";
import { copyPasteAction, newlineSeq } from "./termKeys.js";

test("Shift+Enter is the default newline", () => {
  assert.equal(newlineSeq({ type: "keydown", key: "Enter", shiftKey: true }, "shift-enter"), "\x1b[27;2;13~");
  assert.equal(newlineSeq({ type: "keydown", key: "Enter" }, "shift-enter"), null);
});

test("Ctrl+Enter and Alt+Enter follow the pref", () => {
  assert.equal(newlineSeq({ type: "keydown", key: "Enter", ctrlKey: true }, "ctrl-enter"), "\x1b[27;5;13~");
  assert.equal(newlineSeq({ type: "keydown", key: "Enter", altKey: true }, "alt-enter"), "\x1b[27;3;13~");
  assert.equal(newlineSeq({ type: "keydown", key: "Enter", shiftKey: true }, "ctrl-enter"), null);
});

test("Ctrl+Shift+C/V are copy/paste", () => {
  assert.equal(copyPasteAction({ type: "keydown", key: "c", ctrlKey: true, shiftKey: true }), "copy");
  assert.equal(copyPasteAction({ type: "keydown", key: "v", metaKey: true, shiftKey: true }), "paste");
  assert.equal(copyPasteAction({ type: "keydown", key: "c", ctrlKey: true }), null);
});
