import { chipLabel, shortModel } from "./chip.js";
import assert from "node:assert/strict";
import { test } from "node:test";

test("empty config is Default model", () => {
  assert.equal(chipLabel({}), "Default model");
  assert.equal(chipLabel({ provider: "", model: "", thinking: "" }), "Default model");
});

test("model plus thinking", () => {
  assert.equal(chipLabel({ model: "claude-sonnet-4-5", thinking: "medium" }), "claude-sonnet-4-5 · medium");
});

test("provider only", () => {
  assert.equal(chipLabel({ provider: "anthropic" }), "anthropic");
});

test("shortens long ids", () => {
  assert.ok(shortModel("claude-very-long-model-name-xyz").length < 30);
});
