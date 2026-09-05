export function agentsOf(ws) {
  if (!ws) return [];
  const list = (ws.agents && ws.agents.length) ? ws.agents : (ws.agent && ws.agent.id ? [ws.agent] : []);
  return list.filter((a) => a && a.id);
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

export function mentionAgents(workspaces, freeAgents, currentId) {
  const out = [];
  const seen = new Set();
  function add(agent, workspace) {
    if (!agent || agent.id === currentId || seen.has(agent.id)) return;
    seen.add(agent.id);
    out.push({ id: agent.id, name: displayAgentName(agent, workspace) });
  }
  for (const ws of workspaces || []) {
    for (const a of agentsOf(ws)) add(a, ws);
  }
  for (const a of freeAgents || []) add(a, null);
  return out;
}

export function paneContext(agentName, workspaceName) {
  const a = String(agentName || "").trim();
  const w = String(workspaceName || "").trim();
  if (a && w && a !== w) return a + " · " + w;
  return a || w;
}
