import { agentsOf, displayAgentName } from "./tree.js";

// needsYou (ADR-0044): the phone's home is a queue of decisions. Live
// dialogs come first — they expire, and the agent is blocked on them —
// then blocking inbox items, newest first. Everything an entry needs to
// be answered without opening another screen rides along.
//   entry := { key, kind: "ask"|"inbox", agentId, agentName, where, title,
//              message, method, options, dialogId, placeholder, prefill,
//              itemId, verbs, reason, at }
export function needsYou({ workspaces, freeAgents, inbox }) {
  const out = [];
  const seen = new Set();
  const push = (a, where) => {
    if (!a || !a.id || seen.has(a.id)) return;
    seen.add(a.id);
    if (!a.waiting || !a.dialog || !a.dialog.id) return;
    const d = a.dialog;
    out.push({
      key: "ask:" + a.id, kind: "ask", agentId: a.id, agentName: displayAgentName(a, where || null),
      where: (where && where.name) || "", title: d.title || methodLabel(d.method), message: d.message || "",
      method: d.method || "select", options: (d.options || []).filter((o) => o !== "‹ back"), dialogId: d.id,
      placeholder: d.placeholder || "", prefill: d.prefill || "", at: null,
    });
  };
  for (const ws of workspaces || []) for (const a of agentsOf(ws)) push(a, ws);
  for (const a of freeAgents || []) push(a, null);
  out.sort((x, y) => x.agentName.localeCompare(y.agentName));
  const items = (inbox || []).filter((it) => it && it.blocking && it.state !== "done" && !snoozedNow(it));
  items.sort((a, b) => String(b.createdAt || "").localeCompare(String(a.createdAt || "")));
  for (const it of items) {
    out.push({
      key: "inbox:" + it.id, kind: "inbox", itemId: it.id, agentId: it.sourceKind === "agent" ? it.sourceId || "" : "",
      agentName: "", where: "", title: it.title || "", message: it.body || "", reason: it.reason || "",
      inboxKind: it.kind || "", verbs: it.allowedResponses || [], at: it.createdAt || null,
    });
  }
  return out;
}

function snoozedNow(it, now = Date.now()) {
  if (!it.snoozedUntil) return false;
  const t = Date.parse(it.snoozedUntil);
  return Number.isFinite(t) && t > now;
}

export function methodLabel(method) {
  switch (method) {
    case "confirm": return "Needs a yes or no";
    case "input": return "Needs a line of text";
    case "editor": return "Needs some text";
    default: return "Needs a choice";
  }
}

// verbLabel: the inbox's allowed_responses (ADR-0037) as button copy.
export function verbLabel(verb) {
  switch (verb) {
    case "accept": return "Accept";
    case "edit": return "Edit & send";
    case "respond": return "Reply";
    case "ignore": return "Ignore";
    default: return verb;
  }
}
