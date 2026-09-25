import { agentsOf } from "@picode/shared/domain/tree.js";
import { resolveInteractiveTerminal } from "./agentTerminalView.js";
import { agentTerminalKeys } from "@picode/shared/domain/agentTerminal.js";

export function fleetAgents(fleet) {
  return [...(fleet.freeAgents || []), ...(fleet.workspaces || []).flatMap(agentsOf)];
}

export function canvasTerminalHost(kind, target, cwd, fleet, tabs) {
  const id = kind === "agent" ? resolveInteractiveTerminal(target, fleet.terminals)?.id : target?.id;
  const owner = fleetAgents(fleet).find((a) => a.terminalId === id);
  const owned = !!id && (tabs.includes("t:" + id) || (kind === "agent" && tabs.includes(target.id)) || !!(owner && tabs.includes(owner.id)));
  return { id, owned, epoch: fleet.termEpochs?.[kind === "agent" ? target?.id : owner?.id] || 0 };
}

// Runtime addresses serve input/search; owner and tab addresses serve UI actions.
export function terminalHost(pane, agents, tabs = []) {
  const agent = agents.find((a) => pane.kind === "agent" ? a.id === pane.id : a.terminalId === pane.id) || null;
  const terminalTab = "t:" + pane.id;
  const tabId = pane.tabId || (agent && (tabs.includes(agent.id) || !tabs.includes(terminalTab)) ? agent.id : terminalTab);
  return { agent, tabId, ownerKind: agent ? "agent" : "term", ownerId: agent?.id || pane.id };
}

export function canvasTerminalTools(model, host) {
  const id = model?.runtimeId;
  return {
    attach: id && host?.termAttach?.id === id ? host.termAttach : null,
    find: !!id && host?.termFind === id,
  };
}

// Both aliases may exist after a failed preparation or a host migration.
export function agentPaneKeys(agent) {
  return agentTerminalKeys(agent).map(key => key.slice(3));
}

export function closingPaneKeys(tabId, agents, tabs) {
  const terminal = tabId.startsWith("t:");
  const id = terminal ? tabId.slice(2) : tabId;
  const agent = agents.find((a) => terminal ? a.terminalId === id : a.id === id);
  if (terminal) return agent && tabs.includes(agent.id) ? [] : [id];
  return agentPaneKeys(agent).filter((key) => key === agent.id || !tabs.includes("t:" + key));
}
