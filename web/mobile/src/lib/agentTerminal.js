// Agent CLIs owns the terminal for every non-Pi agent. Pi's interactive TUI
// keeps its existing agent session until the managed runtime migration lands.
export function resolveAgentCliTerminal(agent, terminal) {
  if (!agent || agent.cli === "pi" || !agent.cli || !agent.terminalId) return null;
  if (!terminal || terminal.id !== agent.terminalId) return null;
  return terminal;
}
