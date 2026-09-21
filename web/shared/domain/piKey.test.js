import assert from "node:assert/strict";
import { test } from "node:test";
import { effectiveKeys, fromEvent, isOverride, matchKeys, platformAlternates } from "./piKey.js";

test("fromEvent maps chords Pi understands", () => {
  assert.equal(fromEvent({ key: "Backspace", ctrlKey: true }), "ctrl+backspace");
  assert.equal(fromEvent({ key: "Enter", shiftKey: true }), "shift+enter");
  assert.equal(fromEvent({ key: "ArrowLeft", altKey: true }), "alt+left");
  assert.equal(fromEvent({ key: "f" , ctrlKey: true, shiftKey: true }), "ctrl+shift+f");
  assert.equal(fromEvent({ key: "a" }), null);
  assert.equal(fromEvent({ key: "Control", ctrlKey: true }), null);
});

test("effectiveKeys uses user override including empty", () => {
  const a = { id: "x", defaults: ["ctrl+w"] };
  assert.deepEqual(effectiveKeys(a, {}), ["ctrl+w"]);
  assert.deepEqual(effectiveKeys(a, { x: ["ctrl+backspace"] }), ["ctrl+backspace"]);
  assert.deepEqual(effectiveKeys(a, { x: [] }), []);
  assert.equal(isOverride(a, { x: [] }), true);
  assert.equal(isOverride(a, {}), false);
});

test("matchKeys filters label and chord", () => {
  const a = { id: "tui.editor.deleteWordBackward", group: "Delete", label: "Delete word", defaults: ["ctrl+w"] };
  assert.equal(matchKeys(a, {}, ""), true);
  assert.equal(matchKeys(a, {}, "word"), true);
  assert.equal(matchKeys(a, {}, "ctrl+w"), true);
  assert.equal(matchKeys(a, {}, "paste"), false);
});

// Nine of pi's actions bind differently on Windows and WSL; an alternate is
// the whole binding there, and a declared empty list means pi binds nothing.
// The pane reads its platform from the server, so both halves are pinned here.
test("a platform alternate replaces the default it covers", () => {
  const undo = { id: "tui.editor.undo", defaults: ["ctrl+-"], alt: { windows: ["ctrl+z"], wsl: ["alt+z"] } };
  assert.deepEqual(effectiveKeys(undo, {}, "wsl"), ["alt+z"]);
  assert.deepEqual(effectiveKeys(undo, {}, "windows"), ["ctrl+z"]);
  assert.deepEqual(effectiveKeys(undo, {}, "linux"), ["ctrl+-"]);
  assert.deepEqual(effectiveKeys(undo, {}), ["ctrl+-"], "no platform falls back to the base default");
  // A user's own binding is the user's on every platform.
  assert.deepEqual(effectiveKeys(undo, { "tui.editor.undo": ["ctrl+u"] }, "wsl"), ["ctrl+u"]);
  assert.equal(matchKeys(undo, {}, "alt+z", "wsl"), true);
  assert.equal(matchKeys(undo, {}, "alt+z", "linux"), false);

  const suspend = { id: "app.suspend", defaults: ["ctrl+z"], alt: { windows: [] } };
  assert.deepEqual(effectiveKeys(suspend, {}, "windows"), [], "a declared empty list is unbound there");
  assert.deepEqual(effectiveKeys(suspend, {}, "wsl"), ["ctrl+z"], "wsl keeps the base binding");

  assert.deepEqual(platformAlternates(undo, "wsl"), [{ platform: "windows", keys: ["ctrl+z"] }]);
  assert.deepEqual(platformAlternates(undo, "linux"), [
    { platform: "windows", keys: ["ctrl+z"] },
    { platform: "wsl", keys: ["alt+z"] },
  ]);
  assert.deepEqual(platformAlternates({ id: "x", defaults: ["up"] }, "linux"), []);
});
