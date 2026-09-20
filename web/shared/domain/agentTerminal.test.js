import test from "node:test";
import assert from "node:assert/strict";
import { resolveAgentTerminal } from "./agentTerminal.js";

test("all interactive agents resolve through their bound terminal", () => {
  const term = { id: "term-1", launchCli: "pi" };
  assert.deepEqual(resolveAgentTerminal({ terminalId: "term-1" }, term), term);
  assert.deepEqual(resolveAgentTerminal({ mode: "interactive", cli: "claude", terminalId: "term-1" }, term), term);
});

test("managed and mismatched agents fail closed", () => {
  const term = { id: "term-1" };
  assert.equal(resolveAgentTerminal({ mode: "managed", terminalId: "term-1" }, term), null);
  assert.equal(resolveAgentTerminal({ mode: "interactive", terminalId: "term-1" }, { id: "term-2" }), null);
});
