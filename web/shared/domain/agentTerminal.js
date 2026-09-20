// Resolve the one terminal record an interactive agent owns. Both web shells
// use this identity before choosing their presentation; no CLI is special.
export function resolveAgentTerminal(agent, terminal) {
  if (!agent || agent.mode === "managed" || !agent.terminalId) return null;
  if (!terminal || terminal.id !== agent.terminalId) return null;
  return terminal;
}
