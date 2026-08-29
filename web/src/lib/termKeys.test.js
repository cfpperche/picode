import assert from "node:assert/strict";
import { test } from "node:test";
import { copyPasteAction, newlineSeq, termShortcutRows } from "./termKeys.js";

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
});

test("Ctrl+C copies only when copyIfSelection is on", () => {
  const ev = { type: "keydown", key: "c", ctrlKey: true };
  assert.equal(copyPasteAction(ev, { copyIfSelection: true }), "copy-if-sel");
  assert.equal(copyPasteAction(ev, { copyIfSelection: false }), null);
  assert.equal(copyPasteAction(ev, {}), null);
  assert.equal(copyPasteAction(ev), null);
});

test("termShortcutRows follow newline and copy prefs", () => {
  const on = termShortcutRows({ newlineKey: "shift-enter", copyIfSelection: true });
  assert.ok(on.some((r) => r.key === "Ctrl+C"));
  assert.ok(on.some((r) => r.key === "Shift+Enter"));
  const off = termShortcutRows({ newlineKey: "ctrl-enter", copyIfSelection: false });
  assert.ok(!off.some((r) => r.key === "Ctrl+C"));
  assert.ok(off.some((r) => r.key === "Ctrl+Enter"));
});
