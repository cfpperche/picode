import { test } from "node:test";
import assert from "node:assert/strict";
import { faceSlice, FACE_MAX, providerFaviconUrl, providerLetter, workspaceAgents } from "./providerIcon.js";

test("faceSlice caps at 5", () => {
  const agents = [1, 2, 3, 4, 5, 6, 7].map((n) => ({ id: String(n) }));
  const { shown, extra } = faceSlice(agents);
  assert.equal(shown.length, FACE_MAX);
  assert.equal(extra, 2);
  assert.equal(shown[0].id, "1");
});

test("providerFaviconUrl", () => {
  assert.ok(providerFaviconUrl("xai").includes("grok.svg"));
  assert.ok(providerFaviconUrl("anthropic").includes("claude.svg"));
  assert.equal(providerFaviconUrl("nope"), "");
  assert.equal(providerLetter("xai"), "X");
});

test("workspaceAgents follows sidebar order", () => {
  const ws = [
    { agents: [{ id: "a", provider: "xai" }] },
    { agents: [{ id: "b", provider: "openai" }, { id: "c", provider: "anthropic" }] },
  ];
  assert.deepEqual(workspaceAgents(ws).map((a) => a.id), ["a", "b", "c"]);
});

test("workspaceAgents drops the id-less fallback agent (empty workspace)", () => {
  assert.deepEqual(workspaceAgents([{ id: "w", agents: [], agent: {} }]), []);
  assert.deepEqual(workspaceAgents([{ id: "w" }]), []);
});
