import { modelChipLabel, providerChipLabel, providerChoices, modelChoices, filterChoices, shortModel } from "./chip.js";
import assert from "node:assert/strict";
import { test } from "node:test";

const catalog = {
  providers: [
    { id: "xai", signedIn: true, models: [{ id: "grok-4.5", thinking: true }] },
    { id: "anthropic", signedIn: false, models: [{ id: "claude-sonnet-4-5", thinking: true }] },
  ],
};

test("empty chips name the field", () => {
  assert.equal(providerChipLabel({}), "Provider");
  assert.equal(modelChipLabel({}), "Model");
});

test("model plus thinking", () => {
  assert.equal(modelChipLabel({ model: "claude-sonnet-4-5", thinking: "medium" }), "claude-sonnet-4-5 · medium");
});

test("provider choices prefer signed-in", () => {
  const opts = providerChoices(catalog, "");
  assert.deepEqual(opts.map((o) => o.id), ["", "xai"]);
});

test("current unsigned provider stays listed", () => {
  const opts = providerChoices(catalog, "anthropic");
  assert.ok(opts.some((o) => o.id === "anthropic"));
});

test("models scoped to provider", () => {
  assert.equal(modelChoices(catalog, "").length, 0);
  assert.equal(modelChoices(catalog, "xai")[0].id, "grok-4.5");
});

test("filterChoices is case-insensitive", () => {
  const hits = filterChoices([{ id: "a", label: "claude-opus-4" }], "OPUS");
  assert.equal(hits.length, 1);
});

test("shortens long ids", () => {
  assert.ok(shortModel("claude-very-long-model-name-xyz").length < 30);
});
