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

test("the resolver carries pi's machine-only keys", () => {
  // A fixed list drops whatever it was not told about: the Theme row rendered
  // empty while the file said `dark` (2026-09-20).
  const layer = { theme: "dark", hideThinkingBlock: true, has: { theme: true, hideThinkingBlock: true } };
  const got = resolveLayer(layer, catalogBase(null));
  assert.equal(got.theme, "dark");
  assert.equal(got.hideThinkingBlock, true);
  // Unset keys fall back to what pi itself does.
  assert.equal(got.quietStartup, false);
  assert.equal(got.defaultProjectTrust, "ask");
  assert.equal(got.shellPath, "");
  // And a layer that sets nothing inherits its parent, not a fresh default.
  const child = resolveLayer({ has: {} }, got);
  assert.equal(child.theme, "dark");
  assert.equal(child.hideThinkingBlock, true);
});
