import { describe, it } from "node:test";
import assert from "node:assert/strict";
import { resolveAgentCliTerminal } from "./agentTerminal.js";

describe("resolveAgentCliTerminal", () => {
  it("uses the bound Agent CLI terminal for guest agents", () => {
    const terminal = { id: "term-1", launchCli: "claude" };
    assert.deepEqual(resolveAgentCliTerminal({ cli: "claude", terminalId: "term-1" }, terminal), terminal);
  });

  it("keeps Pi on its existing interactive session", () => {
    assert.equal(resolveAgentCliTerminal({ cli: "pi", terminalId: "term-1" }, { id: "term-1" }), null);
  });

  it("fails closed when the binding is missing or mismatched", () => {
    assert.equal(resolveAgentCliTerminal({ cli: "codex", terminalId: "term-1" }, null), null);
    assert.equal(resolveAgentCliTerminal({ cli: "codex", terminalId: "term-1" }, { id: "term-2" }), null);
  });
});
