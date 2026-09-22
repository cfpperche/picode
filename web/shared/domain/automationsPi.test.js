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

test("no automations, or no catalog yet, never blocks", () => {
  assert.equal(automationsBlockedByPi(noPi, [], workspaces, free), false);
  assert.equal(automationsBlockedByPi([], [{ action: "start" }], workspaces, free), false);
});
