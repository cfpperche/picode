import test from "node:test";
import assert from "node:assert/strict";
import { resolveInteractiveTerminal } from "./agentTerminalView.js";

test("Pi interactive view uses the bound terminal record", () => {
  const term = { id: "term-pi", session: "picode-term-pi", name: "Pi", running: true };
  const resolved = resolveInteractiveTerminal({ id: "agent-pi", terminalId: "term-pi", name: "Pi" }, [term], "/tmp");
  assert.deepEqual(resolved, { term, id: "term-pi", cwdKind: "terminal", canonical: true });
});

test("legacy Pi interactive view keeps the agent address until restart", () => {
  const resolved = resolveInteractiveTerminal({ id: "agent-pi", name: "Pi", workPath: "" }, [], "/tmp");
  assert.equal(resolved.id, "agent-pi");
  assert.equal(resolved.cwdKind, "agent");
  assert.equal(resolved.canonical, false);
  assert.equal(resolved.term.session, "picode-agent-pi");
  assert.equal(resolved.term.cwd, "/tmp");
});
