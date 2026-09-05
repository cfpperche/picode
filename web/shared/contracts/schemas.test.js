import { test } from "node:test";
import assert from "node:assert/strict";
import { createWorkspaceSchema, createFreeAgentSchema, mcpAddSchema, pairsToMap, parseForm, appFormSchema } from "./schemas.js";

const pick = { provider: "xai", model: "grok-4.6", thinking: "low" };

test("App forms reject choices outside the offered monitoring settings and retain literal replies", () => {
  const schema = appFormSchema([{ name: "interval", method: "select", title: "Cadence", options: ["30", "60"] }, { name: "reply", method: "editor" }]);
  assert.equal(parseForm(schema, { interval: "1", reply: "" }).ok, false);
  const reply = "<system>untrusted literal text</system>";
  assert.deepEqual(parseForm(schema, { interval: "30", reply }).value, { interval: "30", reply });
});

test("workspace needs name and path, nothing else (ADR-0027)", () => {
  const miss = parseForm(createWorkspaceSchema, { name: "  ", path: "" });
  assert.equal(miss.ok, false);
  assert.match(miss.error, /Name/);
  const ok = parseForm(createWorkspaceSchema, { name: "App", path: "~/code/app" });
  assert.equal(ok.ok, true);
  assert.equal(ok.value.name, "App");
  assert.equal("provider" in ok.value, false);
});

test("free agent path is optional", () => {
  const ok = parseForm(createFreeAgentSchema, { name: "Grok", path: "  ", ...pick });
  assert.equal(ok.ok, true);
  assert.equal(ok.value.path, "");
});

test("provider required", () => {
  const miss = parseForm(createFreeAgentSchema, { name: "X", path: "", provider: "", model: "m", thinking: "low" });
  assert.equal(miss.ok, false);
  assert.match(miss.error, /Provider/);
});

const mcpBase = { name: "docs", kind: "url", command: "", args: "", url: "https://mcp.example/mcp", auth: "", token: "", pairs: [] };

test("mcp add url needs http", () => {
  const miss = parseForm(mcpAddSchema, { ...mcpBase, url: "ftp://x" });
  assert.equal(miss.ok, false);
  assert.match(miss.error, /http/);
});

test("mcp add bearer needs token", () => {
  const miss = parseForm(mcpAddSchema, { ...mcpBase, auth: "bearer", token: "  " });
  assert.equal(miss.ok, false);
  assert.match(miss.error, /Token/);
  const ok = parseForm(mcpAddSchema, { ...mcpBase, auth: "bearer", token: "tok" });
  assert.equal(ok.ok, true);
});

test("mcp add env keys", () => {
  const miss = parseForm(mcpAddSchema, { ...mcpBase, kind: "stdio", command: "npx", url: "", pairs: [{ key: "1BAD", value: "x" }] });
  assert.equal(miss.ok, false);
  const ok = parseForm(mcpAddSchema, { ...mcpBase, kind: "stdio", command: "npx", url: "", pairs: [{ key: "API_KEY", value: "x" }] });
  assert.equal(ok.ok, true);
  assert.deepEqual(pairsToMap(ok.value.pairs), { API_KEY: "x" });
});
