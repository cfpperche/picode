// Which automations still need Pi on this machine (ADR-0179). A `start`
// run creates a Pi agent; a `message` run needs Pi only when its target is
// a Pi agent — a guest agent is reached through its launch terminal. An
// unknown target counts as Pi, which is what a legacy row without `cli` is.

export function piInstalled(clis) {
  const pi = (clis || []).find((c) => c && c.id === "pi");
  return pi ? !!pi.installed : null; // null: the catalog has not answered yet
}

export function agentCliIndex(workspaces, freeAgents) {
  const out = new Map();
  for (const ws of workspaces || []) for (const ag of (ws && ws.agents) || []) if (ag && ag.id) out.set(ag.id, ag.cli || "pi");
  for (const ag of freeAgents || []) if (ag && ag.id) out.set(ag.id, ag.cli || "pi");
  return out;
}

export function automationNeedsPi(a, cliOf) {
  if (!a || a.enabled === false) return false; // a disabled one never runs
  if (a.action !== "message") return true;
  return (cliOf.get(a.targetAgentId) || "pi") === "pi";
}

// automationsBlockedByPi: true when Pi is known to be absent and at least
// one automation would need it. No automations, or no catalog yet → false.
export function automationsBlockedByPi(clis, items, workspaces, freeAgents) {
  if (piInstalled(clis) !== false) return false;
  const cliOf = agentCliIndex(workspaces, freeAgents);
  return (items || []).some((a) => automationNeedsPi(a, cliOf));
}
