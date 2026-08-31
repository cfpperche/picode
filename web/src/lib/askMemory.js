const KEY = "picode-asks";

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

function slot(agentId, sessionPath) {
  return String(agentId || "") + "\t" + String(sessionPath || "");
}

function isSlashUser(it) {
  return it && it.kind === "block" && it.cls === "user" && /^\s*\//.test(it.text || "");
}

function keepExtra(it) {
  if (!it) return false;
  if (isSlashUser(it)) return true;
  if (it.kind === "ask" && it.status !== "cancelled" && it.status !== "timeout") return true;
  return false;
}

export function writeAskMemory(agentId, sessionPath, items) {
  if (!agentId || !sessionPath) return;
  const extras = (items || []).filter(keepExtra);
  const all = loadAll();
  const k = slot(agentId, sessionPath);
  if (!extras.length) {
    if (!(k in all)) return;
    delete all[k];
    saveAll(all);
    return;
  }
  all[k] = extras.slice(-40);
  saveAll(all);
}

export function mergeAskMemory(agentId, sessionPath, items) {
  if (!agentId || !sessionPath) return items || [];
  const extras = loadAll()[slot(agentId, sessionPath)];
  if (!Array.isArray(extras) || extras.length === 0) return items || [];
  const out = (items || []).slice();
  for (const ex of extras) {
    if (isSlashUser(ex)) {
      if (!out.some((it) => isSlashUser(it) && it.text === ex.text)) out.push(ex);
      continue;
    }
    if (ex.kind === "ask") {
      const dup = out.some((it) => it.kind === "ask" && (it.id === ex.id || (it.ts && it.ts === ex.ts)));
      if (!dup) out.push(ex);
    }
  }
  out.sort((a, b) => (a.ts || 0) - (b.ts || 0));
  return out;
}
