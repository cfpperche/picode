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
// Web tab addresses, for shells without a webview to read them back from.
// The desktop shell gets each page's URL from its own WebView2 (`btab_meta`);
// a plain browser has nothing to ask, so the tab restores what it opened.
const WEBTAB_KEY = "picode-webtab-urls";

export function readWebTabUrls() {
  try {
    const j = JSON.parse(localStorage.getItem(WEBTAB_KEY) || "null");
    if (!j || typeof j !== "object") return {};
    const out = {};
    for (const [id, v] of Object.entries(j)) {
      const url = v && typeof v.url === "string" ? v.url : "";
      if (!id || !url) continue;
      out[id] = { url, title: typeof v.title === "string" ? v.title : "" };
    }
    return out;
  } catch {
    return {};
  }
}

export function writeWebTabUrls(map) {
  try {
    const out = {};
    for (const [id, v] of Object.entries(map || {})) {
      if (!id || !v || !v.url) continue;
      out[id] = { url: v.url, title: v.title || "" };
    }
    localStorage.setItem(WEBTAB_KEY, JSON.stringify(out));
  } catch { /* a full or blocked storage is not worth breaking a tab over */ }
}

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

const SPLIT_KEY = "picode-agent-splits";
const SPLIT_URLS_KEY = "picode-agent-split-urls";
const splitEmpty = () => ({ panes: {}, ratios: {}, max: {}, urls: {} });
const clampPct = (n) => Math.min(80, Math.max(20, Math.round(n)));
const cleanPaneId = (v) => {
  const id = String(v ?? "");
  return /^\d+$/.test(id) ? id : "";
};

// The agent split (ADR-0135): which agent tab hosts a work-browser pane,
// the dragged width, the maximized flag and the last url per pane. Kept in
// localStorage so a shell relaunch brings the panes back at the same page
// (owner directive 2026-09-14). A pane id is the shell's webview id, which
// restarts at zero every session — the caller seeds its sequence from here.
export function readAgentSplits() {
  const out = splitEmpty();
  try {
    const j = JSON.parse(localStorage.getItem(SPLIT_KEY) || "null");
    if (j && typeof j === "object" && !Array.isArray(j)) {
      for (const [key, id] of Object.entries(j.panes || {})) {
        const wid = cleanPaneId(id);
        if (key && wid) out.panes[key] = wid;
      }
      for (const [key, pct] of Object.entries(j.ratios || {})) {
        if (key in out.panes && Number.isFinite(pct)) out.ratios[key] = clampPct(pct);
      }
      for (const [key, on] of Object.entries(j.max || {})) {
        if (key in out.panes && on === true) out.max[key] = true;
      }
    }
    const u = JSON.parse(localStorage.getItem(SPLIT_URLS_KEY) || "null");
    if (u && typeof u === "object" && !Array.isArray(u)) {
      for (const [id, url] of Object.entries(u)) {
        const wid = cleanPaneId(id);
        if (wid && url) out.urls[wid] = String(url);
      }
    }
  } catch {
    return splitEmpty();
  }
  return out;
}

export function writeAgentSplits({ panes, ratios, max } = {}) {
  try {
    const p = {};
    for (const [key, id] of Object.entries(panes || {})) {
      const wid = cleanPaneId(id);
      if (key && wid) p[key] = wid;
    }
    const r = {};
    for (const [key, pct] of Object.entries(ratios || {})) {
      if (key in p && Number.isFinite(pct)) r[key] = clampPct(pct);
    }
    const m = {};
    for (const [key, on] of Object.entries(max || {})) if (key in p && on === true) m[key] = true;
    localStorage.setItem(SPLIT_KEY, JSON.stringify({ panes: p, ratios: r, max: m }));
  } catch {
    /* private mode, quota — the split simply will not survive a relaunch */
  }
}

export function writeAgentSplitUrls(urls) {
  try {
    const u = {};
    for (const [id, url] of Object.entries(urls || {})) {
      const wid = cleanPaneId(id);
      if (wid && url) u[wid] = String(url);
    }
    localStorage.setItem(SPLIT_URLS_KEY, JSON.stringify(u));
  } catch {
    /* private mode, quota */
  }
}

// A split only exists while its host tab does: boot drops the panes whose
// agent or terminal did not come back, and the urls with them. `tabExists`
// is the caller's predicate over the pruned strip.
export function filterAgentSplits(splits, tabExists) {
  const s = splits || splitEmpty();
  const panes = {};
  for (const [key, id] of Object.entries(s.panes || {})) {
    if (tabExists(key)) panes[key] = id;
  }
  const ratios = {};
  for (const [key, pct] of Object.entries(s.ratios || {})) {
    if (key in panes) ratios[key] = pct;
  }
  const max = {};
  for (const [key, on] of Object.entries(s.max || {})) {
    if (key in panes && on === true) max[key] = true;
  }
  const urls = {};
  const live = new Set(Object.values(panes));
  for (const [id, url] of Object.entries(s.urls || {})) {
    if (live.has(id)) urls[id] = url;
  }
  return { panes, ratios, max, urls };
}

const DASH_RANGE_KEY = "picode-dash-range";
const DASH_RANGES = ["today", "7d", "30d", "all"];
const DASH_STATS_PREFIX = "picode-dash-stats:";
const DASH_STATS_TTL_MS = 10 * 60 * 1000;

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

// Last good /api/sessions/stats payload per range, so the dashboard paints
// instantly on reload instead of a full skeleton over a cold ~8s parse.
// Served stale while the fresh fetch runs underneath (F1); the fetch still
// replaces it on success. TTL is a freshness hint for the "updated" line,
// never a gate — stale numbers with a refresh beat a blank well.
export function readDashboardStats(range) {
  if (!DASH_RANGES.includes(range)) return null;
  try {
    const j = JSON.parse(localStorage.getItem(DASH_STATS_PREFIX + range) || "null");
    if (!j || typeof j !== "object" || !j.data || typeof j.data !== "object") return null;
    if (!j.data.current || !Array.isArray(j.data.series)) return null;
    return { data: j.data, at: typeof j.at === "string" ? j.at : null };
  } catch {
    return null;
  }
}

export function writeDashboardStats(range, data) {
  if (!DASH_RANGES.includes(range) || !data || typeof data !== "object") return;
  try {
    localStorage.setItem(DASH_STATS_PREFIX + range, JSON.stringify({ at: new Date().toISOString(), data }));
  } catch { /* private mode, quota — the dashboard simply loads cold */ }
}

export function dashboardStatsAge(at) {
  if (!at) return Infinity;
  const t = Date.parse(at);
  if (!Number.isFinite(t)) return Infinity;
  return Date.now() - t;
}

export { DASH_STATS_TTL_MS };

const FILE_WT_KEY = "picode-file-worktrees";

// Which checkout each file tab reads through: a tab id is owner+path, so a
// worktree tab's {ref, root} lives beside it. Persisted, not derived — after
// a reload the tab must read the same checkout, never the same relative path
// through the anchor folder (one path, two different files).
export function readFileWorktrees() {
  try {
    const j = JSON.parse(localStorage.getItem(FILE_WT_KEY) || "null");
    if (!j || typeof j !== "object") return {};
    const out = {};
    for (const [id, v] of Object.entries(j)) {
      if (id && v && typeof v.ref === "string" && v.ref && typeof v.root === "string" && v.root) {
        out[id] = { ref: v.ref, root: v.root, branch: typeof v.branch === "string" ? v.branch : "" };
      }
    }
    return out;
  } catch {
    return {};
  }
}

export function writeFileWorktrees(map) {
  try {
    localStorage.setItem(FILE_WT_KEY, JSON.stringify(map || {}));
  } catch { /* preference is optional */ }
}

const DASH_SCOPE_KEY = "picode-dash-scope";

// The dashboard scope picker (machine|picode) retired with ADR-0127 — the
// dashboard measures the whole machine. Drop the stale key so a returning
// viewer does not carry a preference nothing reads.
try { localStorage.removeItem(DASH_SCOPE_KEY); } catch { /* private mode */ }
