import { renamedTabId } from "./routes.js";

const KEY = "picode-tabs";

// A renamed app's tab id is rewritten on the way out of storage (ADR-0118:
// `x:matrix` is the Canvas app's tab), so a strip saved before the rename
// reopens the app instead of dropping a tab that answers to nothing. It is
// one map, shared with the hash redirect (lib/routes.js), and the rewrite
// sticks the next time the strip is written.
const canonical = (id) => renamedTabId(id) || id;

export function readOpenTabs() {
  try {
    const j = JSON.parse(localStorage.getItem(KEY) || "null");
    if (!j || !Array.isArray(j.ids)) return { ids: [], selected: null };
    const ids = [...new Set(j.ids.map((x) => canonical(String(x || ""))).filter(Boolean))];
    const want = j.selected ? canonical(String(j.selected)) : "";
    const selected = want && ids.includes(want) ? want : (ids[0] || null);
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
      const kind = owner.kind === "term" || owner.kind === "workspace" ? owner.kind : "agent";
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

// Same shape as the git owners, for the file tree (ADR-0030): the tab is a
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

const DASH_RANGE_KEY = "picode-dash-range";
const DASH_RANGES = ["today", "7d", "30d", "all"];

// The dashboard's date-range choice is a per-viewer preference, not a
// navigable identity — it has no hash route (see ADR on the session
// observability dashboard), same reasoning as termView/git/tree owners above.
export function readDashboardRange() {
  const v = (() => { try { return localStorage.getItem(DASH_RANGE_KEY) || ""; } catch { return ""; } })();
  return DASH_RANGES.includes(v) ? v : "7d";
}

export function writeDashboardRange(range) {
  if (!DASH_RANGES.includes(range)) return;
  try { localStorage.setItem(DASH_RANGE_KEY, range); } catch { /* private mode, quota */ }
}

const DASH_SCOPE_KEY = "picode-dash-scope";
const DASH_SCOPES = ["machine", "picode"];

// Which folders the dashboard counts (ADR-0097). A view filter, so it lives
// beside the range in localStorage and not in the hash — hash routes name
// what object is open, never how a view is filtered.
//
// The default is the whole machine: the v1 dashboard counted every session
// wherever it ran, and a narrower default would silently drop rows the
// surface has always shown.
export function readDashboardScope() {
  const v = (() => { try { return localStorage.getItem(DASH_SCOPE_KEY) || ""; } catch { return ""; } })();
  return DASH_SCOPES.includes(v) ? v : "machine";
}

export function writeDashboardScope(scope) {
  if (!DASH_SCOPES.includes(scope)) return;
  try { localStorage.setItem(DASH_SCOPE_KEY, scope); } catch { /* private mode, quota */ }
}
