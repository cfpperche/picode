import { describe, it } from "node:test";
import assert from "node:assert/strict";
import { resolveAgentTerminal } from "./agentTerminal.js";

describe("resolveAgentCliTerminal", () => {
  it("uses the bound terminal for every interactive agent", () => {
    const terminal = { id: "term-1", launchCli: "pi" };
    assert.deepEqual(resolveAgentTerminal({ cli: "pi", terminalId: "term-1" }, terminal), terminal);
    assert.deepEqual(resolveAgentTerminal({ mode: "interactive", cli: "claude", terminalId: "term-1" }, terminal), terminal);
  });

  it("does not resolve a managed Pi runtime as a terminal", () => {
    assert.equal(resolveAgentTerminal({ mode: "managed", cli: "pi", terminalId: "term-1" }, { id: "term-1" }), null);
  });

  it("fails closed when the binding is missing or mismatched", () => {
    assert.equal(resolveAgentTerminal({ cli: "codex", terminalId: "term-1" }, null), null);
    assert.equal(resolveAgentTerminal({ cli: "codex", terminalId: "term-1" }, { id: "term-2" }), null);
  });
});
