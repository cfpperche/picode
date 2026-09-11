// Agent status vocabulary (ADR-0062): the sidebar rows, the tab strip and
// the Canvas panel chips all say the same five words for an agent, derived
// the same way. `live` carries what the fleet row does not: the id the
// desktop is streaming for, the ids tmux reports as working (agent.tui),
// the id whose dialog is on screen.
//   agentRowStatus(agent, live) -> "needs-you" | "working" | "interactive" | "stopped" | "ready"
//   agentStatusLabel(status)    -> the chip's copy
export function agentRowStatus(ag, live = {}) {
  const mode = (ag && ag.mode) || "stopped";
  const waiting = !!(ag && (ag.waiting || (live.waitingId && ag.id === live.waitingId)));
  const working = !waiting && !!(ag && (ag.streaming || (live.workingId && ag.id === live.workingId) || (live.workingIds || []).includes(ag.id)));
  if (waiting) return "needs-you";
  if (working) return "working";
  if (mode === "interactive") return "interactive";
  if (mode === "stopped") return "stopped";
  return "ready";
}

export function agentStatusLabel(status) {
  if (status === "needs-you") return "Needs you";
  if (status === "working") return "Working";
  if (status === "interactive") return "In terminal";
  if (status === "stopped") return "Stopped";
  return "Ready";
}
