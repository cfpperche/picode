import { isTermSocketSuspended, suspendTermSocket } from "@picode/shared/client/termSocket.js";
import { suspendedToDispose } from "@picode/shared/domain/matrix.js";
import { terms } from "../../lib/terms.js";
import { closeShellTerm } from "../ShellTerm.jsx";

// Pane ownership (docs/plans/matrix-app.md §4.5/§4.6): what unloading a
// terminal body does to its attach.
//
//   the terminal's tab is open   → the tab owns the attach; unloading only
//                                  parks the pane (ShellTerm's unmount)
//   it lives only on matrices    → suspendTermSocket: the socket closes,
//                                  the xterm and its scrollback stay; the
//                                  next load mounts TermSurface and its
//                                  ShellTerm kicks the socket itself
//   > 24 suspended, or 10 min    → LRU disposal (closeShellTerm); a later
//                                  load builds a fresh xterm — tmux still
//                                  holds the screen
//
// The registry is module-level on purpose: one desktop, one `terms` map,
// one list of what the Matrix parked.

const suspendedAt = new Map(); // terminal or agent id → when it was parked
let sweep = 0;
const SWEEP_MS = 30000;

// ownedByTab(kind, ref, openTabs): a terminal's tab is "t:<id>", an agent's
// tab is the agent id (the TUI shares the agent's tab).
export function ownedByTab(kind, ref, openTabs) {
  const tabs = Array.isArray(openTabs) ? openTabs : [];
  return kind === "terminal" ? tabs.includes("t:" + ref) : tabs.includes(ref);
}

// claimPane(ref): a body mounted — whatever was parked is live again.
export function claimPane(ref) {
  suspendedAt.delete(ref);
}

// releasePane(ref, owned): a body unmounted. An attach the tab holds is
// left alone; one only the Matrix held is suspended and remembered.
export function releasePane(ref, owned) {
  const entry = terms.get("sh:" + ref);
  if (!entry || owned) return;
  suspendTermSocket(entry);
  suspendedAt.set(ref, Date.now());
  sweepSuspended();
  if (!sweep && suspendedAt.size) sweep = setInterval(() => sweepSuspended(), SWEEP_MS);
}

// sweepSuspended(now): apply the LRU rule. An entry somebody kicked since
// (its tab reopened, another matrix showed it) is no longer suspended and
// is left alone; one already closed by its tab is just forgotten.
export function sweepSuspended(now = Date.now()) {
  const list = [...suspendedAt].map(([id, at]) => ({ id, at }));
  for (const id of suspendedToDispose(list, now)) {
    const entry = terms.get("sh:" + id);
    if (entry && isTermSocketSuspended(entry)) closeShellTerm(id);
    suspendedAt.delete(id);
  }
  if (!suspendedAt.size && sweep) {
    clearInterval(sweep);
    sweep = 0;
  }
}

// QA hook (same pattern as window.__picodeTerms): how many panes the
// Matrix is holding suspended right now.
if (typeof window !== "undefined") window.__picodeMatrixSuspended = () => [...suspendedAt.keys()];
