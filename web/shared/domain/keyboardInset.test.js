import assert from "node:assert/strict";
import { test } from "node:test";
import { createSticky } from "./termSticky.js";
import {
  HARD_KEYBOARD_WAIT_MS,
  extraKeysVisible,
  hardKeyboardLikely,
  inputIsFocused,
  keyboardInset,
  sendTermSeq,
  shellLayout,
} from "./keyboardInset.js";

test("keyboardInset is innerHeight minus visual viewport minus offsetTop", () => {
  assert.equal(keyboardInset({ innerHeight: 800, vvHeight: 500, vvOffsetTop: 0 }), 300);
  assert.equal(keyboardInset({ innerHeight: 800, vvHeight: 500, vvOffsetTop: 40 }), 260);
  assert.equal(keyboardInset({ innerHeight: 800, vvHeight: 800, vvOffsetTop: 0 }), 0);
  assert.equal(keyboardInset({ innerHeight: 500, vvHeight: 800, vvOffsetTop: 0 }), 0);
});

test("decision table: extra keys follow focus, not a header flag", () => {
  const rows = [
    { name: "rest", termFocused: false, hardKeyboard: false, want: false },
    { name: "software IME", termFocused: true, hardKeyboard: false, want: true },
    { name: "hardware", termFocused: true, hardKeyboard: true, want: false },
    { name: "blurred hardware", termFocused: false, hardKeyboard: true, want: false },
  ];
  for (const row of rows) {
    assert.equal(extraKeysVisible(row), row.want, row.name);
  }
});

test("decision table: hardware hide is conservative", () => {
  const focused = { termFocused: true, inset: 0, elapsedMs: HARD_KEYBOARD_WAIT_MS, finePointer: true };
  assert.equal(hardKeyboardLikely(focused), true);
  assert.equal(hardKeyboardLikely({ ...focused, termFocused: false }), false);
  assert.equal(hardKeyboardLikely({ ...focused, elapsedMs: HARD_KEYBOARD_WAIT_MS - 1 }), false);
  assert.equal(hardKeyboardLikely({ ...focused, inset: 300, finePointer: true }), false);
  assert.equal(hardKeyboardLikely({ ...focused, finePointer: false }), false);
  assert.equal(extraKeysVisible({ termFocused: true, hardKeyboard: false }), true);
});

test("decision table: pin the shell only while the IME covers pixels", () => {
  const rest = shellLayout({
    innerHeight: 844, vvHeight: 800, vvOffsetTop: 0, scale: 1,
    restHeight: 800, inputFocused: false,
  });
  assert.equal(rest.keyboardOpen, false);

  const attachFocus = shellLayout({
    innerHeight: 844, vvHeight: 800, vvOffsetTop: 0, scale: 1,
    restHeight: 800, inputFocused: true,
  });
  assert.equal(attachFocus.keyboardOpen, false);

  const iosOverlay = shellLayout({
    innerHeight: 844, vvHeight: 500, vvOffsetTop: 0, scale: 1,
    restHeight: 800, inputFocused: true,
  });
  assert.equal(iosOverlay.keyboardOpen, true);
  assert.equal(iosOverlay.height, 500);
  assert.equal(iosOverlay.inset, 344);

  const classic = shellLayout({
    innerHeight: 800, vvHeight: 480, vvOffsetTop: 12, scale: 1,
    restHeight: 800, inputFocused: true,
  });
  assert.equal(classic.keyboardOpen, true);
  assert.equal(classic.height, 480);
  assert.equal(classic.offsetTop, 12);

  assert.equal(shellLayout({
    innerHeight: 800, vvHeight: 400, vvOffsetTop: 0, scale: 2, inputFocused: true,
  }), null);
});

test("inputIsFocused recognises xterm's helper textarea", () => {
  assert.equal(inputIsFocused(null), false);
  assert.equal(inputIsFocused({ tagName: "DIV" }), false);
  assert.equal(inputIsFocused({ tagName: "TEXTAREA" }), true);
  assert.equal(inputIsFocused({ tagName: "INPUT" }), true);
  assert.equal(inputIsFocused({ tagName: "DIV", isContentEditable: true }), true);
});

test("sendTermSeq applies sticky modifiers and refocuses only when the host had focus", () => {
  const sent = [];
  const sticky = createSticky();
  sticky.arm("ctrl");
  const entry = {
    sticky,
    sock: { readyState: 1, send: (b) => sent.push(b) },
  };
  const focused = sendTermSeq(entry, "\x1b[A", true);
  assert.equal(focused.sent, true);
  assert.equal(focused.refocus, true);
  assert.equal(focused.bytes, "\x1b[1;5A");
  assert.equal(sent.length, 1);

  const blurred = sendTermSeq({ sock: { readyState: 1, send: (b) => sent.push(b) } }, "\x1b", false);
  assert.equal(blurred.refocus, false);
  assert.equal(blurred.bytes, "\x1b");

  assert.deepEqual(sendTermSeq(null, "a", true), { sent: false, refocus: false, bytes: "" });
});
