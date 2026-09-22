// Launchable, installed catalog rows a workspace can bind as a managed
// CLI principal (ADR-0159 Fatia 3). Uninstalled and detect-only CLIs stay
// off the picker; Agent CLIs is the way to install them.

import { terminalCliLabel } from "./terminalCli.js";

export function catalogForPrincipal(clis) {
  return (clis || []).filter((c) => c && c.id && c.launchable !== false && c.installed);
}

// New → Agent, in a workspace or free (ADR-0160, ADR-0179): Pi first when
// it is installed (managed + TUI), then every other installed launchable
// CLI. An absent Pi is absent like any other CLI — the picker never offers
// a runtime that cannot launch; the Agent CLIs hub is how you install one.
export function catalogForAgent(clis) {
  const rows = catalogForPrincipal(clis);
  const pi = rows.find((c) => c.id === "pi");
  return pi ? [pi, ...rows.filter((c) => c.id !== "pi")] : rows;
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
