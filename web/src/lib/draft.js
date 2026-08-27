const KEY = "picode-drafts";

function loadAll() {
  try {
    const j = JSON.parse(localStorage.getItem(KEY) || "{}");
    return j && typeof j === "object" && !Array.isArray(j) ? j : {};
  } catch {
    return {};
  }
}

function saveAll(map) {
  try {
    localStorage.setItem(KEY, JSON.stringify(map));
  } catch { /* quota / private mode */ }
}

function normKind(kind) {
  return kind === "steer" || kind === "follow_up" ? kind : "prompt";
}

export function readDraft(agentId) {
  if (!agentId) return { text: "", kind: "prompt" };
  const row = loadAll()[agentId];
  if (!row || typeof row !== "object") return { text: "", kind: "prompt" };
  return { text: String(row.text || ""), kind: normKind(row.kind) };
}

export function writeDraft(agentId, text, kind) {
  if (!agentId) return;
  const t = String(text || "");
  const all = loadAll();
  if (!t.trim()) {
    if (!(agentId in all)) return;
    delete all[agentId];
    saveAll(all);
    return;
  }
  all[agentId] = { text: t, kind: normKind(kind) };
  saveAll(all);
}

export function clearDraft(agentId) {
  writeDraft(agentId, "", "prompt");
}
