import assert from "node:assert/strict";
import { test } from "node:test";
import { KEYS } from "./termKeyBar.js";

test("accessory keys are one Fable-first row, hide is not in the scroller", () => {
  assert.equal(KEYS.length, 17);
  assert.deepEqual(KEYS.map((k) => k.id).slice(0, 9), [
    "esc", "tab", "ctrl", "alt", "left", "up", "down", "right", "intr",
  ]);
  assert.equal(KEYS.filter((k) => k.mod).map((k) => k.mod).join(","), "ctrl,alt");
  assert.equal(KEYS.find((k) => k.id === "intr").seq, "\x03");
  assert.equal(new Set(KEYS.map((k) => k.id)).size, KEYS.length);
  assert.ok(!KEYS.some((k) => k.id === "hide" || k.id === "keyboard"));
});
