import { test } from "node:test";
import assert from "node:assert/strict";
import { readToastPrefs, persistToastPrefs, defaultToastPrefs } from "./toastPrefs.js";

function memoryStorage(seed = {}) {
  const mem = { ...seed };
  globalThis.localStorage = {
    getItem: (k) => (k in mem ? mem[k] : null),
    setItem: (k, v) => { mem[k] = String(v); },
  };
  globalThis.window = { dispatchEvent() {} };
  return mem;
}

test("toast prefs round-trip", () => {
  memoryStorage();
  const d = defaultToastPrefs();
  assert.equal(readToastPrefs().position, d.position);
  persistToastPrefs({ position: "bottom-left", duration: 8000, expand: true, announceFinished: false });
  const got = readToastPrefs();
  assert.equal(got.position, "bottom-left");
  assert.equal(got.duration, 8000);
  assert.equal(got.expand, true);
  assert.equal(got.announceFinished, false);
  assert.equal(got.announceNeedsYou, true);
});

test("both announcements default to on, and a retired key is ignored", () => {
  // What a browser that used the pre-card preferences has on disk.
  memoryStorage({ "picode-toast": JSON.stringify({ position: "bottom-right", closePlace: "edge-left", richColors: true }) });
  const got = readToastPrefs();
  assert.equal(got.position, "bottom-right");
  assert.equal(got.announceFinished, true);
  assert.equal(got.announceNeedsYou, true);
  assert.equal("closePlace" in got, false);
  assert.equal("richColors" in got, false);
});
