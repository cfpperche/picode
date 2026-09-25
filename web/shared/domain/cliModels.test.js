import assert from "node:assert/strict";
import test from "node:test";
import {
  MODELS_CLIS, supportsCliModels, effectiveList, toggledList, isUnreadable, groupByProvider,
  filterModels, kindCounts, formatContext, formatPrice, patternsNotListed, stringList,
  splitAllowedExtras, allowedLeavesNothing, kindLabel,
} from "./cliModels.js";

const layers = [
  { scope: "user", label: "Global", values: { disabledProviders: ["ollama", "groq"] } },
  { scope: "project", label: "This workspace", values: {} },
];

test("only a CLI with a measured catalog command has the pane", () => {
  assert.deepEqual(MODELS_CLIS, ["omp"]);
  assert.equal(supportsCliModels("omp"), true);
  assert.equal(supportsCliModels("pi"), false);
});

test("a layer that does not set the list inherits the nearest one, whole", () => {
  assert.deepEqual(effectiveList(layers, "project", "disabledProviders"), { list: ["ollama", "groq"], setHere: false, from: "Global" });
  assert.deepEqual(effectiveList(layers, "user", "disabledProviders"), { list: ["ollama", "groq"], setHere: true, from: "" });
  assert.deepEqual(effectiveList(layers, "user", "enabledModels"), { list: [], setHere: false, from: "" });
});

// The trap omp's own docs call the most common surprise: a project array
// replaces the global one. Toggling on a layer that inherits must write the
// inherited entries too, or they silently stop applying there.
test("toggling on an inheriting layer writes the complete list", () => {
  assert.deepEqual(toggledList(layers, "project", "disabledProviders", "anthropic", true), ["ollama", "groq", "anthropic"]);
  assert.deepEqual(toggledList(layers, "project", "disabledProviders", "groq", false), ["ollama"]);
  assert.deepEqual(toggledList(layers, "user", "enabledModels", "openai/gpt-5", true), ["openai/gpt-5"]);
  // turning on what is already there does not duplicate it
  assert.deepEqual(toggledList(layers, "user", "disabledProviders", "groq", true), ["ollama", "groq"]);
});

test("folder-scoped entries are the server's to call unreadable, and never read as strings", () => {
  assert.equal(isUnreadable({ unreadable: ["enabledModels"] }, "enabledModels"), true);
  assert.equal(isUnreadable({}, "enabledModels"), false);
  assert.deepEqual(stringList(["a", { path: "~/x", models: ["b"] }]), ["a"]);
  assert.deepEqual(stringList("a"), ["a"]);
  assert.deepEqual(stringList(undefined), []);
});

const models = [
  { provider: "openai", id: "gpt-5-nano", name: "GPT-5 Nano", selector: "openai/gpt-5-nano", kind: "chat" },
  { provider: "local", id: "kokoro", name: "Kokoro-82M", selector: "local/kokoro", kind: "tts" },
  { provider: "openai", id: "gpt-4o-mini", name: "GPT-4o mini", selector: "openai/gpt-4o-mini", kind: "chat" },
];

test("rows group by provider, sorted, and filter by kind and text", () => {
  const groups = groupByProvider(models);
  assert.deepEqual(groups.map((g) => g.provider), ["local", "openai"]);
  assert.deepEqual(groups[1].models.map((m) => m.id), ["gpt-4o-mini", "gpt-5-nano"]);
  assert.deepEqual(filterModels(models, { kind: "tts" }).map((m) => m.id), ["kokoro"]);
  assert.deepEqual(filterModels(models, { query: "NANO" }).map((m) => m.id), ["gpt-5-nano"]);
  assert.deepEqual(kindCounts(models), { chat: 2, tts: 1 });
});

test("numbers read the way a catalog prints them, and unknown stays blank", () => {
  assert.equal(formatContext(1_000_000), "1M");
  assert.equal(formatContext(1_048_576), "1M");
  assert.equal(formatContext(128_000), "128K");
  assert.equal(formatContext(0), "");
  assert.equal(formatPrice({ input: 0.3, output: 1.2 }), "$0.30 / $1.20");
  assert.equal(formatPrice({ input: 3, output: 15 }), "$3 / $15");
  // 0 / 0 is both "local" and "no price listed": the pane does not pick one.
  assert.equal(formatPrice({ input: 0, output: 0 }), "");
  assert.equal(formatPrice(undefined), "");
});

test("an allowed entry that is not a reported selector is listed as written", () => {
  assert.deepEqual(patternsNotListed(["openai/gpt-5-nano", "claude-*", "gone/model"], models), ["claude-*", "gone/model"]);
});

// Measured on a scratch, 2026-09-22: the global list allowed only
// openai/gpt-4o-mini, the workspace hid openai, and omp was left with nothing.
test("an allowed list that matches nothing reachable is called out, a glob is not guessed", () => {
  assert.equal(allowedLeavesNothing(["openai/gpt-4o-mini"], [models[1]]), true);
  assert.equal(allowedLeavesNothing(["openai/gpt-4o-mini"], models), false);
  assert.equal(allowedLeavesNothing(["claude-*"], [models[1]]), false);
  assert.equal(allowedLeavesNothing([], []), false);
  assert.deepEqual(splitAllowedExtras(["claude-*", "sonnet", "gone/model"]), { patterns: ["claude-*", "sonnet"], unreachable: ["gone/model"] });
});

test("kinds read as words", () => {
  assert.equal(kindLabel("tts"), "Speech");
  assert.equal(kindLabel("stt"), "Dictation");
  assert.equal(kindLabel("tiny"), "Small");
  assert.equal(kindLabel("mystery"), "mystery");
});

test("the model list fills only the model field of a CLI with a reader", async () => {
  const { MODEL_READERS, modelPickField } = await import("./cliModels.js");
  assert.deepEqual(MODEL_READERS, ["omp", "pi", "codex", "opencode", "muse", "grok", "claude-code"]);
  assert.equal(modelPickField("codex", "model"), true);
  assert.equal(modelPickField("opencode", "model"), true);
  assert.equal(modelPickField("muse", "model"), true);
  assert.equal(modelPickField("muse", "provider"), false);
  assert.equal(modelPickField("claude-code", "model"), true);
  assert.equal(modelPickField("agy", "model"), false, "no reader: the field stays text only");
  assert.equal(modelPickField("grok", "models.default"), true);
  assert.equal(modelPickField("grok", "models.default_reasoning_effort"), false);
});
