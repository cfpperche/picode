import test from "node:test";
import assert from "node:assert/strict";
import {
  customModelIds, validateCustomProvider, customProviderPayload, customProviderForm, customTakenIds,
  customThinkingLevels, customThinkingFormat, mergeModelIds, syncModelLimits, limitsFromDefinitions, modelLimitRow,
  THINKING_LEVELS, DEFAULT_THINKING_LEVELS, THINKING_FORMAT_NEEDS,
} from "./customProviders.js";
import { CHAT_TEMPLATE_VARS, customApiHint, CUSTOM_PROVIDER_APIS, chatTemplateObjectError } from "../contracts/schemas.js";

const good = {
  id: "CheaperInference",
  baseUrl: "https://api.cheaperinference.com/v1",
  api: "openai-completions",
  modelsText: "gpt-5.4\ngemini-3.7-flash\n",
  modelLimits: { "gpt-5.4": { contextWindow: "400000", maxTokens: "" } },
  compatDeveloper: false,
  compatReasoning: false,
  thinkingFormat: "",
  chatTemplateKwargs: "",
  chatTemplateArgs: "",
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
    ["zero context for a model", { ...good, modelLimits: { "gpt-5.4": { contextWindow: "0", maxTokens: "" } } }],
    ["fractional max for a model", { ...good, modelLimits: { "gpt-5.4": { contextWindow: "", maxTokens: "1.5" } } }],
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

test("payload carries models, per-model sizes and compat", () => {
  const r = validateCustomProvider(good);
  const payload = customProviderPayload(r.value);
  assert.deepEqual(payload.models, [
    { id: "gpt-5.4", contextWindow: 400000, reasoning: false },
    { id: "gemini-3.7-flash", reasoning: false },
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

test("the thinking format rides at the top level, empty meaning pi's default", () => {
  const set = validateCustomProvider({ ...good, thinkingFormat: "deepseek" });
  assert.equal(set.ok, true);
  assert.equal(customProviderPayload(set.value).thinkingFormat, "deepseek");

  const unset = validateCustomProvider(good);
  assert.equal(unset.ok, true);
  assert.equal("thinkingFormat" in customProviderPayload(unset.value), false);

  const invented = validateCustomProvider({ ...good, thinkingFormat: "mind-meld" });
  assert.equal(invented.ok, false);
});

test("customProviderForm prefills from a catalog row and starts blank", () => {
  const row = {
    id: "cheaperinference",
    baseUrl: "https://api.cheaperinference.com/v1",
    api: "anthropic-messages",
    compat: { supportsDeveloperRole: true },
    thinkingFormat: "deepseek",
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
    modelLimits: {
      "claude-sonnet-5": { contextWindow: "200000", maxTokens: "32000" },
      "claude-opus-5": { contextWindow: "", maxTokens: "" },
    },
    compatDeveloper: true,
    compatReasoning: false,
    thinkingFormat: "deepseek",
    chatTemplateKwargs: "",
    chatTemplateArgs: "",
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

// Every API type the form offers carries its own hint, so no type can leave the
// base-URL field unexplained.
test("each API type ships a base-URL hint and placeholder", () => {
  for (const api of CUSTOM_PROVIDER_APIS) {
    assert.ok(api.urlHint && api.urlHint.length > 20, api.value + " has no hint");
    assert.ok(api.placeholder.startsWith("Base URL"), api.value + " has no placeholder");
  }
  assert.equal(customApiHint("anthropic-messages").value, "anthropic-messages");
  // A type nobody listed still gets a hint instead of a blank line.
  assert.equal(customApiHint("future-type").value, CUSTOM_PROVIDER_APIS[0].value);
});

// The chat template object rides only with the format that reads it, and the
// form writes pi's own value for the OpenAI branch.
test("thinking format writes pi's value and carries the object it needs", () => {
  const base = { ...good, reasoningModel: false, thinkingLevels: [] };

  const openai = validateCustomProvider({ ...base, thinkingFormat: "openai" });
  assert.equal(openai.ok, true);
  assert.equal(customProviderPayload(openai.value).thinkingFormat, "openai");

  const chat = validateCustomProvider({
    ...base, thinkingFormat: "chat-template", chatTemplateKwargs: '{"enable_thinking": true}',
  });
  assert.equal(chat.ok, true, chat.error);
  const payload = customProviderPayload(chat.value);
  assert.deepEqual(payload.chatTemplateKwargs, { enable_thinking: true });
  assert.equal(payload.chatTemplateArgs, undefined);

  // The object is dropped (not carried along) for a format that ignores it.
  const plain = customProviderPayload(validateCustomProvider({ ...base, thinkingFormat: "zai", chatTemplateKwargs: '{"x": true}' }).value);
  assert.equal(plain.chatTemplateKwargs, undefined);

  // baseten reads the other object.
  const baseten = customProviderPayload(validateCustomProvider({ ...base, thinkingFormat: "baseten", chatTemplateArgs: '{"thinking": {"$var": "thinking.enabled"}}' }).value);
  assert.deepEqual(baseten.chatTemplateArgs, { thinking: { $var: "thinking.enabled" } });
  assert.equal(baseten.chatTemplateKwargs, undefined);
});

test("a bad chat template object is refused with a message, not saved", () => {
  const base = { ...good, thinkingFormat: "chat-template" };
  for (const [raw, want] of [
    ["{", "not valid JSON"],
    ["[1]", "use a JSON object"],
    ['{"t": {"$var": "nope"}}', '"$var" must be one of'],
    ['{"t": {"$var": "thinking.enabled", "x": 1}}', 'only "$var" and "omitWhenOff"'],
    ['{"t": [1]}', "use a string, number"],
  ]) {
    const res = validateCustomProvider({ ...base, chatTemplateKwargs: raw });
    assert.equal(res.ok, false, raw + " was accepted");
    assert.ok(res.error.includes(want), raw + " → " + res.error);
  }
  // The three pi-controlled references and the omit flag are accepted.
  for (const ref of CHAT_TEMPLATE_VARS) {
    assert.equal(chatTemplateObjectError(`{"t": {"$var": "${ref}"}}`), "", ref);
  }
  assert.equal(chatTemplateObjectError('{"t": {"$var": "thinking.budget", "omitWhenOff": false}}'), "");
});

// A hand-edited file may carry the undocumented value the old form wrote; the
// form reads it as the branch pi actually takes and writes it explicitly next
// time. Anything the form does not offer reads as "pi's default".
test("the thinking-format prefill normalizes what the file says", () => {
  assert.equal(customThinkingFormat("reasoning_effort"), "openai");
  assert.equal(customThinkingFormat("chat-template"), "chat-template");
  assert.equal(customThinkingFormat(""), "");
  assert.equal(customThinkingFormat("mind-meld"), "");
  assert.equal(customThinkingFormat(undefined), "");
  assert.deepEqual(Object.keys(THINKING_FORMAT_NEEDS).filter((k) => THINKING_FORMAT_NEEDS[k]), ["chat-template", "baseten"]);
  const form = customProviderForm({
    id: "gw",
    compat: { thinkingFormat: "chat-template", chatTemplateKwargs: { enable_thinking: true } },
    thinkingFormat: "chat-template",
    chatTemplateKwargs: { thinking: { $var: "thinking.enabled" } },
    definitions: [{ id: "m" }],
  });
  assert.equal(form.thinkingFormat, "chat-template");
  assert.deepEqual(JSON.parse(form.chatTemplateKwargs), { thinking: { $var: "thinking.enabled" } });
  assert.equal(form.chatTemplateArgs, "");
});

// Loading a list must never cost the typed ids: it appends what is missing, in
// the endpoint's order, and reports how many it added.
test("mergeModelIds appends only what is missing, keeping typed order", () => {
  const found = [{ id: "glm-4.6" }, { id: "deepseek-v4.1-flash" }, { id: "glm-4.6" }, { id: "" }, null];
  const merged = mergeModelIds("mine-1\n  glm-4.6  \n\nmine-2", found);
  assert.equal(merged.text, "mine-1\nglm-4.6\nmine-2\ndeepseek-v4.1-flash");
  assert.equal(merged.added, 1);

  const nothing = mergeModelIds("a\nb", [{ id: "b" }, { id: "a" }]);
  assert.equal(nothing.text, "a\nb");
  assert.equal(nothing.added, 0);
  assert.deepEqual(mergeModelIds("", []), { text: "", added: 0 });
});

// The form writes one context window and one max output for the whole list, so
// it only fills them when every listed model reports the same number.
test("limits are per model, and a row follows its id", () => {
  const defs = [
    { id: "big", contextWindow: 1000000, maxTokens: 384000 },
    { id: "small" },
  ];
  const limits = limitsFromDefinitions(defs);
  assert.deepEqual(limits, {
    big: { contextWindow: "1000000", maxTokens: "384000" },
    small: { contextWindow: "", maxTokens: "" },
  });

  // A row appears and disappears with the id list; typed numbers survive.
  const two = { ...limits, small: { contextWindow: "128000", maxTokens: "" } };
  const synced = syncModelLimits(two, ["small", "new"]);
  assert.deepEqual(Object.keys(synced), ["small", "new"]);
  assert.equal(synced.small.contextWindow, "128000");
  assert.deepEqual(synced.new, { contextWindow: "", maxTokens: "" });
  assert.deepEqual(modelLimitRow(undefined, "x"), { contextWindow: "", maxTokens: "" });
});

// The form used to write one context window and max output for every id, which
// had to refuse a gateway whose models disagreed. Each id carries its own now.
test("the payload carries each model's own numbers", () => {
  const parsed = validateCustomProvider({
    ...good,
    modelsText: "big\nsmall\nbare",
    modelLimits: {
      big: { contextWindow: "1000000", maxTokens: "384000" },
      small: { contextWindow: "128000", maxTokens: "" },
      bare: { contextWindow: "", maxTokens: "" },
      gone: { contextWindow: "5", maxTokens: "5" },
    },
  });
  assert.equal(parsed.ok, true, parsed.error);
  const models = customProviderPayload(parsed.value).models;
  assert.deepEqual(models, [
    { id: "big", contextWindow: 1000000, maxTokens: 384000, reasoning: false },
    { id: "small", contextWindow: 128000, reasoning: false },
    { id: "bare", reasoning: false },
  ]);
});

test("a per-model row with a bad number names the model", () => {
  const parsed = validateCustomProvider({
    ...good,
    modelLimits: { "glm-4.6": { contextWindow: "0", maxTokens: "abc" } },
  });
  assert.equal(parsed.ok, false);
  assert.match(parsed.error, /glm-4\.6: context window must be a positive whole number/);
  const ok = validateCustomProvider({ ...good, modelLimits: { "glm-4.6": { contextWindow: "", maxTokens: "" } } });
  assert.equal(ok.ok, true, ok.error);
});
