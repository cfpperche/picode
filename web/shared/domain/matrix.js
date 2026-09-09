// Matrix (ADR-0108): the client side of the store's contract. Pure — no
// React, no fetch. normalize* turn the API's JSON into the shapes the
// surface holds (junk dropped, the way contracts/appPrimitives.js does);
// validate* refuse what the server would refuse, in the server's words, so
// the UI never asks a question it knows the answer to; applyMatrixEvent
// reduces the six feed events over { list: [summaries], byId: { id:
// { matrix, panels } } }. touches(ev, ["matrix"]) in feedReducers.js keys
// on the prefix before the first dot, so matrix.panel.* reaches the
// surface with the rest.

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
