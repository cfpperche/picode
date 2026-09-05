import assert from "node:assert/strict";
import { test } from "node:test";
import { createSticky, controlByte } from "./termSticky.js";

test("Ctrl arms once and turns the next letter into its control byte", () => {
  const s = createSticky();
  assert.deepEqual(s.arm("ctrl"), { ctrl: true, alt: false });
  assert.equal(s.apply("c"), "\x03");
  assert.deepEqual(s.state(), { ctrl: false, alt: false });
  assert.equal(s.apply("c"), "c"); // used up
});

test("Ctrl+[ is Esc, Ctrl+? is DEL; Ctrl+digit keeps waiting", () => {
  const s = createSticky();
  s.arm("ctrl");
  assert.equal(s.apply("["), "\x1b");
  s.arm("ctrl");
  assert.equal(s.apply("7"), "7");
  assert.equal(s.state().ctrl, true);
  assert.equal(controlByte("Z"), "\x1a");
  assert.equal(controlByte("é"), null);
});

test("Alt prefixes ESC; both together compose", () => {
  const s = createSticky();
  s.arm("alt");
  assert.equal(s.apply("b"), "\x1bb");
  s.arm("alt");
  s.arm("ctrl");
  assert.equal(s.apply("x"), "\x1b\x18");
  assert.deepEqual(s.state(), { ctrl: false, alt: false });
});

test("a paste or an escape sequence passes untouched and keeps the modifier", () => {
  const s = createSticky();
  s.arm("ctrl");
  assert.equal(s.apply("hello"), "hello");
  assert.equal(s.apply("\x1b[A"), "\x1b[A");
  assert.equal(s.state().ctrl, true);
});

test("second tap disarms; the timeout disarms", () => {
  let t = 1000;
  const s = createSticky({ now: () => t, timeout: 5000 });
  s.arm("ctrl");
  assert.equal(s.arm("ctrl").ctrl, false);
  s.arm("alt");
  t += 5001;
  assert.deepEqual(s.state(), { ctrl: false, alt: false });
  assert.equal(s.apply("b"), "b");
});

test("bar keys: armed modifiers become xterm modified arrows; Esc/Tab pass", () => {
  const s = createSticky();
  s.arm("ctrl");
  assert.equal(s.applyKey("\x1b[A"), "\x1b[1;5A");
  s.arm("alt");
  assert.equal(s.applyKey("\x1b[D"), "\x1b[1;3D");
  s.arm("ctrl");
  s.arm("alt");
  assert.equal(s.applyKey("\x1b[H"), "\x1b[1;7H");
  s.arm("ctrl");
  assert.equal(s.applyKey("\t"), "\t");
  assert.equal(s.state().ctrl, true); // Tab could not use it
  assert.equal(s.applyKey("/"), "\x1f"); // Ctrl+/ has a byte
});

test("subscribers hear every change", () => {
  const s = createSticky();
  const seen = [];
  const off = s.subscribe((st) => seen.push(st.ctrl + ":" + st.alt));
  s.arm("ctrl");
  s.apply("c");
  off();
  s.arm("alt");
  assert.deepEqual(seen, ["true:false", "false:false"]);
});
