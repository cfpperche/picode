// Which automations still need Pi on this machine (ADR-0179). A `start`
// run creates a Pi agent; a `message` run needs Pi only when its target is
// a Pi agent — a guest agent is reached through its launch terminal. An
// unknown target counts as Pi.

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
  if (a.action !== "message") return (a.cli || "pi") === "pi"; // ADR-0217: a start run on another CLI needs no pi
  return (cliOf.get(a.targetAgentId) || "pi") === "pi";
}

// automationsBlockedByPi: true when Pi is known to be absent and at least
// one automation would need it. No automations, or no catalog yet → false.
export function automationsBlockedByPi(clis, items, workspaces, freeAgents) {
  if (piInstalled(clis) !== false) return false;
  const cliOf = agentCliIndex(workspaces, freeAgents);
  return (items || []).some((a) => automationNeedsPi(a, cliOf));
}

// The CLIs a start run can use (ADR-0217; store.UnattendedCLIs pins the
// same list): Pi's managed runtime, and the four whose composer PiCode reads
// and whose hooks say when a turn ends.
export const START_CLIS = ["pi", "claude-code", "codex", "grok", "hermes", "opencode", "omp"];

// The CLIs whose session file PiCode prices (internal/climetrics
// MeterSessionFile): a start run on another CLI has no measured cost, so its
// cost limit cannot stop it and its runs show no cost.
export const METERED_CLIS = ["pi", "claude-code", "codex", "omp"];

export function costMeasured(cli) {
  return METERED_CLIS.includes(cli || "pi");
}

// The line under the start fields, by CLI.
export function startRunHint(cli, cliName = "") {
  if (!cli || cli === "pi") return "A fresh Pi agent each run, in that workspace. Empty provider, model or thinking means Pi's own defaults.";
  const name = cliName || cli;
  const cost = costMeasured(cli) ? "" : " PiCode cannot read " + name + "'s cost yet, so a cost limit does not stop its runs.";
  return "A fresh " + name + " conversation each run, in that workspace, with " + name + "'s own settings. PiCode never approves anything for it: a run that asks waits for you, and the Inbox says where." + cost;
}
