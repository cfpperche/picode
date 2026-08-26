export function agentsOf(ws) {
  if (!ws) return [];
  if (ws.agents && ws.agents.length) return ws.agents;
  return ws.agent ? [ws.agent] : [];
}

export function locate(workspaces, freeAgents, agentId) {
  if (!agentId) return null;
  for (const ws of workspaces || []) {
    for (const a of agentsOf(ws)) {
      if (a.id === agentId) return { workspace: ws, agent: a };
    }
    if (ws.id === agentId) {
      const a = agentsOf(ws)[0];
      if (a) return { workspace: ws, agent: a };
    }
  }
  for (const a of freeAgents || []) {
    if (a.id === agentId) return { workspace: null, agent: a };
  }
  return null;
}

export function firstAgentId(workspaces, freeAgents) {
  for (const ws of workspaces || []) {
    const a = agentsOf(ws)[0];
    if (a) return a.id;
  }
  const f = (freeAgents || [])[0];
  return f ? f.id : null;
}

export function displayAgentName(agent, workspace) {
  if (!agent) return "";
  if (agent.name && agent.name !== "default") return agent.name;
  return (workspace && workspace.name) || agent.name || "Agent";
}

export function paneContext(agentName, workspaceName) {
  const a = String(agentName || "").trim();
  const w = String(workspaceName || "").trim();
  if (a && w && a !== w) return a + " · " + w;
  return a || w;
}
