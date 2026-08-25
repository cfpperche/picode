import assert from "node:assert/strict";
import { test } from "node:test";
import { resolveLayer, catalogBase } from "./resolveLayer.js";

test("workspace wins over global like skills", () => {
  const global = { defaultProvider: "xai", defaultModel: "grok-4.6", compactionEnabled: true };
  const project = { has: { defaultProvider: true }, defaultProvider: "openai" };
  const got = resolveLayer(project, global);
  assert.equal(got.defaultProvider, "openai");
  assert.equal(got.defaultModel, "grok-4.6");
  assert.equal(got.compactionEnabled, true);
});

test("workspace model patterns beat global", () => {
  const got = resolveLayer(
    { has: { enabledModels: true }, enabledModels: ["gpt-4o"] },
    { enabledModels: ["claude-*"] },
  );
  assert.deepEqual(got.enabledModels, ["gpt-4o"]);
});

test("catalog base fills the floor", () => {
  const b = catalogBase({ providers: [{ id: "xai", models: [{ id: "grok-4.6" }] }] });
  assert.equal(b.defaultProvider, "xai");
  assert.equal(b.defaultModel, "grok-4.6");
});
