import { test } from "node:test";
import assert from "node:assert/strict";
import { isReloadKey, desktopReloadAction } from "./desktopReload.js";

function ev(overrides) {
  return { key: "r", ctrlKey: false, shiftKey: false, altKey: false, metaKey: false, repeat: false, isComposing: false, keyCode: 0, ...overrides };
}

test("isReloadKey accepts Ctrl+R, Cmd+R and F5", () => {
  assert.equal(isReloadKey(ev({ key: "r", ctrlKey: true })), true);
  assert.equal(isReloadKey(ev({ key: "R", ctrlKey: true })), true);
  assert.equal(isReloadKey(ev({ key: "r", metaKey: true })), true);
  assert.equal(isReloadKey(ev({ key: "F5" })), true);
});

test("isReloadKey rejects chords a terminal or the browser still owns", () => {
  assert.equal(isReloadKey(ev({ key: "r" })), false);
  assert.equal(isReloadKey(ev({ key: "r", ctrlKey: true, shiftKey: true })), false);
  assert.equal(isReloadKey(ev({ key: "r", ctrlKey: true, altKey: true })), false);
  assert.equal(isReloadKey(ev({ key: "F5", ctrlKey: true })), false);
  assert.equal(isReloadKey(ev({ key: "r", ctrlKey: true, repeat: true })), false);
  assert.equal(isReloadKey(ev({ key: "r", ctrlKey: true, isComposing: true })), false);
  assert.equal(isReloadKey(ev({ key: "r", ctrlKey: true, keyCode: 229 })), false);
});

test("a terminal pane keeps Ctrl+R", () => {
  assert.equal(desktopReloadAction({ inTerminal: true, selectedTab: "w:3", panes: {}, onPane: false }), null);
});

test("a visible work tab reloads that page", () => {
  assert.deepEqual(
    desktopReloadAction({ inTerminal: false, selectedTab: "w:3", panes: {}, onPane: false }),
    { kind: "page", id: "3" },
  );
});

test("an agent split reloads the bound page", () => {
  assert.deepEqual(
    desktopReloadAction({ inTerminal: false, selectedTab: "ag1", panes: { ag1: "9" }, onPane: false }),
    { kind: "page", id: "9" },
  );
});

test("settings covering the pane reloads PiCode, not the parked page", () => {
  assert.deepEqual(
    desktopReloadAction({ inTerminal: false, selectedTab: "w:3", panes: {}, onPane: true }),
    { kind: "app" },
  );
});

test("everywhere else on desktop reloads PiCode", () => {
  assert.deepEqual(
    desktopReloadAction({ inTerminal: false, selectedTab: "ag1", panes: {}, onPane: false }),
    { kind: "app" },
  );
});
