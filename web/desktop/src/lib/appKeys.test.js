import { test } from "node:test";
import assert from "node:assert/strict";
import { CATALOG, matchAction, matchGlobalAction, primaryChord, formatChord } from "./appKeys.js";
import { firstReservedChord } from "@picode/shared/domain/browserChord.js";

test("no default chord sits in the browser's reserved set", () => {
  // The browser consumes the reserved set (Ctrl+T, Ctrl+W, …) before the
  // page sees any keydown, so a default there could never fire — inside a
  // pane or out (browserChord.js, docs-site/guide/keyboard.md).
  for (const action of CATALOG) {
    const bad = firstReservedChord(action.defaults || []);
    assert.equal(bad, null, action.id + " defaults onto reserved chord " + bad);
  }
});


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

test("tab cycling defaults to Alt+bracket, which browsers leave alone", () => {
  assert.equal(matchAction("app.tab.prev", ev({ key: "[", altKey: true }), {}), true);
  assert.equal(matchAction("app.tab.next", ev({ key: "]", altKey: true }), {}), true);
  assert.equal(matchAction("app.tab.next", ev({ key: "]", ctrlKey: true }), {}), false);
});

test("matchGlobalAction covers every Global chord and nothing else", () => {
  assert.equal(matchGlobalAction(ev({ key: "k", ctrlKey: true }), {}), true);
  assert.equal(matchGlobalAction(ev({ key: "]", altKey: true }), {}), true);
  assert.equal(matchGlobalAction(ev({ key: "d", ctrlKey: true }), {}), false); // Composer group
  assert.equal(matchGlobalAction(ev({ key: "]" }), {}), false);
});

test("close tab defaults to Alt+W and counts as a Global chord", () => {
  assert.equal(matchAction("app.tab.close", ev({ key: "w", altKey: true }), {}), true);
  assert.equal(matchGlobalAction(ev({ key: "w", altKey: true }), {}), true);
  assert.equal(matchAction("app.tab.close", ev({ key: "w", ctrlKey: true }), {}), false);
});

test("fullscreen defaults to Ctrl+Shift+Enter and collides with no other action", () => {
  const chord = ev({ key: "Enter", ctrlKey: true, shiftKey: true });
  assert.equal(matchAction("app.fullscreen.toggle", chord, {}), true);
  assert.equal(matchGlobalAction(chord, {}), true, "a terminal hands the chord back to the app");
  for (const a of CATALOG) {
    if (a.id === "app.fullscreen.toggle") continue;
    assert.equal(matchAction(a.id, chord, {}), false, `${a.id} also answers Ctrl+Shift+Enter`);
  }
  // One modifier is the terminal's newline (termKeys.js); two are ours.
  assert.equal(matchAction("app.fullscreen.toggle", ev({ key: "Enter", ctrlKey: true }), {}), false);
  assert.equal(matchAction("app.fullscreen.toggle", ev({ key: "Enter", shiftKey: true }), {}), false);
  assert.equal(matchAction("app.fullscreen.toggle", ev({ key: "Enter", altKey: true }), {}), false);
  assert.equal(formatChord(primaryChord("app.fullscreen.toggle", {})), "Ctrl+Shift+Enter");
});

test("every default chord in the catalog is claimed by exactly one action", () => {
  const owner = new Map();
  for (const a of CATALOG) {
    for (const chord of a.defaults || []) {
      assert.equal(owner.has(chord), false, `${chord} is claimed by both ${owner.get(chord)} and ${a.id}`);
      owner.set(chord, a.id);
    }
  }
});
