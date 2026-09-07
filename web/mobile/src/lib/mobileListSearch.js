import { agentsOf } from "@picode/shared/domain/tree.js";
import { workspaceTerminals } from "./termGroups.js";

import { matchesListSearch } from "@picode/shared/domain/listSearch.js";
export { matchesListSearch };

export const agentMatchesSearch = (agent, query) => matchesListSearch(query, agent.name, agent.provider, agent.model, agent.workPath);
export const terminalMatchesSearch = (terminal, query) => matchesListSearch(query, terminal.name, terminal.cli, terminal.launchCli, terminal.cwd);

// A matching folder reveals its whole group. Otherwise retain the folder
// as context for only the matching agents and terminals inside it.
export function searchWorkspaceGroups(workspaces, terminals, query) {
  return (workspaces || []).flatMap(workspace => {
    const context = [workspace.name, workspace.path, workspace.git?.branch];
    const wholeGroup = matchesListSearch(query, ...context);
    const agents = agentsOf(workspace).filter(agent => wholeGroup || matchesListSearch(query, ...context, agent.name, agent.provider, agent.model, agent.workPath));
    const terms = workspaceTerminals(terminals, workspace.id).filter(terminal => wholeGroup || matchesListSearch(query, ...context, terminal.name, terminal.cli, terminal.launchCli, terminal.cwd));
    return wholeGroup || agents.length || terms.length ? [{ workspace, agents, terminals: terms }] : [];
  });
}
