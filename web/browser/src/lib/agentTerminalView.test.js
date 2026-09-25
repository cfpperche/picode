import test from "node:test";
import assert from "node:assert/strict";
import { resolveInteractiveTerminal } from "./agentTerminalView.js";

test("Pi interactive view uses the bound terminal record", () => {
  const term = { id: "term-pi", session: "picode-term-pi", name: "Pi", running: true };
  const resolved = resolveInteractiveTerminal({ id: "agent-pi", terminalId: "term-pi", name: "Pi" }, [term], "/tmp");
  assert.deepEqual(resolved, { term, id: "term-pi", cwdKind: "terminal", canonical: true });
});

test("an unbound interactive agent has no terminal view (pre-ADR-0162 address retired)", () => {
  assert.equal(resolveInteractiveTerminal({ id: "agent-pi", mode: "interactive", name: "Pi", workPath: "" }, []), null);
});
