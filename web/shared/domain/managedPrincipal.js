// Launchable, installed catalog rows a workspace can bind as a managed
// CLI principal (ADR-0159 Fatia 3). Uninstalled and detect-only CLIs stay
// off the picker; Agent CLIs is the way to install them.

export function catalogForPrincipal(clis) {
  return (clis || []).filter((c) => c && c.id && c.launchable !== false && c.installed);
}

// Workspace New → Agent (ADR-0160): Pi is always first (managed + TUI),
// then every installed launchable CLI. Uninstalled guests stay off the
// list; the Agent CLIs hub is how you install them.
export function catalogForAgent(clis) {
  const rest = catalogForPrincipal(clis).filter((c) => c.id !== "pi");
  const raw = (clis || []).find((c) => c && c.id === "pi");
  const pi = { id: "pi", name: (raw && raw.name) || "Pi", installed: true, launchable: true };
  return [pi, ...rest];
}

export function agentIsPi(agent) {
  const c = String((agent && agent.cli) || "pi").trim().toLowerCase();
  return !c || c === "pi";
}
