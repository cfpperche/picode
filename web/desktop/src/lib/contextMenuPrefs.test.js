import { test } from "node:test";
import assert from "node:assert/strict";
import { readContextMenuPrefs, persistContextMenuPrefs, defaultContextMenuPrefs, modifierHeld } from "./contextMenuPrefs.js";

function mockStorage(seed) {
  const mem = { ...seed };
  globalThis.localStorage = {
    getItem: (k) => (k in mem ? mem[k] : null),
    setItem: (k, v) => { mem[k] = String(v); },
  };
}

test("context menu prefs round-trip", () => {
  mockStorage();
  const d = defaultContextMenuPrefs();
  assert.equal(readContextMenuPrefs().bypassModifier, d.bypassModifier);
  persistContextMenuPrefs({ bypassModifier: "alt" });
  assert.equal(readContextMenuPrefs().bypassModifier, "alt");
});

test("context menu prefs fall back to the default on an invalid stored value", () => {
  mockStorage({ "picode-ctxmenu": JSON.stringify({ bypassModifier: "nonsense" }) });
  assert.equal(readContextMenuPrefs().bypassModifier, "shift");
});

test("modifierHeld reads the matching event flag", () => {
  assert.equal(modifierHeld("shift", { shiftKey: true }), true);
  assert.equal(modifierHeld("shift", { altKey: true }), false);
  assert.equal(modifierHeld("alt", { altKey: true }), true);
  assert.equal(modifierHeld("ctrl", { ctrlKey: true }), true);
  assert.equal(modifierHeld("ctrl", { metaKey: true }), true);
});
