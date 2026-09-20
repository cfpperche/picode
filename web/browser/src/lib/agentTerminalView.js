// Resolve the terminal identity used by the desktop Pi view. Newly allocated
// interactive sessions have a bound terminal; old live sessions may still be
// addressed by the agent until an explicit restart (ADR-0160).
export function resolveInteractiveTerminal(agent, terminals, fallbackCwd) {
  const bound = agent && agent.terminalId
    ? (terminals || []).find((term) => term && term.id === agent.terminalId)
    : null;
  if (bound) return { term: bound, id: bound.id, cwdKind: "terminal", canonical: true };
  if (!agent) return null;
  return {
    term: {
      id: agent.id,
      session: "picode-" + agent.id,
      name: agent.name + " · TUI",
      cwd: agent.workPath || fallbackCwd,
    },
    id: agent.id,
    cwdKind: "agent",
    canonical: false,
  };
}
