import test from "node:test";
import assert from "node:assert/strict";
import { resolveAgentTerminalView, initialAgentView, terminalOwnerBase, agentTerminalKeys, agentTerminalActionTarget } from "./agentTerminal.js";

const term = { id: "t", session: "picode-sh-t", running: true };
const agent = { id: "a", name: "Agent", mode: "interactive", terminalId: "t" };
for (const cli of ["pi", "claude-code", "codex", "grok", "hermes", "opencode", "muse", "agy", "omp"]) {
  test(cli + " uses exactly the terminal record, regardless of entry route", () => {
    const resolved = resolveAgentTerminalView({ ...agent, cli }, term);
    assert.equal(resolved.term, term);
    assert.deepEqual(resolved.owner, { kind: "term", id: "t" });
    assert.equal(initialAgentView({ ...agent, cli }), "term");
  });
}
test("managed wins over binding and explicit terminal view", () => {
  const managed = { ...agent, mode: "managed" };
  assert.equal(resolveAgentTerminalView(managed, term), null);
  assert.equal(initialAgentView(managed, "terminal"), "chat");
});
test("stopped bound CLI retains terminal recovery UI, explicit chat is respected", () => {
  const stopped = { ...agent, mode: "stopped" };
  assert.equal(resolveAgentTerminalView(stopped, { ...term, running: false }).term.running, false);
  assert.equal(initialAgentView(stopped), "term");
  assert.equal(initialAgentView(agent, "chat"), "chat");
  assert.equal(initialAgentView({ mode: "stopped" }), "chat");
});
test("missing bound record never guesses a legacy session", () => {
  assert.equal(resolveAgentTerminalView(agent, null), null);
  assert.equal(resolveAgentTerminalView(agent, { id: "wrong" }), null);
  assert.equal(resolveAgentTerminalView({ ...agent, terminal: term }, null).term, term);
});
test("an unbound agent never acquires a fabricated session, Pi included", () => {
  // The pre-ADR-0162 `picode-<id>` session was retired 2026-09-25.
  for (const cli of ["pi", "codex"]) {
    assert.equal(resolveAgentTerminalView({ ...agent, cli, terminalId: "", mode: "interactive" }, null), null, cli);
  }
  assert.equal(initialAgentView({ ...agent, cli: "codex" }, "chat"), "term");
});
test("cleanup covers both old legacy and newly bound keys without duplicates", () => {
  assert.deepEqual(agentTerminalKeys(agent), ["sh:a", "sh:t"]);
  assert.deepEqual(agentTerminalKeys({ id: "a", terminalId: "a" }), ["sh:a"]);
  assert.deepEqual(agentTerminalKeys(null), []);
  assert.equal(terminalOwnerBase({ kind: "term", id: "a/b" }), "/api/terminals/a%2Fb");
});

test("Stop still confirms when a running agent's client terminal record is missing", () => {
  assert.equal(agentTerminalActionTarget(agent, undefined).running, true);
  assert.equal(agentTerminalActionTarget({ ...agent, mode: undefined }, undefined).running, true);
  assert.equal(agentTerminalActionTarget({ ...agent, mode: "stopped" }, undefined).running, false);
  assert.equal(agentTerminalActionTarget(agent, term), term);
});
