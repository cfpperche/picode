import { test } from "node:test";
import assert from "node:assert/strict";
import { CATALOG, matchAction, primaryChord, formatChord } from "./appKeys.js";

function ev(overrides) {
  return { key: "k", ctrlKey: false, shiftKey: false, altKey: false, metaKey: false, ...overrides };
}

test("matchAction matches the default ctrl chord", () => {
  assert.equal(matchAction("app.palette.toggle", ev({ key: "k", ctrlKey: true }), {}), true);
});

test("matchAction matches the default cmd/meta chord", () => {
  assert.equal(matchAction("app.palette.toggle", ev({ key: "k", metaKey: true }), {}), true);
});

test("matchAction rejects an unrelated key", () => {
  assert.equal(matchAction("app.palette.toggle", ev({ key: "j", ctrlKey: true }), {}), false);
});

test("matchAction respects a user override, ignoring the old default", () => {
  const overrides = { "app.palette.toggle": ["alt+p"] };
  assert.equal(matchAction("app.palette.toggle", ev({ key: "p", altKey: true }), overrides), true);
  assert.equal(matchAction("app.palette.toggle", ev({ key: "k", ctrlKey: true }), overrides), false);
});

test("matchAction returns false for an unknown action id", () => {
  assert.equal(matchAction("nope", ev({ key: "k", ctrlKey: true }), {}), false);
});

test("matchAction reads overrides from localStorage when none are passed", () => {
  const mem = { "picode-app-keys": JSON.stringify({ "app.palette.toggle": ["alt+p"] }) };
  globalThis.localStorage = {
    getItem: (k) => (k in mem ? mem[k] : null),
    setItem: (k, v) => { mem[k] = String(v); },
  };
  assert.equal(matchAction("app.palette.toggle", ev({ key: "p", altKey: true })), true);
  assert.equal(matchAction("app.palette.toggle", ev({ key: "k", ctrlKey: true })), false);
});

test("CATALOG keeps same-group entries contiguous", () => {
  const seen = new Set();
  let lastGroup = null;
  for (const a of CATALOG) {
    if (a.group !== lastGroup) {
      assert.equal(seen.has(a.group), false, `group ${a.group} is not contiguous`);
      seen.add(a.group);
      lastGroup = a.group;
    }
  }
});

test("primaryChord returns the first effective chord", () => {
  assert.equal(primaryChord("app.palette.toggle", {}), "ctrl+k");
  assert.equal(primaryChord("app.palette.toggle", { "app.palette.toggle": ["alt+p"] }), "alt+p");
  assert.equal(primaryChord("nope", {}), "");
});

test("formatChord renders a readable label", () => {
  assert.equal(formatChord("ctrl+shift+o"), "Ctrl+Shift+O");
  assert.equal(formatChord("super+d"), "Cmd+D");
  assert.equal(formatChord("ctrl+`"), "Ctrl+`");
  assert.equal(formatChord(""), "");
});
