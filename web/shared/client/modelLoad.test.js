import test from "node:test";
import assert from "node:assert/strict";
import { modelLoadChanges, doneLine, loadingLine, hostOf } from "./modelLoad.js";

const form = {
  baseUrl: "https://api.example.com/v1",
  api: "openai-completions",
  modelsText: "typed-one",
  contextWindow: "",
  maxTokens: "",
};

test("a loaded list merges in and fills what the endpoint agreed on", () => {
  const { changes, line } = modelLoadChanges(
    {
      models: [
        { id: "typed-one", contextWindow: 200000, maxTokens: 65536 },
        { id: "m-2", contextWindow: 200000, maxTokens: 65536 },
      ],
    },
    form,
  );
  assert.equal(changes.modelsText, "typed-one\nm-2");
  assert.equal(changes.contextWindow, "200000");
  assert.equal(changes.maxTokens, "65536");
  assert.equal(line, "Found 2 models. Added 1 to the list. Filled context window and max output from the endpoint.");
});

// Models that disagree leave the fields blank — and the dialog says so, rather
// than leaving a person wondering why nothing was filled.
test("a list with differing limits fills nothing and says why", () => {
  const { changes, line } = modelLoadChanges(
    {
      models: [
        { id: "big", contextWindow: 1000000, maxTokens: 384000 },
        { id: "small", contextWindow: 128000, maxTokens: 16384 },
      ],
    },
    form,
  );
  assert.equal(changes.contextWindow, undefined);
  assert.equal(changes.maxTokens, undefined);
  assert.ok(line.includes("different limits per model"), line);
});

// A list that reports no limits at all is not a "mixed" list: there is nothing
// to explain.
test("a list with no limits reports only the ids", () => {
  const { changes, line } = modelLoadChanges({ models: [{ id: "a" }, { id: "b" }] }, form);
  assert.equal(changes.contextWindow, undefined);
  assert.equal(line, "Found 2 models. Added 2 to the list.");
});

// A typed number is never overwritten by the endpoint's.
test("typed context and max output win over the endpoint's", () => {
  const { changes, line } = modelLoadChanges(
    { models: [{ id: "m", contextWindow: 200000, maxTokens: 65536 }] },
    { ...form, contextWindow: "8000", maxTokens: "512" },
  );
  assert.equal(changes.contextWindow, undefined);
  assert.equal(changes.maxTokens, undefined);
  assert.equal(line, "Found 1 model. Added 1 to the list.");
});

test("nothing new is reported as nothing new, and the empty list as itself", () => {
  const { line } = modelLoadChanges({ models: [{ id: "typed-one" }] }, form);
  assert.equal(line, "Found 1 model. Everything it lists is already here.");
  assert.equal(doneLine(1, 0, []), "Found 1 model. Everything it lists is already here.");
  assert.equal(doneLine(0, 0, []), "The endpoint reports no models — check the API type, or type the ids by hand.");
});

test("the loading line names the host it is asking", () => {
  assert.equal(loadingLine("https://api.cheaperinference.com/v1"), "Asking api.cheaperinference.com for its model list…");
  assert.equal(hostOf(""), "the endpoint");
  assert.equal(loadingLine("notaurl"), "Asking the endpoint for its model list…");
  assert.equal(hostOf("http://127.0.0.1:8080"), "127.0.0.1:8080");
});
