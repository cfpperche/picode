import { resolveAgentTerminalView } from "@picode/shared/domain/agentTerminal.js";

export function resolveInteractiveTerminal(agent, terminals) {
  const resolved = resolveAgentTerminalView(agent, (terminals || []).find(term => term?.id === agent?.terminalId));
  if (!resolved) return null;
  return { term: resolved.term, id: resolved.term.id, cwdKind: resolved.owner.kind === "agent" ? "agent" : "terminal", canonical: resolved.canonical };
}
