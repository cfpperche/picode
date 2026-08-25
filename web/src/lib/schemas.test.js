import { test } from "node:test";
import assert from "node:assert/strict";
import { createWorkspaceSchema, createFreeAgentSchema, parseForm } from "./schemas.js";

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
