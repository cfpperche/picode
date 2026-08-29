import assert from "node:assert/strict";
import { test } from "node:test";
import { copyPasteAction, newlineSeq, termShortcutRows, termDataFilter, wireTermKeys } from "./termKeys.js";

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

test("wireTermKeys blocks the trailing keypress and sends once", () => {
  const sent = [];
  const send = (bytes) => sent.push(Array.from(bytes));
  const handler = extractHandler(wireTermKeys, send);
  const kd = { type: "keydown", key: "Enter", shiftKey: true, preventDefault() { this.pd = true; } };
  const kp = { type: "keypress", key: "Enter", shiftKey: true, charCode: 13, preventDefault() { this.pd = true; } };
  assert.equal(handler(kd), false);
  assert.equal(handler(kp), false); // blocked: no \\r from xterm _keyPress
  assert.equal(kd.pd, true);
  assert.equal(kp.pd, true);
  assert.equal(sent.length, 1);     // sequence emitted once (keydown)
  assert.deepEqual(sent[0], Array.from(new TextEncoder().encode("\x1b[27;2;13~")));
  // plain Enter keypress still passes through (submits)
  assert.equal(handler({ type: "keypress", key: "Enter" }), true);
});

function extractHandler(wire, send) {
  let h = null;
  wire({ attachCustomKeyEventHandler(fn) { h = fn; } }, send);
  return h;
}

function fakeEnterKey(mods = {}) {
  const { bubbles, ...rest } = mods;
  const e = new Event("keydown", { bubbles: !!bubbles });
  Object.assign(e, rest);
  e.key = "Enter";
  return e;
}

test("termDataFilter converts a stray \\r after an unhandled Shift+Enter (Windows Chrome)", () => {
  const sent = [];
  const send = (bytes) => sent.push(Array.from(bytes));
  const ta = new EventTarget(); // fake textarea for the capture tracker
  let h = null;
  wireTermKeys({ attachCustomKeyEventHandler(fn) { h = fn; }, textarea: ta }, send);

  // Windows-miss scenario: the keydown tracker sees it, the handler does not
  ta.dispatchEvent(fakeEnterKey({ shiftKey: true, bubbles: true }));
  assert.equal(termDataFilter("\r"), "\x1b[27;2;13~"); // converted, not submitted
  assert.equal(termDataFilter("\r"), ""); // subsequent echo dropped

  // Handler-ran scenario: sequence sent once, echo \r dropped
  ta.dispatchEvent(fakeEnterKey({ shiftKey: true, bubbles: true }));
  assert.equal(h({ type: "keydown", key: "Enter", shiftKey: true, preventDefault() {} }), false);
  assert.equal(termDataFilter("\r"), "");
  assert.equal(sent.length, 1);

  // Plain Enter (no modifiers tracked) passes through
  ta.dispatchEvent(fakeEnterKey({}));
  assert.equal(termDataFilter("\r"), "\r");
  // Non-bound modified Enter keeps VS Code parity (\r passes)
  ta.dispatchEvent(fakeEnterKey({ ctrlKey: true, bubbles: true })); // pref is shift-enter
  assert.equal(termDataFilter("\r"), "\r");
  // Non-Enter data untouched
  assert.equal(termDataFilter("abc"), "abc");
});
