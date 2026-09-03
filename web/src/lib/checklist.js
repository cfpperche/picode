// Internal checklists (ADR-0055): the shells hold one map agentId →
// checklist {items, absent, updatedAt} and project a single operator line
// per agent, the way Tachyon does: the current step "(2/4) text", or a
// discrete "No checklist" when the contract was not met. No checklist
// known → null, and the row shows nothing (silence is not absence).

export const GLYPH = { pending: "☐", "in-progress": "◐", completed: "☑" };

export function currentStep(items) {
  const list = Array.isArray(items) ? items : [];
  if (!list.length) return null;
  let i = list.findIndex((it) => it && it.status === "in-progress");
  if (i < 0) i = list.findIndex((it) => it && it.status === "pending");
  if (i < 0) i = list.length - 1;
  return { text: String(list[i].text || ""), position: i + 1, total: list.length };
}

export function countDone(items) {
  return (Array.isArray(items) ? items : []).filter((it) => it && it.status === "completed").length;
}

// checklistLine(c) -> {kind:"step", text, position, total} | {kind:"absent"} | null
export function checklistLine(c) {
  if (!c) return null;
  const step = currentStep(c.items);
  if (step) return { kind: "step", ...step };
  return c.absent ? { kind: "absent" } : null;
}

// applyChecklists(map, ev) -> the next map after a feed event.
export function applyChecklists(map, ev) {
  const cur = map || {};
  const d = ev && ev.data ? ev.data : {};
  if (ev.type === "agent.checklist" && d.agentId) return { ...cur, [d.agentId]: d };
  if (ev.type === "agent.deleted" && d.id && cur[d.id]) {
    const next = { ...cur };
    delete next[d.id];
    return next;
  }
  return cur;
}

// indexChecklists(list) -> the map from GET /api/checklists.
export function indexChecklists(list) {
  const out = {};
  for (const c of Array.isArray(list) ? list : []) if (c && c.agentId) out[c.agentId] = c;
  return out;
}

// checklistItems(it) -> the items a chat tool item carries: the result's
// details once the call ended, else the call's arguments.
export function checklistItems(it) {
  const fromResult = it && it.result && it.result.details && Array.isArray(it.result.details.items) ? it.result.details.items : null;
  const fromArgs = it && it.toolArgs && Array.isArray(it.toolArgs.items) ? it.toolArgs.items : [];
  const list = fromResult || fromArgs;
  return list
    .filter((x) => x && typeof x.text === "string")
    .map((x) => ({ text: x.text, status: GLYPH[x.status] ? x.status : "pending" }));
}

export function checklistLevelLabel(level) {
  return level === "always" ? "Always" : level === "never" ? "Never" : "Before changes";
}

export function checklistChoices() {
  return [
    { id: "changes", label: "Before changes", hint: "A plan before the first edit, write or bash" },
    { id: "always", label: "Always", hint: "Every task, read-only answers too" },
    { id: "never", label: "Never", hint: "The tool stays, nothing is required" },
  ];
}
