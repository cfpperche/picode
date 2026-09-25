// Resolve the one terminal record an interactive agent owns. Both web shells
// use this identity before choosing their presentation; no CLI is special.
export function resolveAgentTerminal(agent, terminal) {
  if (!agent || agent.mode === "managed" || !agent.terminalId) return null;
  if (!terminal || terminal.id !== agent.terminalId) return null;
  return terminal;
}

// One addressing adapter for every TUI surface: the agent's bound terminal.
// A missing bound record is not another process — never guess a session
// while the feed loads. (The pre-ADR-0162 `picode-<id>` session this once
// rendered was retired 2026-09-25.)
export function resolveAgentTerminalView(agent, terminal) {
  if (!agent || agent.mode === "managed") return null;
  const bound = resolveAgentTerminal(agent, terminal) || resolveAgentTerminal(agent, agent.terminal);
  return bound ? { term: bound, owner: { kind: "term", id: bound.id }, canonical: true } : null;
}

export function initialAgentView(agent, requested = "") {
  if (!agent || agent.mode === "managed") return "chat";
  if (requested === "chat" && (!agent.cli || agent.cli === "pi")) return "chat";
  return agent.mode === "interactive" || agent.terminalId ? "term" : "chat";
}

export function terminalOwnerBase(owner) {
  return (owner.kind === "agent" ? "/api/agents/" : "/api/terminals/") + encodeURIComponent(owner.id);
}

export function agentTerminalKeys(agent) {
  return [...new Set([agent?.id, agent?.terminalId].filter(Boolean))].map(id => "sh:" + id);
}

// Lifecycle fallback is not an attach descriptor. Unknown client state must
// still ask before stopping a process which may be alive on the server.
export function agentTerminalActionTarget(agent, terminal) {
  return terminal?.id === agent.terminalId ? terminal
    : { id: agent.terminalId, name: agent.name, running: agent.mode !== "stopped" };
}
