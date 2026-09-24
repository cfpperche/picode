// Agent status vocabulary (ADR-0062): the sidebar rows, the tab strip and
// the Canvas panel chips all say the same five words for an agent, derived
// the same way. `live` carries what the fleet row does not: the id the
// desktop is streaming for, the ids tmux reports as working (agent.tui),
// the id whose dialog is on screen.
//   agentRowStatus(agent, live) -> "needs-you" | "working" | "interactive" | "stopped" | "ready"
//   agentStatusLabel(status)    -> the chip's copy
import { terminalActivityStamp, terminalStatus } from "./terminalCli.js";

export function agentRowStatus(ag, live = {}) {
  const mode = (ag && ag.mode) || "stopped";
  const term = live.term || (ag && ag.terminal);
  if (ag?.terminalId && !ag.legacyInteractive && mode !== "managed" && term) return terminalStatus(term);
  const waiting = !!(ag && (ag.waiting || (live.waitingId && ag.id === live.waitingId)));
  const working = !waiting && !!(ag && (ag.streaming || (live.workingId && ag.id === live.workingId) || (live.workingIds || []).includes(ag.id)));
  if (waiting) return "needs-you";
  if (working) return "working";
  if (mode === "interactive") return "interactive";
  if (mode === "stopped") return "stopped";
  return "ready";
}

// The pill's age: when the agent entered the state it is in now, resolved
// per status (the sidebar's state-age plan). Terminal-backed rows read the
// hook's stateAt — TermStates.SetForRun rewrites it only on a real
// transition, so it is the transition instant, and the TUI's start covers
// "no report yet" (a terminal has been open since its TUI started).
// Managed rows read the store's own writes: the settle stamp for ready,
// the stop write for stopped, the runtime start for working (the turn
// start itself is not stamped — approximate by design). A stopped row
// never reads a terminal stamp, or the state that preceded the stop would
// show. Never createdAt — "created" is not a state age — and "" when
// nothing truthful exists, so the pill renders the label alone (ADR-0092).
export function agentStatusStamp(status, ag, term) {
  if (status === "stopped") return (ag && ag.lastStatusAt) || "";
  if (status === "interactive") return "";
  if (status === "needs-you" && !term) return ""; // the ask's timestamp is not on the fleet row yet
  if (term) return terminalActivityStamp(term);
  if (status === "working") return (ag && (ag.lastStartedAt || ag.lastStatusAt)) || "";
  return (ag && ag.lastStatusAt) || ""; // ready, managed
}

export function agentStatusLabel(status) {
  if (status === "needs-you") return "Needs you";
  if (status === "working") return "Working";
  if (status === "interactive") return "In terminal";
  if (status === "open") return "Open";
  if (status === "stopped") return "Stopped";
  return "Ready";
}

// The terminal a CLI agent's pill and lifecycle read (ADR-0160): its bound
// terminal from the live list, or the fleet's copy on the agent row. Shared
// so the sidebar's "Working first" buckets and the row's own pill cannot
// disagree about which terminal a row belongs to.
export function agentTerm(ag, terms) {
  if (!ag || ag.legacyInteractive || ag.mode === "managed" || !ag.terminalId) return null;
  return (terms || []).find((t) => t && t.id === ag.terminalId) || ag.terminal || null;
}

// The sidebar's "Working first" view (ADR-0173 amendment, 2026-09-23): one
// presentation-only pass over a container's stored positions. Buckets rank
// by who needs the reader first — the dashboard's fleet ranking, carried to
// the row list — and inside a bucket the stored position rules, so rows move
// only when their state changes, never when a timestamp ticks. Empty buckets
// are absent, never zero-count labels, and a status outside the vocabulary
// still renders its rows at the end: the view may reorder, never lose.
export const STATE_BUCKET_ORDER = ["needs-you", "working", "interactive", "open", "ready", "stopped"];

export function bucketAgentsByState(agents, statusOf) {
  const by = new Map();
  for (const a of agents || []) {
    const s = statusOf(a);
    if (!by.has(s)) by.set(s, []);
    by.get(s).push(a);
  }
  const known = STATE_BUCKET_ORDER.filter((s) => by.has(s));
  const rest = [...by.keys()].filter((s) => !STATE_BUCKET_ORDER.includes(s));
  return [...known, ...rest].map((s) => ({ status: s, agents: by.get(s) }));
}
