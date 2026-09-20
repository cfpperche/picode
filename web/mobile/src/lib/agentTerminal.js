import { resolveAgentTerminal as resolveSharedAgentTerminal } from "@picode/shared/domain/agentTerminal.js";

export function resolveAgentTerminal(agent, terminal) {
  return resolveSharedAgentTerminal(agent, terminal);
}
