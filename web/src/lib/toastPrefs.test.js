import { test } from "node:test";
import assert from "node:assert/strict";
import { readToastPrefs, persistToastPrefs, defaultToastPrefs } from "./toastPrefs.js";

test("toast prefs round-trip", () => {
  const mem = {};
  globalThis.localStorage = {
    getItem: (k) => (k in mem ? mem[k] : null),
    setItem: (k, v) => { mem[k] = String(v); },
  };
  globalThis.window = { dispatchEvent() {} };
  const d = defaultToastPrefs();
  assert.equal(readToastPrefs().position, d.position);
  persistToastPrefs({ position: "bottom-left", duration: 8000, expand: true, closePlace: "edge-left" });
  const got = readToastPrefs();
  assert.equal(got.position, "bottom-left");
  assert.equal(got.duration, 8000);
  assert.equal(got.expand, true);
  assert.equal(got.closePlace, "edge-left");
});
