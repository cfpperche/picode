import test from "node:test";
import assert from "node:assert/strict";
import { modelLoadChanges, doneLine, loadingLine, hostOf } from "./modelLoad.js";

const form = {
  baseUrl: "https://api.example.com/v1",
  api: "openai-completions",
  modelsText: "typed-one",
  modelLimits: {},
};

test("a loaded list fills each model's own limits", () => {
  const { changes, line } = modelLoadChanges(
    {
      models: [
        { id: "typed-one", contextWindow: 1000000, maxTokens: 384000 },
        { id: "m-2", contextWindow: 128000, maxTokens: 16384 },
      ],
    },
    form,
  );
  assert.equal(changes.modelsText, "typed-one\nm-2");
  const blank = { name: "", input: [], cost: { input: "", output: "", cacheRead: "", cacheWrite: "" } };
  assert.deepEqual(changes.modelLimits, {
    "typed-one": { contextWindow: "1000000", maxTokens: "384000", ...blank },
    "m-2": { contextWindow: "128000", maxTokens: "16384", ...blank },
  });
  assert.equal(line, "Found 2 models. Added 1 to the list. Filled 4 limits from the endpoint.");
});

// A typed number is never overwritten by the endpoint's, and a gateway whose
// models disagree is filled correctly instead of refused.
test("typed limits win, and disagreement is not a problem", () => {
  const { changes, line } = modelLoadChanges(
    { models: [{ id: "a", contextWindow: 1000000, maxTokens: 384000 }, { id: "b", contextWindow: 128000, maxTokens: 16384 }] },
    { ...form, modelsText: "a\nb", modelLimits: { a: { contextWindow: "8000", maxTokens: "512" } } },
  );
  const blank = { name: "", input: [], cost: { input: "", output: "", cacheRead: "", cacheWrite: "" } };
  assert.deepEqual(changes.modelLimits.a, { contextWindow: "8000", maxTokens: "512", ...blank });
  assert.deepEqual(changes.modelLimits.b, { contextWindow: "128000", maxTokens: "16384", ...blank });
  assert.equal(line, "Found 2 models. Everything it lists is already here. Filled 2 limits from the endpoint.");
});

test("nothing new is reported as nothing new, and the empty list as itself", () => {
  const { line } = modelLoadChanges({ models: [{ id: "typed-one" }] }, form);
  assert.equal(line, "Found 1 model. Everything it lists is already here.");
  assert.equal(doneLine(0, 0, 0), "The endpoint reports no models — check the API type, or type the ids by hand.");
});

test("the loading line names the host it is asking", () => {
  assert.equal(loadingLine("https://api.cheaperinference.com/v1"), "Asking api.cheaperinference.com for its model list…");
  assert.equal(hostOf(""), "the endpoint");
  assert.equal(loadingLine("notaurl"), "Asking the endpoint for its model list…");
  assert.equal(hostOf("http://127.0.0.1:8080"), "127.0.0.1:8080");
});
