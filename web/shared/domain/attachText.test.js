import assert from "node:assert/strict";
import { test } from "node:test";
import { attachFieldHeight, isAttachSendKey, MAX_ATTACH_LINES } from "./attachText.js";

// Desktop chrome: 7px padding top/bottom, 1px border each side, 20px line.
const DESKTOP = { lineHeight: 20, padTop: 7, padBottom: 7, borderTop: 1, borderBottom: 1, minHeight: 36 };

test("one line is one control height", () => {
  assert.equal(attachFieldHeight({ ...DESKTOP, scrollHeight: 34 }), 36);
});

test("grows with the content, borders included", () => {
  assert.equal(attachFieldHeight({ ...DESKTOP, scrollHeight: 56 }), 58);
});

test("four measured lines land exactly on the four-line cap", () => {
  // 4 lines × 20px + 14px padding = 94px of scrollHeight; +2px border = 96.
  assert.equal(attachFieldHeight({ ...DESKTOP, scrollHeight: 94 }), 96);
  assert.equal(attachFieldHeight({ ...DESKTOP, scrollHeight: 400 }), 96);
});

test("stops at four lines", () => {
  assert.equal(attachFieldHeight({ ...DESKTOP, scrollHeight: 400 }), 20 * MAX_ATTACH_LINES + 16);
});

test("never below one line even without a CSS floor", () => {
  assert.equal(attachFieldHeight({ lineHeight: 20, scrollHeight: 4 }), 20);
});

test("a floor taller than one measured line wins", () => {
  assert.equal(attachFieldHeight({ lineHeight: 20, minHeight: 44, scrollHeight: 20 }), 44);
});

test("maxLines is parametric", () => {
  const mobile = { lineHeight: 24, padTop: 5, padBottom: 5, borderTop: 1, borderBottom: 1, scrollHeight: 900, maxLines: 2 };
  assert.equal(attachFieldHeight(mobile), 24 * 2 + 12);
});

test("isAttachSendKey: Enter sends, Shift+Enter breaks the line", () => {
  assert.equal(isAttachSendKey({ key: "Enter" }), true);
  assert.equal(isAttachSendKey({ key: "Enter", shiftKey: true }), false);
  assert.equal(isAttachSendKey({ key: "a" }), false);
  assert.equal(isAttachSendKey(null), false);
});

test("isAttachSendKey: a composing Enter never sends", () => {
  assert.equal(isAttachSendKey({ key: "Enter", isComposing: true }), false);
  assert.equal(isAttachSendKey({ key: "Enter", keyCode: 229 }), false);
});
