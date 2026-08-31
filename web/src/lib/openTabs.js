const KEY = "picode-tabs";

export function readOpenTabs() {
  try {
    const j = JSON.parse(localStorage.getItem(KEY) || "null");
    if (!j || !Array.isArray(j.ids)) return { ids: [], selected: null };
    const ids = j.ids.map((x) => String(x || "")).filter(Boolean);
    const selected = j.selected && ids.includes(j.selected) ? j.selected : (ids[0] || null);
    return { ids, selected };
  } catch {
    return { ids: [], selected: null };
  }
}

export function writeOpenTabs(ids, selected) {
  const clean = [...new Set((ids || []).map((x) => String(x || "")).filter(Boolean))];
  const sel = selected && clean.includes(selected) ? selected : (clean[0] || null);
  localStorage.setItem(KEY, JSON.stringify({ ids: clean, selected: sel }));
}

export function moveTab(ids, fromId, toId) {
  const list = ids || [];
  const from = list.indexOf(fromId);
  const to = list.indexOf(toId);
  if (from < 0 || to < 0 || from === to) return list;
  const next = list.slice();
  next.splice(from, 1);
  next.splice(to, 0, fromId);
  return next;
}

export function filterOpenTabs(saved, exists) {
  const ids = (saved.ids || []).filter(exists);
  const selected = saved.selected && ids.includes(saved.selected) ? saved.selected : (ids[0] || null);
  return { ids, selected };
}

const TERM_KEY = "picode-term-view";

// Which agents were last viewed in the terminal (TUI dock), so a reload
// lands back in the terminal instead of the chat.
export function readTermWanted() {
  try {
    const j = JSON.parse(localStorage.getItem(TERM_KEY) || "[]");
    if (!Array.isArray(j)) return [];
    return j.map((x) => String(x || "")).filter(Boolean);
  } catch {
    return [];
  }
}

export function writeTermWanted(ids) {
  const clean = [...new Set((ids || []).map((x) => String(x || "")).filter(Boolean))];
  localStorage.setItem(TERM_KEY, JSON.stringify(clean));
}

const GIT_KEY = "picode-git-owners";

// A git tab is identified by its repository (ADR-0022), but a reload has to
// re-fetch it through an owner that authorises the read. This remembers which
// owner opened each graph tab so the tab can come back.
export function readGitOwners() {
  try {
    const j = JSON.parse(localStorage.getItem(GIT_KEY) || "{}");
    if (!j || typeof j !== "object" || Array.isArray(j)) return {};
    const out = {};
    for (const [tab, owner] of Object.entries(j)) {
      if (!owner || typeof owner !== "object") continue;
      const kind = owner.kind === "term" ? "term" : "agent";
      const id = String(owner.id || "");
      if (id) out[tab] = { kind, id, name: String(owner.name || "") };
    }
    return out;
  } catch {
    return {};
  }
}

export function writeGitOwners(map) {
  try {
    localStorage.setItem(GIT_KEY, JSON.stringify(map || {}));
  } catch {
    /* private mode, quota — the tab simply will not survive a reload */
  }
}

const TREE_KEY = "picode-tree-owners";

// Same shape as the git owners, for the file tree (ADR-0028): the tab is a
// folder, the reload needs an owner to re-authorise reading it.
export function readTreeOwners() {
  try {
    const j = JSON.parse(localStorage.getItem(TREE_KEY) || "{}");
    if (!j || typeof j !== "object" || Array.isArray(j)) return {};
    const out = {};
    for (const [tab, owner] of Object.entries(j)) {
      if (!owner || typeof owner !== "object") continue;
      const kind = owner.kind === "term" || owner.kind === "workspace" ? owner.kind : "agent";
      const id = String(owner.id || "");
      if (id) out[tab] = { kind, id, name: String(owner.name || "") };
    }
    return out;
  } catch {
    return {};
  }
}

export function writeTreeOwners(map) {
  try {
    localStorage.setItem(TREE_KEY, JSON.stringify(map || {}));
  } catch {
    /* private mode, quota — the tab simply will not survive a reload */
  }
}
