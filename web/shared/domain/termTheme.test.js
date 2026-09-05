import assert from "node:assert/strict";
import { test, beforeEach } from "node:test";

const mem = {};
globalThis.localStorage = {
  getItem(k) { return Object.prototype.hasOwnProperty.call(mem, k) ? mem[k] : null; },
  setItem(k, v) { mem[k] = String(v); },
  removeItem(k) { delete mem[k]; },
};
globalThis.window = { dispatchEvent() {} };

const {
  readTermPrefs, persistTermPrefs, persistTermTheme, persistTermFontSize,
  bumpTermFontSize, defaultTermPrefs, TERM_SIZE_DEFAULT,
} = await import("./termTheme.js");

beforeEach(() => {
  for (const k of Object.keys(mem)) delete mem[k];
});

test("defaults when empty", () => {
  assert.deepEqual(readTermPrefs(), defaultTermPrefs());
});

test("migrates legacy theme and size keys", () => {
  mem["picode-term-theme"] = "light";
  mem["picode-term-size"] = "18";
  const p = readTermPrefs();
  assert.equal(p.theme, "light");
  assert.equal(p.fontSize, 18);
});

test("clamps size, padding, scrollback, line height", () => {
  const p = persistTermPrefs({
    fontSize: 99, padding: -4, scrollback: 10, lineHeight: 9, letterSpacing: 40,
    font: "nope", cursorStyle: "beam", theme: "neon",
  });
  assert.equal(p.fontSize, 22);
  assert.equal(p.padding, 0);
  assert.equal(p.scrollback, 1000);
  assert.equal(p.lineHeight, 1.8);
  assert.equal(p.letterSpacing, 4);
  assert.equal(p.font, "jetbrains");
  assert.equal(p.cursorStyle, "block");
  assert.equal(p.theme, "dark");
});

test("theme and size helpers write the bundle", () => {
  persistTermTheme("light");
  persistTermFontSize(16);
  const p = readTermPrefs();
  assert.equal(p.theme, "light");
  assert.equal(p.fontSize, 16);
  assert.equal(bumpTermFontSize(0), TERM_SIZE_DEFAULT);
});
