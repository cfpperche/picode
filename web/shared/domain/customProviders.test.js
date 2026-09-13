import test from "node:test";
import assert from "node:assert/strict";
import {
  customModelIds, validateCustomProvider, customProviderPayload, customProviderForm, customTakenIds,
  customThinkingLevels, THINKING_LEVELS, DEFAULT_THINKING_LEVELS,
} from "./customProviders.js";

const good = {
  id: "CheaperInference",
  baseUrl: "https://api.cheaperinference.com/v1",
  api: "openai-completions",
  modelsText: "gpt-5.4\ngemini-3.7-flash\n",
  contextWindow: "400000",
  maxTokens: "",
  compatDeveloper: false,
  compatReasoning: false,
  reasoningModel: false,
  thinkingLevels: [],
  key: "ci_live_x",
};

test("validate lowercases the id and accepts the happy path", () => {
  const r = validateCustomProvider(good);
  assert.equal(r.ok, true);
  assert.equal(r.value.id, "cheaperinference");
});

test("validate refuses the decision-table rows", () => {
  const bad = [
    ["blank name", { ...good, id: "" }],
    ["uppercase dash name", { ...good, id: "-x" }, {}],
    ["dot in name", { ...good, id: "my.gw" }],
    ["builtin name", { ...good, id: "anthropic" }, { takenIds: ["anthropic", "openai"] }],
    ["taken name", { ...good, id: "cheaperinference" }, { takenIds: ["cheaperinference"] }],
    ["no scheme", { ...good, baseUrl: "api.example.com/v1" }],
    ["ftp scheme", { ...good, baseUrl: "ftp://api.example.com/v1" }],
    ["no models", { ...good, modelsText: "  \n" }],
    ["space in model id", { ...good, modelsText: "openai gpt" }],
    ["dup model ids", { ...good, modelsText: "m\nm" }],
    ["zero context", { ...good, contextWindow: "0" }],
    ["fractional max", { ...good, maxTokens: "1.5" }],
    ["missing key on create", { ...good, key: " " }],
    ["bad api", { ...good, api: "ollama" }],
  ];
  for (const [name, form, opts] of bad) {
    const r = validateCustomProvider(form, opts);
    assert.equal(r.ok, false, name + " should fail");
  }
});

test("blank key passes on edit and is omitted from the payload", () => {
  const r = validateCustomProvider({ ...good, key: "" }, { requireKey: false });
  assert.equal(r.ok, true);
  const payload = customProviderPayload(r.value);
  assert.equal("key" in payload, false);
});

test("payload carries models, sizes and compat; sizes only when set", () => {
  const r = validateCustomProvider(good);
  const payload = customProviderPayload(r.value);
  assert.deepEqual(payload.models, [
    { id: "gpt-5.4", contextWindow: 400000, reasoning: false },
    { id: "gemini-3.7-flash", contextWindow: 400000, reasoning: false },
  ]);
  assert.deepEqual(payload.compat, { supportsDeveloperRole: false, supportsReasoningEffort: false });
  assert.equal(payload.key, "ci_live_x");
  assert.equal(payload.api, "openai-completions");
});

// Decision table for the thinking-level row: a reasoning model sends its
// selection as levels, a non-reasoning model sends reasoning:false and no
// levels, and an empty selection never reaches the wire (the schema refuses
// it first).
test("thinking levels ride along only for a reasoning model", () => {
  const base = { ...good, reasoningModel: true };
  const rows = [
    ["full scale", THINKING_LEVELS, THINKING_LEVELS],
    ["top only", ["max"], ["max"]],
    ["pi default set", DEFAULT_THINKING_LEVELS, DEFAULT_THINKING_LEVELS],
    ["order forced to the scale", ["max", "high"], ["high", "max"]],
  ];
  for (const [name, picked, expected] of rows) {
    const r = validateCustomProvider({ ...base, thinkingLevels: picked });
    assert.equal(r.ok, true, name);
    const payload = customProviderPayload(r.value);
    assert.deepEqual(payload.models[0].thinkingLevels, expected, name);
    assert.equal(payload.models[0].reasoning, true, name);
  }

  const off = validateCustomProvider(good);
  assert.equal(off.ok, true);
  const payload = customProviderPayload(off.value);
  assert.equal(payload.models[0].reasoning, false);
  assert.equal("thinkingLevels" in payload.models[0], false);
});

test("a reasoning model with no level selected is refused", () => {
  const r = validateCustomProvider({ ...good, reasoningModel: true, thinkingLevels: [] });
  assert.equal(r.ok, false);
  assert.match(r.error, /at least one thinking level/);
});

test("an unknown level is refused by the schema", () => {
  const r = validateCustomProvider({ ...good, reasoningModel: true, thinkingLevels: ["turbo"] });
  assert.equal(r.ok, false);
});

test("customModelIds trims and drops blanks", () => {
  assert.deepEqual(customModelIds(" a \n\nb\n"), ["a", "b"]);
  assert.deepEqual(customModelIds(""), []);
});

test("customProviderForm prefills from a catalog row and starts blank", () => {
  const row = {
    id: "cheaperinference",
    baseUrl: "https://api.cheaperinference.com/v1",
    api: "anthropic-messages",
    compat: { supportsDeveloperRole: true },
    definitions: [
      { id: "claude-opus-5" },
      { id: "claude-sonnet-5", contextWindow: 200000, maxTokens: 32000 },
    ],
  };
  const form = customProviderForm(row);
  assert.deepEqual(form, {
    id: "cheaperinference",
    baseUrl: "https://api.cheaperinference.com/v1",
    api: "anthropic-messages",
    modelsText: "claude-opus-5\nclaude-sonnet-5",
    contextWindow: "200000",
    maxTokens: "32000",
    compatDeveloper: true,
    compatReasoning: false,
    reasoningModel: false,
    thinkingLevels: DEFAULT_THINKING_LEVELS,
    key: "",
  });
  assert.equal(customProviderForm(null).api, "openai-completions");
});

// The prefill mirrors what the file says, so an untouched Edit writes back
// the same levels: an explicit map wins, a reasoning model without one gets
// pi's default set.
test("thinking-level prefill reads the map, the levels list, or pi's default", () => {
  const map = { id: "m", reasoning: true, thinkingLevelMap: { minimal: null, low: null, medium: null, high: "high", max: "max" } };
  const listed = { id: "m", reasoning: true, thinkingLevels: ["low", "high"] };
  const bare = { id: "m", reasoning: true };
  assert.deepEqual(customThinkingLevels([map]), ["high", "max"]);
  assert.deepEqual(customThinkingLevels([listed]), ["low", "high"]);
  assert.deepEqual(customThinkingLevels([bare]), DEFAULT_THINKING_LEVELS);
  assert.deepEqual(customThinkingLevels([]), DEFAULT_THINKING_LEVELS);
  assert.deepEqual(customThinkingLevels([{ id: "m", thinkingLevelMap: { off: null, high: "high" } }]), ["high"]);
});

test("customTakenIds keeps the edited provider claimable", () => {
  const providers = [{ id: "anthropic" }, { id: "cheaperinference" }, { id: "openai" }];
  assert.deepEqual(customTakenIds(providers, "cheaperinference"), ["anthropic", "openai"]);
});
