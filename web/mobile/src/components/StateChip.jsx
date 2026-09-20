import { agentRowStatus } from "@picode/shared/domain/agentStatus.js";
const LABEL = { working: "Working", waiting: "Needs you", idle: "Idle", stopped: "Stopped", interactive: "In terminal", open: "Open" };

// One word per state; waiting alone takes the accent, since it is the
// only one that is the user's move. `age` rides along so a working chip
// can say "Working · 2m" — the same shape the terminal row shows.
export default function StateChip({ state, age }) {
  const s = LABEL[state] ? state : "stopped";
  return <span className={"m-state is-" + s}>{LABEL[s]}{age || ""}</span>;
}

// agentState folds the three signals the fleet carries into one word.
export function agentState(a, workingIds) {
  if (a?.terminalId && a.mode !== "managed" && a.terminal) {
    const state = agentRowStatus(a, { workingIds });
    return state === "needs-you" ? "waiting" : state === "ready" ? "idle" : state;
  }
  if (!a || !a.mode || a.mode === "stopped") return "stopped";
  if (a.waiting) return "waiting";
  if (a.streaming || (workingIds || []).includes(a.id)) return "working";
  if (a.mode === "interactive") return "interactive";
  return "idle";
}
