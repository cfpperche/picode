import test from "node:test";
import assert from "node:assert/strict";
import { piInstalled, automationsBlockedByPi } from "./automationsPi.js";

const noPi = [{ id: "pi", name: "Pi", installed: false }, { id: "claude-code", name: "Claude Code", installed: true }];
const withPi = [{ id: "pi", name: "Pi", installed: true }];
const workspaces = [{ id: "w", agents: [{ id: "a-pi", cli: "pi" }, { id: "a-claude", cli: "claude-code" }, { id: "a-legacy" }] }];
const free = [{ id: "f-omp", cli: "omp" }];

test("piInstalled answers null until the catalog arrives", () => {
  assert.equal(piInstalled([]), null);
  assert.equal(piInstalled(noPi), false);
  assert.equal(piInstalled(withPi), true);
});

test("a start automation needs Pi", () => {
  assert.equal(automationsBlockedByPi(noPi, [{ action: "start" }], workspaces, free), true);
  assert.equal(automationsBlockedByPi(withPi, [{ action: "start" }], workspaces, free), false);
});

test("a message to a guest agent does not need Pi", () => {
  assert.equal(automationsBlockedByPi(noPi, [{ action: "message", targetAgentId: "a-claude" }], workspaces, free), false);
  assert.equal(automationsBlockedByPi(noPi, [{ action: "message", targetAgentId: "f-omp" }], workspaces, free), false);
});

test("a message to a Pi, legacy or unknown agent needs Pi", () => {
  assert.equal(automationsBlockedByPi(noPi, [{ action: "message", targetAgentId: "a-pi" }], workspaces, free), true);
  assert.equal(automationsBlockedByPi(noPi, [{ action: "message", targetAgentId: "a-legacy" }], workspaces, free), true);
  assert.equal(automationsBlockedByPi(noPi, [{ action: "message", targetAgentId: "gone" }], workspaces, free), true);
});

test("a disabled automation never blocks", () => {
  assert.equal(automationsBlockedByPi(noPi, [{ action: "start", enabled: false }], workspaces, free), false);
  assert.equal(automationsBlockedByPi(noPi, [{ action: "start", enabled: false }, { action: "start", enabled: true }], workspaces, free), true);
});

test("no automations, or no catalog yet, never blocks", () => {
  assert.equal(automationsBlockedByPi(noPi, [], workspaces, free), false);
  assert.equal(automationsBlockedByPi([], [{ action: "start" }], workspaces, free), false);
});

import { START_CLIS, automationNeedsPi, startRunHint } from "./automationsPi.js";

test("a start run on another CLI needs no pi (ADR-0217)", () => {
  const none = new Map();
  assert.equal(automationNeedsPi({ action: "start" }, none), true);
  assert.equal(automationNeedsPi({ action: "start", cli: "pi" }, none), true);
  assert.equal(automationNeedsPi({ action: "start", cli: "claude-code" }, none), false);
  assert.deepEqual(START_CLIS, ["pi", "claude-code", "codex", "grok", "hermes", "opencode", "omp"]);
  assert.match(startRunHint("pi"), /Pi agent/);
  assert.match(startRunHint("codex", "Codex"), /fresh Codex conversation.*waits for you/);
});

import { costMeasured } from "./automationsPi.js";

test("every start CLI is priced; the hint names the turns a cost limit cannot see", () => {
  for (const cli of ["omp", "grok", "hermes", "opencode"]) assert.equal(costMeasured(cli), true, cli);
  assert.equal(costMeasured("antigravity"), false);
  assert.match(startRunHint("opencode", "OpenCode"), /no price for does not count toward a cost limit/);
  assert.doesNotMatch(startRunHint("opencode", "OpenCode"), /cannot read/);
  assert.match(startRunHint("antigravity", "Antigravity"), /cannot read Antigravity's cost yet/);
});
