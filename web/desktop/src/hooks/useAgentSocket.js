import { useCallback, useEffect, useRef, useState } from "react";
import { api, wsURL } from "@picode/shared/client/api.js";
import { reduceAgentEvent, initialAgentState } from "../lib/agentEvents.js";
import { eventsToItems } from "@picode/shared/domain/replay.js";
import { reconcileTranscript, liveSince, transcriptGate } from "@picode/shared/domain/transcriptMerge.js";

// useAgentSocket — one `/ws/agent?agent=<id>` per mount, for a **read-only**
// conversation (docs/plans/matrix-app.md §2.4, phase 4). It is the desktop
// port of `web/mobile/src/hooks/useAgentSocket.js`: the same shape, the same
// pure reducer, the same transcript reconciliation — with the composer verbs
// left out, because a Matrix panel reads and the agent's tab answers.
//
// **What it does not do, and why.** No `send`, `abort`, `replyAsk`,
// `runBash` or `abortBash`: phase 4 ships a reader, and the one action a
// panel offers is Open (plan §9, decision 3 — the quick reply line is
// phase 5). When phase 5 wants them, they are the mobile hook's verbs
// verbatim, and this file is where they land. The reducer's two *writing*
// effects are dropped for the same reason: `replyUI` (the auto-answer that
// walks an ask backwards) would make a reader answer, and `toast` would let
// N panels shout one agent's error — the error is already an `alert` item in
// the conversation this panel renders, and the selected agent's tab
// (App.jsx) is what toasts. Only `scroll` is executed.
//
// **Two sockets for one agent are fine, and are the point.** The desktop
// holds one App-level socket for the *selected* agent; a panel opens its
// own. The server subscribes per connection (`internal/rpc/runtime.go`
// `Hub.Subscribe` hands every subscriber its own buffered channel, and
// `Broadcast` writes to all of them), so neither drops or double-counts the
// other's events. The two transcript reads are independent GETs of the same
// file. One consequence is worth knowing: `Hub.Len()` is the server's
// "is anybody watching" gate (ADR-0037's unobserved-run rule), so a live
// chat panel counts as a watcher exactly as an open tab does — see
// docs/architecture/matrix.md.
//
// The socket's lifetime IS this hook's mount: chunk loading and the zoom
// decide whether the body renders (`loadPolicy`'s `bodies`), and the body
// not rendering is the socket not existing. No component holds an `if`
// about it.

// QA hook, the pattern of window.__picodeTerms: how many agent sockets this
// page is holding, and for whom. The cap in `loadPolicy` is measured with it.
const open = new Map(); // agentId → how many mounts hold a socket for it
if (typeof window !== "undefined") {
  window.__picodeAgentSockets = {
    count: () => [...open.values()].reduce((n, c) => n + c, 0),
    agents: () => Object.fromEntries(open),
  };
}
function track(id, delta) {
  const n = (open.get(id) || 0) + delta;
  if (n > 0) open.set(id, n);
  else open.delete(id);
}

const RETRY_MS = 1500;
const TAIL = 200;

export function useAgentSocket(agentId, workspaceId = "ws_free") {
  const [state, setState] = useState(initialAgentState);
  const [scrollTick, setScrollTick] = useState(0);
  // `ready` is "the transcript window has answered once": until it has, an
  // empty conversation means "not read yet", never "nothing was said" — the
  // same rule the surface follows for the fleet.
  const [ready, setReady] = useState(false);
  const stateRef = useRef(initialAgentState);
  const sockRef = useRef(null);
  const historyGate = useRef(transcriptGate());
  const historyPath = useRef("");
  const retryRef = useRef(null);

  const set = useCallback((next) => {
    stateRef.current = next;
    setState(next);
  }, []);

  useEffect(() => {
    if (!agentId) return undefined;
    let stopped = false;
    historyGate.current = transcriptGate();
    historyPath.current = "";
    setReady(false);
    set(initialAgentState);

    function apply(ev) {
      const r = reduceAgentEvent(stateRef.current, ev);
      set(r.state);
      for (const fx of r.effects) if (fx && fx.type === "scroll") setScrollTick((t) => t + 1);
    }

    async function loadHistory(panel) {
      const ticket = historyGate.current.begin();
      const baseline = stateRef.current.items;
      try {
        const t = await api("/api/workspaces/" + encodeURIComponent(workspaceId) +
          "/sessions/transcript?agent=" + encodeURIComponent(agentId) + "&tail=" + TAIL);
        if (stopped || !historyGate.current.current(ticket) || sockRef.current !== panel || panel.stopped) return;
        // A session that changed under us (a fork, a resume) invalidates
        // what the live stream appended before it: only a same-session
        // window may keep the deltas that arrived while it was in flight.
        const sameSession = !historyPath.current || historyPath.current === t.path;
        historyPath.current = t.path || "";
        const cur = stateRef.current;
        const live = liveSince(cur.items, baseline, sameSession);
        set({ ...cur, items: reconcileTranscript(eventsToItems(t.events || []), live) });
        setReady(true);
      } catch {
        // Keep the live state: a reconnect or the next settle retries it.
        if (!stopped) setReady(true);
      }
    }

    function close() {
      clearTimeout(retryRef.current);
      historyGate.current.invalidate();
      const p = sockRef.current;
      if (!p) return;
      p.stopped = true;
      try { p.sock.close(); } catch { /* already gone */ }
      if (p.counted) { track(agentId, -1); p.counted = false; }
      sockRef.current = null;
    }

    function connect() {
      close();
      if (stopped) return;
      const sock = new WebSocket(wsURL("/ws/agent?agent=" + encodeURIComponent(agentId)));
      const panel = { sock, stopped: false, counted: true };
      track(agentId, 1);
      sockRef.current = panel;
      sock.onmessage = (ev) => {
        if (sockRef.current !== panel || panel.stopped) return;
        try {
          const event = JSON.parse(ev.data).event || {};
          apply(event);
          // The snapshot is the join point and a settle is the end of a
          // turn: both are when the session file is the truth, so the
          // window is refetched and reconciled against what streamed.
          if (event.type === "snapshot" || event.type === "agent_settled") loadHistory(panel);
        } catch { /* a frame we do not understand is not a reason to drop the rest */ }
      };
      sock.onclose = () => {
        if (panel.counted) { track(agentId, -1); panel.counted = false; }
        if (sockRef.current !== panel || panel.stopped) return;
        set({ ...stateRef.current, streaming: false, waiting: false, status: "disconnected" });
        retryRef.current = setTimeout(() => { if (sockRef.current === panel && !panel.stopped) connect(); }, RETRY_MS);
        if (window.__picodeKickHealth) window.__picodeKickHealth();
      };
    }

    connect();
    return () => { stopped = true; close(); };
  }, [agentId, workspaceId, set]);

  // Expanding a tool card and a file list is this reader's own view state,
  // not a verb: it changes nothing on the agent.
  const toggleTool = useCallback((id) => {
    set({ ...stateRef.current, items: stateRef.current.items.map((it) => (it.kind === "tool" && it.id === id ? { ...it, expanded: !it.expanded } : it)) });
  }, [set]);
  const toggleFiles = useCallback((index) => {
    set({ ...stateRef.current, items: stateRef.current.items.map((it, i) => (i === index && it.kind === "files" ? { ...it, expanded: !it.expanded } : it)) });
  }, [set]);

  return { state, scrollTick, ready, toggleTool, toggleFiles };
}

export default useAgentSocket;
