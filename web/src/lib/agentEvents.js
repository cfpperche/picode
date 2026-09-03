import { putAsk, timeoutAsk, cancelOpenAsks, walkReply, askJustAnswered, noteAsk, slashNoteTarget } from "./askForm.js";
import { mergeAssistant } from "./assistantMsg.js";
import { fileChangeFromTool } from "./diff.js";
import { isSearchTool, hitsFromResult } from "./searchCards.js";
import { alertFromPi } from "./piError.js";
import { humanizeError } from "./api.js";
import { summarizeArgs } from "./toolArgs.js";
import { previewFromDetails } from "./toolPreview.js";

// Pure reducer over the agent WebSocket stream (ADR-0044). The desktop's
// inline handleEvent (desktop/App.jsx) is the reference for every case
// below; this module carries the same item-level semantics without React
// state or refs, so the mobile shell can drive it from one small hook and
// a test can drive it from a list of events. Side effects come back as
// plain data in `effects` — the hook decides what a toast or a reply
// request looks like.
//
//   state   := { items, streaming, waiting, status, pendingPayload }
//   effect  := { type: "toast", level: "info"|"error", text }
//            | { type: "replyUI", id, value }        // auto-answer BACK while walking
//            | { type: "scroll" }
export const initialAgentState = Object.freeze({
  items: [],
  streaming: false,
  waiting: false,
  status: "idle",
  pendingPayload: "",
});

export function appendDelta(cur, cls, actor, delta) {
  const last = cur[cur.length - 1];
  if (last && last.kind === "block" && last.actor === actor && last.cls === cls) {
    const next = cur.slice();
    next[next.length - 1] = { ...last, text: last.text + delta };
    return next;
  }
  return [...cur, { kind: "block", cls, actor, text: delta }];
}

function statusOf(streaming, waiting) {
  return waiting ? "waiting" : streaming ? "streaming" : "idle";
}

function withAlert(state, a, now, stopStreaming) {
  if (!a) return { state, effects: [] };
  const items = [...state.items, { kind: "alert", level: a.level, text: a.text, ts: now }];
  const effects = [{ type: "scroll" }];
  let streaming = state.streaming;
  if (a.level === "error") {
    if (stopStreaming) streaming = false;
    effects.push({ type: "toast", level: "error", text: a.text });
  }
  return { state: { ...state, items, streaming, status: statusOf(streaming, state.waiting) }, effects };
}

export function reduceAgentEvent(state, ev, now = Date.now()) {
  const s = state || initialAgentState;
  const e = ev || {};
  switch (e.type) {
    case "snapshot": {
      const waiting = !!e.waiting;
      const streaming = !!e.streaming;
      const items = waiting && e.dialog ? putAsk(s.items, e.dialog, "open") : cancelOpenAsks(s.items);
      return { state: { ...s, items, waiting, streaming, status: statusOf(streaming, waiting) }, effects: [] };
    }
    case "agent_start":
      return { state: { ...s, streaming: true, status: statusOf(true, s.waiting) }, effects: [{ type: "scroll" }] };
    case "agent_settled":
      return { state: { ...s, streaming: false, status: statusOf(false, s.waiting) }, effects: [] };
    case "message_update": {
      const d = e.assistantMessageEvent;
      if (!d) return { state: s, effects: [] };
      if (d.type === "text_delta") return { state: { ...s, items: appendDelta(s.items, "", "agent", d.delta || "") }, effects: [{ type: "scroll" }] };
      if (d.type === "thinking_delta") return { state: { ...s, items: appendDelta(s.items, "thinking", "thinking", d.delta || "") }, effects: [{ type: "scroll" }] };
      return { state: s, effects: [] };
    }
    case "tool_execution_start": {
      const item = {
        kind: "tool", id: e.toolCallId, name: e.toolName || "tool",
        args: summarizeArgs(e.args), toolArgs: e.args || {}, status: "···",
        detail: JSON.stringify(e.args || {}, null, 2), expanded: false,
        change: fileChangeFromTool(e.toolName, e.args, null), preview: null, ts: now,
      };
      return { state: { ...s, items: [...s.items, item] }, effects: [{ type: "scroll" }] };
    }
    case "tool_execution_update": {
      // ADR-0057: a tool streaming partial results may carry a preview frame;
      // latest wins, nothing else about the item moves.
      const preview = previewFromDetails(e.partialResult && e.partialResult.details);
      if (!preview) return { state: s, effects: [] };
      const items = s.items.map((it) => (it.kind === "tool" && it.id === e.toolCallId ? { ...it, preview } : it));
      return { state: { ...s, items }, effects: [] };
    }
    case "tool_execution_end": {
      const items = s.items.map((it) => {
        if (it.kind !== "tool" || it.id !== e.toolCallId) return it;
        const change = fileChangeFromTool(e.toolName || it.name, e.args, e.result) || it.change;
        const hits = isSearchTool(e.toolName || it.name) ? hitsFromResult(e.result) : [];
        return {
          ...it, status: e.isError ? "error" : "ok",
          detail: JSON.stringify(e.result || {}, null, 2), result: e.result,
          expanded: it.expanded || hits.length > 0, change,
          preview: previewFromDetails(e.result && e.result.details) || it.preview,
        };
      });
      return { state: { ...s, items }, effects: [] };
    }
    case "bash_execution_update": {
      const chunk = e.delta || "";
      if (!chunk) return { state: s, effects: [] };
      const items = s.items.map((it) => (it.kind === "bash" && it.status === "run" ? { ...it, output: (it.output || "") + chunk } : it));
      return { state: { ...s, items }, effects: [{ type: "scroll" }] };
    }
    case "enqueue_accepted": {
      const text = s.pendingPayload;
      const items = text
        ? [...s.items, { kind: "block", cls: "user", actor: "You", chip: e.kind || "prompt", text, ts: now }]
        : s.items;
      return { state: { ...s, items, pendingPayload: "" }, effects: [{ type: "scroll" }] };
    }
    case "message_end":
    case "turn_end": {
      const m = e.message || {};
      let next = s;
      const effects = [];
      if (m.role === "assistant") {
        next = { ...next, items: mergeAssistant(next.items, m) };
        effects.push({ type: "scroll" });
      }
      const r = withAlert(next, alertFromPi(e), now, true);
      return { state: r.state, effects: [...effects, ...r.effects] };
    }
    case "agent_end":
      return withAlert(s, alertFromPi(e), now, !e.willRetry);
    case "auto_retry_start":
      return withAlert(s, alertFromPi(e), now, false);
    case "auto_retry_end":
    case "extension_error":
      return withAlert(s, alertFromPi(e), now, true);
    case "compaction_end": {
      const sum = !e.aborted && e.result && e.result.summary ? String(e.result.summary) : "";
      if (!sum || s.items.some((it) => it.kind === "compaction" && it.text === sum)) return { state: s, effects: [] };
      return { state: { ...s, items: [...s.items, { kind: "compaction", text: sum, ts: now }] }, effects: [{ type: "scroll" }] };
    }
    case "task_failed": {
      const items = [...s.items, { kind: "alert", level: "error", text: humanizeError(e.error || "Task failed"), ts: now }];
      return { state: { ...s, items, streaming: false, status: statusOf(false, s.waiting) }, effects: [] };
    }
    case "enqueue_rejected":
      return { state: s, effects: [{ type: "toast", level: "error", text: humanizeError(e.error || "Rejected") }] };
    case "extension_ui_request": {
      const method = e.method || "";
      if (method === "select" || method === "confirm" || method === "input" || method === "editor") {
        // Walking back to a clicked pill: answer BACK to the wrong fields
        // instead of showing them; the target field arrives next.
        const back = walkReply(s.items, e);
        if (back) {
          return { state: { ...s, waiting: true, status: "waiting" }, effects: e.id ? [{ type: "replyUI", id: e.id, value: back }] : [] };
        }
        return { state: { ...s, waiting: true, status: "waiting", items: putAsk(s.items, e, "open") }, effects: [{ type: "scroll" }] };
      }
      if (method === "notify") {
        const msg = e.message || "Notice";
        // Right after a finished form the notify is its result: fold it
        // into the card. A quiet slash command's notify is its whole
        // result: keep it in the thread. Anything else is a toast.
        if (e.notifyType !== "error" && askJustAnswered(s.items)) {
          return { state: { ...s, items: noteAsk(s.items, msg) }, effects: [] };
        }
        const cmd = slashNoteTarget(s.items);
        if (cmd) {
          const items = [...s.items, { kind: "note", cmd, level: e.notifyType || "info", text: msg, ts: now }];
          return { state: { ...s, items }, effects: [{ type: "scroll" }] };
        }
        return { state: s, effects: [{ type: "toast", level: e.notifyType === "error" ? "error" : "info", text: msg }] };
      }
      return { state: s, effects: [] };
    }
    case "extension_ui_timeout":
      return { state: { ...s, waiting: false, status: statusOf(s.streaming, false), items: timeoutAsk(s.items, e.id) }, effects: [] };
    case "exit":
      // The pi process is gone: any open ask card is dead — close it so
      // nothing clickable points at a dead dialog.
      return { state: { ...s, streaming: false, waiting: false, status: "idle", items: cancelOpenAsks(s.items) }, effects: [] };
    default:
      return { state: s, effects: [] };
  }
}

// Local (not event-driven) transitions the hook needs, kept here so the
// tests cover them beside the events they interleave with.
export function markSent(state, { kind, text, images, ts, busy }) {
  const item = { kind: "block", cls: "user", actor: "You", chip: kind, text, images: images || [], ts };
  return { ...state, items: [...state.items, item], pendingPayload: "", streaming: busy ? state.streaming : true, status: busy ? state.status : "streaming" };
}

export function markUndelivered(state, ts, reason) {
  const items = state.items.map((it) => (it.kind === "block" && it.cls === "user" && it.ts === ts
    ? { ...it, text: it.text + "\n\n— not delivered: " + reason }
    : it));
  return { ...state, items, streaming: false, status: statusOf(false, state.waiting) };
}

export function markAborted(state) {
  const items = cancelOpenAsks(state.items).map((it) => (
    it.kind === "block" && it.cls === "user" && it.chip === "steer" && !it.dropped ? { ...it, dropped: true } : it
  ));
  return { ...state, items, streaming: false, waiting: false, status: "idle" };
}
