import { test } from "node:test";
import assert from "node:assert/strict";
import {
  readLayoutPrefs, persistLayoutPrefs, defaultLayoutPrefs,
  resolveBottomBar, letterboxPx, LETTERBOX_MIN, BOTTOM_BAR_MODES, BAR_HEIGHTS,
} from "./layoutPrefs.js";

function memoryStorage(seed = {}) {
  const mem = { ...seed };
  globalThis.localStorage = {
    getItem: (k) => (k in mem ? mem[k] : null),
    setItem: (k, v) => { mem[k] = String(v); },
  };
  globalThis.document = { documentElement: { dataset: {}, style: { setProperty() {} } } };
  return mem;
}

test("layout prefs round-trip", () => {
  memoryStorage();
  const d = defaultLayoutPrefs();
  assert.deepEqual(readLayoutPrefs(), d);
  persistLayoutPrefs({ bottomBar: "edge", barHeight: 64 });
  assert.deepEqual(readLayoutPrefs(), { bottomBar: "edge", barHeight: 64 });
});

test("unknown stored values fall back to defaults", () => {
  memoryStorage({ "picode-layout": JSON.stringify({ bottomBar: "skyward", barHeight: 999 }) });
  assert.deepEqual(readLayoutPrefs(), { bottomBar: "auto", barHeight: 56 });
  memoryStorage({ "picode-layout": "not json" });
  assert.deepEqual(readLayoutPrefs(), { bottomBar: "auto", barHeight: 56 });
});

test("mode catalogs cover the persisted space", () => {
  assert.ok(BOTTOM_BAR_MODES.includes(defaultLayoutPrefs().bottomBar));
  assert.ok(BAR_HEIGHTS.includes(defaultLayoutPrefs().barHeight));
});

test("letterboxPx is the screen-minus-layout gap, ignoring noise", () => {
  assert.equal(letterboxPx(844, 844), 0);
  assert.equal(letterboxPx(844, 840), 0); // 4px < LETTERBOX_MIN
  assert.equal(letterboxPx(844, 844 - LETTERBOX_MIN), 0);
  assert.equal(letterboxPx(844, 797), 47);
  assert.equal(letterboxPx(852, 771), 81);
});

test("letterboxPx with an opaque status bar splits bar from strip", () => {
  assert.equal(letterboxPx(844, 797, { opaqueStatusBar: true }), 0); // 47px is the bar alone
  assert.equal(letterboxPx(844, 750, { opaqueStatusBar: true }), 47); // 94px = bar + strip
  assert.equal(letterboxPx(852, 744, { opaqueStatusBar: true }), 54);
});

test("auto resolves by the bootstrap's letterbox measurement", () => {
  memoryStorage();
  const doc = globalThis.document;
  doc.documentElement.dataset.standalone = "1";
  assert.equal(resolveBottomBar("auto"), "edge");
  assert.equal(resolveBottomBar("low"), "low");
  doc.documentElement.dataset.standalone = "";
  assert.equal(resolveBottomBar("auto"), "default");
});
