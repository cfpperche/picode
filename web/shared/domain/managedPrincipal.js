// Launchable, installed catalog rows a workspace can bind as a managed
// CLI principal (ADR-0159 Fatia 3). Uninstalled and detect-only CLIs stay
// off the picker; Agent CLIs is the way to install them.

import { terminalCliLabel } from "./terminalCli.js";

export function catalogForPrincipal(clis) {
  return (clis || []).filter((c) => c && c.id && c.launchable !== false && c.installed);
}

// Workspace New → Agent (ADR-0160): Pi is always first (managed + TUI),
// then every installed launchable CLI. Uninstalled CLIs stay off the
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

// The row's fallback subtitle once the model is out of the way: a CLI agent
// names the CLI it runs (ADR-0160 — the row IS the agent, the CLI is its
// face), Pi keeps the interactive/managed wording.
export function agentSubtitle(agent) {
  if (!agentIsPi(agent)) return terminalCliLabel((agent && agent.cli) || "");
  return ((agent && agent.mode) || "stopped") === "interactive" ? "Interactive session" : "Pi agent";
}

// Contact and participant lists label a peer owner by kind. A CLI agent
// names its CLI (ADR-0160); Pi agents — and anything the caller already
// narrows by kind — keep the "Pi agent" wording.
export function principalLabel(owner) {
  if (owner && owner.kind === "agent" && !agentIsPi(owner)) return terminalCliLabel(owner.cli || "");
  return "Pi agent";
}
