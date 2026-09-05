import { bashLine } from "./bashLine.js";
import assert from "node:assert/strict";
import { test } from "node:test";

test("whole draft !cmd is bash", () => {
  assert.deepEqual(bashLine("!ls -la"), { command: "ls -la" });
  assert.deepEqual(bashLine("  ! git status  "), { command: "git status" });
});

test("!! is refused (TUI-only)", () => {
  assert.deepEqual(bashLine("!!ls"), { refused: "!!" });
});

test("bare ! and prose are not bash", () => {
  assert.equal(bashLine("!"), null);
  assert.equal(bashLine(""), null);
  assert.equal(bashLine("see !ls please"), null);
  assert.equal(bashLine("email@example.com"), null);
});
