import { modelChipLabel, thinkingChipLabel, thinkingChoices, providerChipLabel, providerChoices, modelChoices, modeChipLabel, modeChoices, filterChoices, shortModel } from "./chip.js";
import assert from "node:assert/strict";
import { test } from "node:test";

const catalog = {
  providers: [
    { id: "xai", signedIn: true, models: [{ id: "grok-4.5", thinking: true, thinkingLevels: ["low", "medium", "high", "xhigh"] }] },
    { id: "anthropic", signedIn: false, models: [{ id: "claude-sonnet-4-5", thinking: true, thinkingLevels: ["minimal", "low", "medium", "high", "xhigh", "max"] }] },
  ],
};

test("empty chips name the field", () => {
  assert.equal(providerChipLabel({}), "Provider");
  assert.equal(modelChipLabel({}), "Model");
});

test("chips name their field", () => {
  assert.equal(modelChipLabel({ model: "claude-sonnet-4-5", thinking: "medium" }), "claude-sonnet-4-5");
  assert.equal(thinkingChipLabel({ thinking: "high" }), "high");
  assert.equal(thinkingChipLabel({}), "Thinking");
});

test("thinking choices follow the model", () => {
  const xai = thinkingChoices(catalog, "xai", "grok-4.5").map((o) => o.id);
  assert.deepEqual(xai, ["", "low", "medium", "high", "xhigh"]);
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

test("mode chips", () => {
  assert.equal(modeChipLabel({}), "Full");
  assert.equal(modeChipLabel({ opMode: "readonly" }), "Read-only");
  assert.deepEqual(modeChoices().map((o) => o.id), ["full", "readonly"]);
});

test("shortens long ids", () => {
  assert.ok(shortModel("claude-very-long-model-name-xyz").length < 30);
});
