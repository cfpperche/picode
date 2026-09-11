import { isTermSocketSuspended, suspendTermSocket } from "@picode/shared/client/termSocket.js";
import { suspendedToDispose } from "@picode/shared/domain/canvas.js";
import { terms } from "../../lib/terms.js";
import { closeShellTerm } from "../ShellTerm.jsx";

// Pane ownership (docs/plans/matrix-app.md §4.5/§4.6): what unloading a
// terminal body does to its attach.
//
//   the terminal's tab is open   → the tab owns the attach; unloading only
//                                  parks the pane (ShellTerm's unmount)
//   it lives only on canvases    → suspendTermSocket: the socket closes,
//                                  the xterm and its scrollback stay; the
//                                  next load mounts TermSurface and its
//                                  ShellTerm kicks the socket itself
//   > 24 suspended, or 10 min    → LRU disposal (closeShellTerm); a later
//                                  load builds a fresh xterm — tmux still
//                                  holds the screen
//   the terminal or agent is     → forgetPane: a suspended pane is disposed
//   deleted (feed)                  now, a mounted one when its body
//                                  unmounts — never the LRU's wait
//
// Which host shows a pane is counted, not guessed: `mounted` holds how many
// bodies are up for that id, so a hand-off between hosts parks nothing
// whichever way round the two commits fall. The grid unmounts the old body
// before it mounts the new one (one commit, release then claim); the canvas
// mounts the maximize layer's body a commit *before* its React Flow node
// drops the wrapper's, so a release with no count would have suspended the
// socket of the pane the viewer is looking at. The one-tick wait stays for
// the first order — a release that a claim cancels in the same task.
//
// The registry is module-level on purpose: one desktop, one `terms` map,
// one list of what the Canvas parked.

const suspendedAt = new Map(); // terminal or agent id → when it was parked
const pendingRelease = new Map(); // id → the timer of a release still waiting its tick
const mounted = new Map(); // id → how many bodies are showing that pane right now
const gone = new Set(); // ids whose target was deleted while their pane was mounted
let sweep = 0;
const SWEEP_MS = 30000;

// ownedByTab(kind, ref, openTabs): a terminal's tab is "t:<id>", an agent's
// tab is the agent id (the TUI shares the agent's tab).
export function ownedByTab(kind, ref, openTabs) {
  const tabs = Array.isArray(openTabs) ? openTabs : [];
  return kind === "terminal" ? tabs.includes("t:" + ref) : tabs.includes(ref);
}

// claimPane(ref): a body mounted — whatever was parked, or about to be, is
// live again.
export function claimPane(ref) {
  mounted.set(ref, (mounted.get(ref) || 0) + 1);
  const t = pendingRelease.get(ref);
  if (t) {
    clearTimeout(t);
    pendingRelease.delete(ref);
  }
  suspendedAt.delete(ref);
}

// releasePane(ref, owned): a body unmounted. Another host still showing the
// pane keeps it: only the last body out parks it, and even then on the next
// tick, so a claim arriving in the same task cancels it.
export function releasePane(ref, owned) {
  const left = (mounted.get(ref) || 1) - 1;
  if (left > 0) {
    mounted.set(ref, left);
    return;
  }
  mounted.delete(ref);
  const t = pendingRelease.get(ref);
  if (t) clearTimeout(t);
  pendingRelease.set(ref, setTimeout(() => {
    pendingRelease.delete(ref);
    if (mounted.has(ref)) return; // a body came back before the tick
    park(ref, owned);
  }, 0));
}

// park(ref, owned): an attach the tab holds is left alone; one only the
// Canvas held is suspended and remembered — or disposed, when its target
// went away while the body was up.
function park(ref, owned) {
  const entry = terms.get("sh:" + ref);
  const dead = gone.delete(ref);
  if (!entry || owned) return;
  if (dead) {
    closeShellTerm(ref);
    return;
  }
  suspendTermSocket(entry);
  suspendedAt.set(ref, Date.now());
  sweepSuspended();
  if (!sweep && suspendedAt.size) sweep = setInterval(() => sweepSuspended(), SWEEP_MS);
}

// forgetPane(ref, owned): the feed said the terminal or agent is gone. A
// pane the Canvas holds suspended is disposed at once; one still mounted
// is disposed when its body unmounts (the gone row replaces it); one a tab
// owns is the tab's to close.
export function forgetPane(ref, owned) {
  const entry = terms.get("sh:" + ref);
  if (!entry || owned) {
    gone.delete(ref);
    return;
  }
  if (isTermSocketSuspended(entry)) {
    closeShellTerm(ref);
    suspendedAt.delete(ref);
    gone.delete(ref);
    return;
  }
  gone.add(ref);
}

// sweepSuspended(now): apply the LRU rule. An entry somebody kicked since
// (its tab reopened, another canvas showed it) is no longer suspended and
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
// Canvas is holding suspended right now.
if (typeof window !== "undefined") window.__picodeCanvasSuspended = () => [...suspendedAt.keys()];
