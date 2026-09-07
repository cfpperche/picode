import { useEffect, useRef, useState } from "react";
import { api, wsURL } from "@picode/shared/client/api.js";
import { reduceAgentEvent, initialAgentState, markSent, markUndelivered, markAborted } from "../lib/agentEvents.js";
import { answerAsk, unanswerAsk, backAsk, BACK } from "@picode/shared/domain/askForm.js";
import { bashLine } from "@picode/shared/domain/bashLine.js";
import { toast, toastError } from "../lib/toast.js";
import { eventsToItems } from "@picode/shared/domain/replay.js";
import { reconcileTranscript, liveSince, transcriptGate } from "@picode/shared/domain/transcriptMerge.js";
import { submissionKind } from "../lib/agentDrafts.js";

// One WebSocket for the agent screen only (ADR-0044). Drives the pure
// reducer in lib/agentEvents.js and executes its effects; exposes the
// same verbs the desktop composer has — send, abort, replyAsk — with the
// desktop's busy rules (a prompt while busy becomes a follow-up, which
// goes straight to the server queue).
export function useAgentSocket(agent, workspaceId = "ws_free") {
  const [state, setState] = useState(initialAgentState);
  const [scrollTick, setScrollTick] = useState(0);
  const stateRef = useRef(initialAgentState);
  const sockRef = useRef(null);
  const agentId = agent && agent.id;
  const selectedRef = useRef(agentId);
  selectedRef.current = agentId;
  const historyGate = useRef(transcriptGate());
  const historyPath = useRef("");
  const retryRef = useRef(null);
  const managed = !!agent && agent.mode === "managed";
  const activeRef = useRef(true);

  function set(next) {
    stateRef.current = next;
    setState(next);
  }

  function runEffect(fx) {
    if (!fx) return;
    if (fx.type === "toast") return fx.level === "error" ? toastError(fx.text) : toast.info(fx.text);
    if (fx.type === "scroll") return setScrollTick((t) => t + 1);
    if (fx.type === "replyUI" && agentId) {
      api("/api/agents/" + agentId + "/ui", {
        method: "POST", headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ id: fx.id, value: fx.value }),
      }).catch(() => {});
    }
    return undefined;
  }

  function apply(ev) {
    const r = reduceAgentEvent(stateRef.current, ev);
    set(r.state);
    for (const fx of r.effects) runEffect(fx);
  }

  function close() {
    clearTimeout(retryRef.current);
    historyGate.current.invalidate();
    const p = sockRef.current;
    if (!p) return;
    p.stopped = true;
    try { p.sock.close(); } catch { /* ignore */ }
    sockRef.current = null;
  }

  function connect(id) {
    close();
    const sock = new WebSocket(wsURL("/ws/agent?agent=" + encodeURIComponent(id)));
    const panel = { agentId: id, sock, stopped: false };
    sockRef.current = panel;
    sock.onmessage = (ev) => {
      if (sockRef.current !== panel || panel.stopped || selectedRef.current !== id) return;
      try {
        const event = JSON.parse(ev.data).event || {};
        apply(event);
        if (event.type === "snapshot" || event.type === "agent_settled") loadHistory(panel);
      } catch { /* ignore */ }
    };
    sock.onclose = () => {
      if (sockRef.current === panel && !panel.stopped) {
        set({ ...stateRef.current, streaming: false, waiting: false, status: "disconnected" });
        retryRef.current = setTimeout(() => {
          if (sockRef.current === panel && !panel.stopped && selectedRef.current === id) connect(id);
        }, 1500);
      }
      if (window.__picodeKickHealth) window.__picodeKickHealth();
    };
  }

  async function loadHistory(panel) {
    const ticket = historyGate.current.begin();
    const baseline = stateRef.current.items;
    try {
      const t = await api("/api/workspaces/" + encodeURIComponent(workspaceId) +
        "/sessions/transcript?agent=" + encodeURIComponent(panel.agentId) + "&tail=200");
      if (!historyGate.current.current(ticket) || sockRef.current !== panel || panel.stopped || selectedRef.current !== panel.agentId) return;
      const sameSession = !historyPath.current || historyPath.current === t.path;
      historyPath.current = t.path || "";
      const cur = stateRef.current;
      const live = liveSince(cur.items, baseline, sameSession);
      set({ ...cur, items: reconcileTranscript(eventsToItems(t.events || []), live) });
    } catch { /* Keep live state; reconnect or settle retries the history. */ }
  }

  useEffect(() => {
    activeRef.current = true;
    historyPath.current = "";
    set(initialAgentState);
    if (!agentId || !managed) { close(); return () => { activeRef.current = false; close(); }; }
    connect(agentId);
    return () => { activeRef.current = false; close(); };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [agentId, managed]);

  function ensureConnected() {
    const p = sockRef.current;
    if (!agentId || !activeRef.current || selectedRef.current !== agentId) return;
    if (!p || p.agentId !== agentId || (p.sock && p.sock.readyState !== 1)) connect(agentId);
  }

  async function runBash(command) {
    if (!agentId) return;
    const itemId = "bash-" + Date.now();
    set({ ...stateRef.current, items: [...stateRef.current.items, { kind: "bash", id: itemId, command, output: "", status: "run" }] });
    try {
      try { await api("/api/agents/" + agentId + "/managed/start", { method: "POST" }); } catch { /* already */ }
      ensureConnected();
      const res = await api("/api/agents/" + agentId + "/bash", {
        method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ command }),
      });
      if (activeRef.current && selectedRef.current === agentId) set({ ...stateRef.current, items: stateRef.current.items.map((it) => (it.kind === "bash" && it.id === itemId ? {
        ...it, output: res.output || it.output, exit: res.exitCode,
        status: res.cancelled ? "cancelled" : (res.exitCode === 0 ? "ok" : "err"),
      } : it)) });
      return { accepted: true };
    } catch (e) {
      if (activeRef.current && selectedRef.current === agentId) set({ ...stateRef.current, items: stateRef.current.items.map((it) => (it.kind === "bash" && it.id === itemId ? { ...it, status: "err", output: it.output || (e && e.message) || String(e) } : it)) });
      throw e;
    }
  }

  async function abortBash() {
    if (!agentId) return;
    try { await api("/api/agents/" + agentId + "/bash/abort", { method: "POST" }); } catch { /* nothing running */ }
  }

  async function send(text, images, kind) {
    const payload = String(text || "").trim();
    const pics = images || [];
    if ((!payload && !pics.length) || !agentId) return { accepted: false };
    const cur = stateRef.current;
    const busy = cur.streaming || cur.waiting;
    const sendKind = submissionKind(kind, cur);
    const bash = bashLine(payload);
    if (bash && bash.refused) {
      return { accepted: false, error: "Use the terminal to run a command without sending its output." };
    }
    if (bash && !pics.length) return runBash(bash.command);
    const ts = Date.now();
    set(markSent(cur, { kind: sendKind, text: payload, images: pics.map((p) => p.url), ts, busy }));
    try { await api("/api/agents/" + agentId + "/managed/start", { method: "POST" }); } catch { /* already running */ }
    ensureConnected();
    try {
      if (pics.length || busy || sendKind !== "prompt") {
        await api("/api/agents/" + agentId + "/prompt", {
          method: "POST", headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ kind: sendKind, message: payload, images: pics.map((p) => ({ mimeType: p.mime, data: p.data })) }),
        });
      } else {
        await api("/api/agents/" + agentId + "/tasks", {
          method: "POST", headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ kind: sendKind, payload, source: "user" }),
        });
      }
      return { accepted: true };
    } catch (e) {
      if (activeRef.current && selectedRef.current === agentId) set(markUndelivered(stateRef.current, ts, (e && e.message) || String(e), { preserveActivity: busy }));
      throw e;
    }
  }

  async function abort() {
    if (!agentId) return;
    set(markAborted(stateRef.current));
    try { await api("/api/agents/" + agentId + "/abort", { method: "POST" }); } catch (e) { toastError(e); }
  }

  async function replyAsk(askId, body) {
    if (!agentId || !askId) return;
    const backTo = Number.isInteger(body.backTo) ? body.backTo : null;
    if (backTo != null) {
      set({ ...stateRef.current, items: backAsk(stateRef.current.items, askId, backTo) });
      try {
        await api("/api/agents/" + agentId + "/ui", {
          method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ id: askId, value: BACK }),
        });
      } catch (e) { toastError(e); }
      return;
    }
    const cancelled = !!body.cancelled;
    const answer = cancelled ? "Cancelled" : body.confirmed === true ? "Yes" : body.confirmed === false ? "No" : (body.value || "Answered");
    const cur = stateRef.current;
    set({ ...cur, items: answerAsk(cur.items, askId, answer, cancelled), waiting: false, status: cur.streaming ? "streaming" : "idle" });
    try {
      await api("/api/agents/" + agentId + "/ui", {
        method: "POST", headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ id: askId, cancelled: body.cancelled, value: body.value, confirmed: body.confirmed }),
      });
    } catch (e) {
      // The dialog is gone (restart, answered elsewhere): reopen honestly.
      set({ ...stateRef.current, items: unanswerAsk(stateRef.current.items, askId) });
      toastError(e);
    }
  }

  function toggleTool(id) {
    set({ ...stateRef.current, items: stateRef.current.items.map((it) => (it.kind === "tool" && it.id === id ? { ...it, expanded: !it.expanded } : it)) });
  }

  function toggleFiles(index) {
    set({ ...stateRef.current, items: stateRef.current.items.map((it, i) => i === index && it.kind === "files" ? { ...it, expanded: !it.expanded } : it) });
  }

  return { state, scrollTick, send, abort, replyAsk, abortBash, toggleTool, toggleFiles };
}
