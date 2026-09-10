// Matrix (ADR-0108): the client side of the store's contract. Pure — no
// React, no fetch. normalize* turn the API's JSON into the shapes the
// surface holds (junk dropped, the way contracts/appPrimitives.js does);
// validate* refuse what the server would refuse, in the server's words, so
// the UI never asks a question it knows the answer to; applyMatrixEvent
// reduces the six feed events over { list: [summaries], byId: { id:
// { matrix, panels } } }. touches(ev, ["matrix"]) in feedReducers.js keys
// on the prefix before the first dot, so matrix.panel.* reaches the
// surface with the rest. The surface's arithmetic lives here too (plan
// docs/plans/matrix-app.md §4.3–§4.6): nextSlot, layoutDiff, bindingState,
// loadPolicy, the suspended-instance LRU and the keyboard's neighbour
// arithmetic — decisions the components only carry out.

import { locate } from "./tree.js";

export const MATRIX_LIMITS = Object.freeze({
  matrices: 64, // per machine
  panels: 500, // per matrix
  name: 80, // characters (code points), never bytes
  cols: 12, // fixed in v1
  minW: 4, // columns
  minH: 8, // rows
});

export const MATRIX_KINDS = Object.freeze(["agent", "terminal"]);
export const MATRIX_COMPACT = Object.freeze(["vertical", "none"]);
export const MATRIX_EVENTS = Object.freeze([
  "matrix.created",
  "matrix.updated",
  "matrix.layout",
  "matrix.panel.added",
  "matrix.panel.removed",
  "matrix.deleted",
]);

const str = (v) => (typeof v === "string" ? v : "");
const nonEmpty = (v) => typeof v === "string" && v !== "";
const runes = (s) => Array.from(s).length; // code points, the way Go counts runes

// normalizeMatrix(json) -> summary | null. The summary is what the list,
// the feed and a PATCH answer carry: never panels.
export function normalizeMatrix(m) {
  if (!m || typeof m !== "object" || !nonEmpty(m.id) || !nonEmpty(m.name)) return null;
  return {
    id: m.id,
    name: m.name,
    compact: MATRIX_COMPACT.includes(m.compact) ? m.compact : "vertical",
    createdAt: str(m.createdAt),
    updatedAt: str(m.updatedAt),
    panelCount: Number.isInteger(m.panelCount) && m.panelCount >= 0 ? m.panelCount : 0,
  };
}

// normalizeMatrixList(GET /api/matrices) -> summaries.
export function normalizeMatrixList(payload) {
  const list = Array.isArray(payload?.matrices) ? payload.matrices : [];
  return list.map(normalizeMatrix).filter(Boolean);
}

// normalizePanel(json) -> panel | null. A panel needs a slot id, a known
// binding and a whole rectangle; anything else cannot be placed.
export function normalizePanel(p) {
  if (!p || typeof p !== "object" || !nonEmpty(p.id) || !MATRIX_KINDS.includes(p.kind)) return null;
  const ref = str(p.ref).trim();
  if (!ref) return null;
  const { x, y, w, h } = p;
  if (![x, y, w, h].every(Number.isInteger)) return null;
  return { id: p.id, kind: p.kind, ref, x, y, w, h, createdAt: str(p.createdAt) };
}

// normalizeMatrixDetail(GET /api/matrices/{id}) -> { matrix, panels } | null.
// The count follows the panels that survived normalization.
export function normalizeMatrixDetail(d) {
  const m = normalizeMatrix(d);
  if (!m) return null;
  const panels = (Array.isArray(d.panels) ? d.panels : []).map(normalizePanel).filter(Boolean);
  return { matrix: { ...m, panelCount: panels.length }, panels };
}

// validateName(name) -> "" | the server's refusal.
export function validateName(name) {
  const n = str(name).trim();
  if (!n) return "name is required";
  if (runes(n) > MATRIX_LIMITS.name) return `name is too long (max ${MATRIX_LIMITS.name} characters)`;
  return "";
}

// validateCompact(mode) -> "" | the server's refusal.
export function validateCompact(mode) {
  return MATRIX_COMPACT.includes(mode) ? "" : `compact must be ${MATRIX_COMPACT.join(" or ")}`;
}

// validatePlacement({x, y, w, h}) -> "" | the server's refusal. The grid
// rule: 12 columns, panels of at least 4×8 cells, rows without end. The
// "whole number" line is the client's own — the server answers a
// fractional cell with "invalid JSON body".
export function validatePlacement(rect) {
  const { x, y, w, h } = rect || {};
  for (const [k, v] of [["x", x], ["y", y], ["w", w], ["h", h]]) {
    if (!Number.isInteger(v)) return `${k} must be a whole number`;
  }
  if (x < 0) return "x must be 0 or more";
  if (y < 0) return "y must be 0 or more";
  if (w < MATRIX_LIMITS.minW) return `w must be at least ${MATRIX_LIMITS.minW} columns`;
  if (h < MATRIX_LIMITS.minH) return `h must be at least ${MATRIX_LIMITS.minH} rows`;
  if (x + w > MATRIX_LIMITS.cols) return `x + w must be at most ${MATRIX_LIMITS.cols} columns`;
  return "";
}

// validatePanel({kind, ref, x, y, w, h}) -> "" | the server's refusal, in
// the server's order: binding first, then the rectangle.
export function validatePanel(panel) {
  const p = panel || {};
  if (!MATRIX_KINDS.includes(p.kind)) return `kind must be ${MATRIX_KINDS.join(" or ")}`;
  if (!str(p.ref).trim()) return "ref is required";
  return validatePlacement(p);
}

// ---- the reducer ---------------------------------------------------------

function byName(list) {
  return [...list].sort((a, b) => {
    const an = a.name.toLowerCase();
    const bn = b.name.toLowerCase();
    if (an !== bn) return an < bn ? -1 : 1;
    return a.createdAt < b.createdAt ? -1 : a.createdAt > b.createdAt ? 1 : 0;
  });
}

function upsertSummary(list, m) {
  return byName([...list.filter((x) => x.id !== m.id), m]);
}

// patchSummary returns the same list when the id is not in it: a summary
// the list has never seen cannot be patched faithfully, and the two
// events that carry a whole summary (created, updated) insert it instead.
function patchSummary(list, id, fn) {
  return list.some((m) => m.id === id) ? list.map((m) => (m.id === id ? fn(m) : m)) : list;
}

function finish(state, list, byId) {
  return list === state.list && byId === state.byId ? state : { ...state, list, byId };
}

const EMPTY = Object.freeze({ list: [], byId: {} });

// applyMatrixEvent(state, ev) -> next | same. state is { list, byId };
// an event for a matrix that is not loaded (no byId entry) only touches
// the summary list. Unknown types and junk return the same object.
export function applyMatrixEvent(state, ev) {
  const s = state || EMPTY;
  const list = Array.isArray(s.list) ? s.list : [];
  const byId = s.byId && typeof s.byId === "object" ? s.byId : {};
  const d = ev && ev.data && typeof ev.data === "object" ? ev.data : {};
  const id = str(d.id);
  if (!id) return s;
  const loaded = byId[id];
  const at = str(d.updatedAt);
  const stamp = (m) => ({ ...m, updatedAt: at || m.updatedAt });
  switch (ev.type) {
    case "matrix.created":
    case "matrix.updated": {
      const m = normalizeMatrix(d);
      if (!m) return s;
      const nextById = loaded ? { ...byId, [id]: { matrix: { ...m, panelCount: loaded.panels.length }, panels: loaded.panels } } : byId;
      return finish(s, upsertSummary(list, m), nextById);
    }
    case "matrix.layout": {
      const nextList = patchSummary(list, id, stamp);
      if (!loaded) return finish(s, nextList, byId);
      const to = new Map();
      for (const p of Array.isArray(d.panels) ? d.panels : []) {
        if (p && nonEmpty(p.id) && !validatePlacement(p)) to.set(p.id, { x: p.x, y: p.y, w: p.w, h: p.h });
      }
      const panels = loaded.panels.map((p) => (to.has(p.id) ? { ...p, ...to.get(p.id) } : p));
      return finish(s, nextList, { ...byId, [id]: { matrix: stamp(loaded.matrix), panels } });
    }
    case "matrix.panel.added": {
      const panel = normalizePanel(d.panel);
      if (!panel) return s;
      if (!loaded) return finish(s, patchSummary(list, id, (m) => ({ ...stamp(m), panelCount: m.panelCount + 1 })), byId);
      const panels = loaded.panels.some((p) => p.id === panel.id)
        ? loaded.panels.map((p) => (p.id === panel.id ? panel : p))
        : [...loaded.panels, panel];
      const count = (m) => ({ ...stamp(m), panelCount: panels.length });
      return finish(s, patchSummary(list, id, count), { ...byId, [id]: { matrix: count(loaded.matrix), panels } });
    }
    case "matrix.panel.removed": {
      const panelId = str(d.panelId);
      if (!panelId) return s;
      if (!loaded) return finish(s, patchSummary(list, id, (m) => ({ ...stamp(m), panelCount: Math.max(0, m.panelCount - 1) })), byId);
      const panels = loaded.panels.filter((p) => p.id !== panelId);
      const count = (m) => ({ ...stamp(m), panelCount: panels.length });
      return finish(s, patchSummary(list, id, count), { ...byId, [id]: { matrix: count(loaded.matrix), panels } });
    }
    case "matrix.deleted": {
      const nextList = list.some((m) => m.id === id) ? list.filter((m) => m.id !== id) : list;
      if (!loaded) return finish(s, nextList, byId);
      const rest = { ...byId };
      delete rest[id];
      return finish(s, nextList, rest);
    }
    default:
      return s;
  }
}

// ---- the surface's arithmetic (plan §4.3–§4.6) ----------------------------

// A new panel is 4×14 cells: ≈ 58×20 characters at 1584 px (phase 0 study);
// minW/minH are the store's 4×8.
export const PANEL_DEFAULT = Object.freeze({ w: 4, h: 14 });
// Chunk loading (§4.5): a body mounts after its wrapper has been near the
// viewport for 300 ms and unmounts 5 s after it left; a suspended xterm is
// disposed past 24 instances or 10 minutes.
export const LOAD_DWELL_MS = 300;
export const UNLOAD_AFTER_MS = 5000;
export const SUSPENDED_MAX = 24;
export const SUSPENDED_TTL_MS = 10 * 60 * 1000;

const wholeRect = (p) => !!p && [p.x, p.y, p.w, p.h].every(Number.isInteger);
const overlaps = (a, b) => a.x < b.x + b.w && b.x < a.x + a.w && a.y < b.y + b.h && b.y < a.y + a.h;

// nextSlot(panels, w, h, cols) -> {x, y}: the first free w×h slot scanning
// rows top-down and left-to-right, else the row under everything. Only a
// panel's edges can start a first-free slot (a free rectangle slides up and
// left until it touches one), so the scan visits edges, not cells.
export function nextSlot(panels, w = PANEL_DEFAULT.w, h = PANEL_DEFAULT.h, cols = MATRIX_LIMITS.cols) {
  const rects = (panels || []).filter(wholeRect);
  const width = Math.min(Math.max(1, w | 0), cols);
  const height = Math.max(1, h | 0);
  const ys = new Set([0]);
  const xs = new Set([0]);
  let bottom = 0;
  for (const r of rects) {
    bottom = Math.max(bottom, r.y + r.h);
    ys.add(r.y + r.h);
    xs.add(r.x + r.w);
  }
  const rows = [...ys].sort((a, b) => a - b);
  const columns = [...xs].filter((x) => x + width <= cols).sort((a, b) => a - b);
  for (const y of rows) {
    for (const x of columns) {
      const cand = { x, y, w: width, h: height };
      if (!rects.some((r) => overlaps(cand, r))) return { x, y };
    }
  }
  return { x: 0, y: bottom };
}

// layoutDiff(prev, next) -> [{id, x, y, w, h}]: the panels whose rectangle
// changed between two layouts, which is exactly what PATCH …/layout takes.
// A panel only in `next` is an add (its own POST), only in `prev` a remove;
// an invalid rectangle is left out so one bad row cannot sink the batch.
export function layoutDiff(prev, next) {
  const before = new Map((prev || []).filter((p) => p && nonEmpty(p.id)).map((p) => [p.id, p]));
  const out = [];
  for (const p of next || []) {
    const b = p && before.get(p.id);
    if (!b || !wholeRect(p) || validatePlacement(p)) continue;
    if (b.x !== p.x || b.y !== p.y || b.w !== p.w || b.h !== p.h) out.push({ id: p.id, x: p.x, y: p.y, w: p.w, h: p.h });
  }
  return out;
}

// bindingState(panel, fleet) -> one row of plan §4.4:
//   terminal-running | terminal-stopped | terminal-gone
//   agent-interactive | agent-managed | agent-stopped | agent-gone
// fleet is the host's { workspaces, freeAgents, terminals }. A shell whose
// tmux session died is still "running" for the panel: opening it revives
// the shell; only a configured CLI terminal has a stopped state of its own.
export function bindingState(panel, fleet) {
  const f = fleet || {};
  if (!panel) return "";
  if (panel.kind === "terminal") {
    const t = (f.terminals || []).find((x) => x && x.id === panel.ref);
    if (!t) return "terminal-gone";
    return t.launchCli && t.running === false ? "terminal-stopped" : "terminal-running";
  }
  if (panel.kind === "agent") {
    const loc = locate(f.workspaces, f.freeAgents, panel.ref);
    const a = loc && loc.agent && loc.agent.id === panel.ref ? loc.agent : null;
    if (!a) return "agent-gone";
    if (a.mode === "interactive") return "agent-interactive";
    if (a.mode === "managed") return "agent-managed";
    return "agent-stopped";
  }
  return "";
}

// loadPolicy(entries, now, pinned, timers) -> { load, unload, timers, wakeAt }
// The chunk-loading decision (§4.5), pure so every row has a test:
//   entries  [{ id, near, loaded }] — near: inside the viewport ± one
//            height (the observer's word) and the surface visible
//   pinned   Set of ids that never unload (dragged, resized, focused,
//            maximized)
//   timers   the previous call's `timers` — the policy's only memory:
//            { id: { loadAt } | { unloadAt } }
// A wrapper loads once it has been near for LOAD_DWELL_MS; leaving cancels
// a pending load at once. A loaded body unloads UNLOAD_AFTER_MS after it
// left; coming back cancels the unload. wakeAt is the earliest deadline,
// 0 when nothing is pending — the caller sleeps until then.
export function loadPolicy(entries, now, pinned, timers) {
  const pin = pinned || new Set();
  const prev = timers || {};
  const next = {};
  const load = [];
  const unload = [];
  let wakeAt = 0;
  const wake = (t) => { if (t && (!wakeAt || t < wakeAt)) wakeAt = t; };
  for (const e of entries || []) {
    if (!e || !nonEmpty(e.id)) continue;
    const t = prev[e.id] || {};
    if (e.near) {
      if (e.loaded) continue;
      const loadAt = t.loadAt || now + LOAD_DWELL_MS;
      if (loadAt <= now) { load.push(e.id); continue; }
      next[e.id] = { loadAt };
      wake(loadAt);
      continue;
    }
    if (!e.loaded || pin.has(e.id)) continue;
    const unloadAt = t.unloadAt || now + UNLOAD_AFTER_MS;
    if (unloadAt <= now) { unload.push(e.id); continue; }
    next[e.id] = { unloadAt };
    wake(unloadAt);
  }
  return { load, unload, timers: next, wakeAt };
}

// suspendedToDispose(suspended, now, max, ttl) -> ids to dispose (§4.5's
// LRU row): the oldest beyond `max` suspended instances and any suspended
// for `ttl` or longer. suspended is [{ id, at }], at = when it was parked.
export function suspendedToDispose(suspended, now, max = SUSPENDED_MAX, ttl = SUSPENDED_TTL_MS) {
  const list = (suspended || []).filter((s) => s && nonEmpty(s.id) && Number.isFinite(s.at)).sort((a, b) => a.at - b.at);
  const over = Math.max(0, list.length - max);
  const out = [];
  list.forEach((s, i) => { if (i < over || now - s.at >= ttl) out.push(s.id); });
  return out;
}

// ---- keyboard (plan §4.6 "Focus") ---------------------------------------

export const PANEL_DIRECTIONS = Object.freeze(["left", "right", "up", "down"]);

const placed = (panels) => (panels || []).filter((p) => p && nonEmpty(p.id) && wholeRect(p));

// panelOrder(panels) -> ids in reading order (top-down, then left-to-right):
// Home, End and the roving tab stop of a matrix nobody has focused yet.
export function panelOrder(panels) {
  return placed(panels).sort((a, b) => a.y - b.y || a.x - b.x).map((p) => p.id);
}

// neighborPanel(panels, fromId, dir) -> id | "": where an arrow key lands.
// The nearest panel in that direction that shares rows (left, right) or
// columns (up, down) with the focused one; ties go to the one whose near
// edge lines up best, then to reading order. Up and down fall back to the
// nearest panel in that half of the grid, so a row holding one panel is
// never a dead end; left and right stay on the row. Unknown id or
// direction: "".
export function neighborPanel(panels, fromId, dir) {
  const rects = placed(panels);
  const from = rects.find((p) => p.id === fromId);
  if (!from || !PANEL_DIRECTIONS.includes(dir)) return "";
  const rowsShared = (c) => c.y < from.y + from.h && from.y < c.y + c.h;
  const colsShared = (c) => c.x < from.x + from.w && from.x < c.x + c.w;
  const rule = {
    right: { ahead: (c) => c.x >= from.x + from.w, gap: (c) => c.x - (from.x + from.w), along: (c) => Math.abs(c.y - from.y), shares: rowsShared, fallback: false },
    left: { ahead: (c) => c.x + c.w <= from.x, gap: (c) => from.x - (c.x + c.w), along: (c) => Math.abs(c.y - from.y), shares: rowsShared, fallback: false },
    down: { ahead: (c) => c.y >= from.y + from.h, gap: (c) => c.y - (from.y + from.h), along: (c) => Math.abs(c.x - from.x), shares: colsShared, fallback: true },
    up: { ahead: (c) => c.y + c.h <= from.y, gap: (c) => from.y - (c.y + c.h), along: (c) => Math.abs(c.x - from.x), shares: colsShared, fallback: true },
  }[dir];
  const ahead = rects.filter((c) => c.id !== from.id && rule.ahead(c));
  const shared = ahead.filter(rule.shares);
  const pool = shared.length ? shared : rule.fallback ? ahead : [];
  let best = null;
  for (const c of pool) {
    if (!best) { best = c; continue; }
    const d = rule.gap(c) - rule.gap(best) || rule.along(c) - rule.along(best) || c.y - best.y || c.x - best.x;
    if (d < 0) best = c;
  }
  return best ? best.id : "";
}
