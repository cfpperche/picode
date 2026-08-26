import { test } from "node:test";
import assert from "node:assert/strict";
import { createWorkspaceSchema, createFreeAgentSchema, mcpAddSchema, pairsToMap, parseForm } from "./schemas.js";

const pick = { provider: "xai", model: "grok-4.6", thinking: "low" };

test("workspace needs name and path", () => {
  const miss = parseForm(createWorkspaceSchema, { name: "  ", path: "", ...pick });
  assert.equal(miss.ok, false);
  assert.match(miss.error, /Name/);
  const ok = parseForm(createWorkspaceSchema, { name: "App", path: "~/code/app", ...pick });
  assert.equal(ok.ok, true);
  assert.equal(ok.value.name, "App");
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
