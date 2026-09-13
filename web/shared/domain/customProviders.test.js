import test from "node:test";
import assert from "node:assert/strict";
import {
  customModelIds, validateCustomProvider, customProviderPayload, customProviderForm, customTakenIds,
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
    { id: "gpt-5.4", contextWindow: 400000 },
    { id: "gemini-3.7-flash", contextWindow: 400000 },
  ]);
  assert.deepEqual(payload.compat, { supportsDeveloperRole: false, supportsReasoningEffort: false });
  assert.equal(payload.key, "ci_live_x");
  assert.equal(payload.api, "openai-completions");
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
    key: "",
  });
  assert.equal(customProviderForm(null).api, "openai-completions");
});

test("customTakenIds keeps the edited provider claimable", () => {
  const providers = [{ id: "anthropic" }, { id: "cheaperinference" }, { id: "openai" }];
  assert.deepEqual(customTakenIds(providers, "cheaperinference"), ["anthropic", "openai"]);
});
