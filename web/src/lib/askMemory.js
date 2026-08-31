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

// An agent whose only activity is extension commands has no session file
// yet; its thread lives under the "@live" slot until a real turn creates
// one (the write then migrates and drops the live slot).
function slot(agentId, sessionPath) {
  return String(agentId || "") + "\t" + String(sessionPath || "@live");
}

function isSlashUser(it) {
  return it && it.kind === "block" && it.cls === "user" && /^\s*\//.test(it.text || "");
}

function keepExtra(it) {
  if (!it) return false;
  if (isSlashUser(it)) return true;
  if (it.kind === "ask" && it.status !== "cancelled" && it.status !== "timeout") return true;
  if (it.kind === "note") return true; // a slash command's one-line result
  return false;
}

export function writeAskMemory(agentId, sessionPath, items) {
  if (!agentId) return;
  const extras = (items || []).filter(keepExtra);
  const all = loadAll();
  const k = slot(agentId, sessionPath);
  const live = slot(agentId, "");
  let dirty = false;
  if (sessionPath && live in all) {
    // The thread now has a real session file; the live slot is stale.
    delete all[live];
    dirty = true;
  }
  if (!extras.length) {
    if (k in all) {
      delete all[k];
      dirty = true;
    }
    if (dirty) saveAll(all);
    return;
  }
  all[k] = extras.slice(-40);
  saveAll(all);
}

export function mergeAskMemory(agentId, sessionPath, items) {
  if (!agentId) return items || [];
  const extras = loadAll()[slot(agentId, sessionPath)];
  if (!Array.isArray(extras) || extras.length === 0) return items || [];
  const out = (items || []).slice();
  for (const ex of extras) {
    if (isSlashUser(ex)) {
      // ts + text: the same command typed twice is two real bubbles.
      if (!out.some((it) => isSlashUser(it) && it.text === ex.text && (it.ts || 0) === (ex.ts || 0))) {
        out.push(ex);
      }
      continue;
    }
    if (ex.kind === "ask" || ex.kind === "note") {
      const dup = out.some((it) => it.kind === ex.kind && ((ex.id && it.id === ex.id) || (it.ts && it.ts === ex.ts)));
      if (!dup) out.push(ex);
    }
  }
  out.sort((a, b) => (a.ts || 0) - (b.ts || 0));
  return out;
}
