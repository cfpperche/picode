// Canvas (ADR-0108, ADR-0116, ADR-0118): the client side of the store's
// contract. Pure — no React, no fetch. normalize* turn the API's JSON into the
// shapes the surface holds (junk dropped, the way contracts/appPrimitives.js
// does); validate* refuse what the server would refuse, in the server's words,
// so the UI never asks a question it knows the answer to; applyCanvasEvent
// reduces the eight feed events over { list: [summaries], byId: { id:
// { canvas, panels, edges } } }. touches(ev, ["canvas"]) in feedReducers.js
// keys on the prefix before the first dot, so canvas.panel.* reaches the
// surface with the rest. The surface's arithmetic lives here too (plan
// docs/plans/matrix-app.md §4.3–§4.6): nextSlot, layoutDiff, bindingState,
// loadPolicy, the suspended-instance LRU and the keyboard's neighbour
// arithmetic — decisions the components only carry out.
//
// There is one layout engine and one set of rules, the plane
// (docs/architecture/canvas.md): ADR-0118 removed the second one, so nothing
// here takes a mode and no rectangle is ever judged in columns.

import { locate } from "./tree.js";

export const CANVAS_LIMITS = Object.freeze({
  canvases: 64, // per machine
  panels: 500, // per canvas
  name: 80, // characters (code points), never bytes
  // A text panel is a label on a plane, not a document — the server's own
  // MaxCanvasText, so the box can stop a reader before the refusal does.
  text: 2000,
  // The plane (ADR-0113, ADR-0118): the unit is 8 px, x and y may be
  // negative, and the plane is bounded only so a panel cannot be lost.
  canvasMinW: 32, // units (256 px)
  canvasMinH: 28, // units (224 px)
  canvasMax: 4096, // units, w and h
  canvasCoord: 100000, // units, |x| and |y|
  // Edges (ADR-0116): a link is cheaper than a panel and a dense board
  // draws more of them than it holds panels, so the cap is twice the
  // panel cap.
  edges: 1000, // per canvas
});

// One canvas unit in CSS pixels — the plane's own 8 px step, so snapGrid
// falls out of the coordinate system (plan §4.1).
export const UNIT_PX = 8;

// The factors migration 045 converted with, once: a cell of the engine that
// is gone was 8 canvas units wide (colWidth / 8) and 3 tall (rowHeight
// 24 px / 8).
const CANVAS_PER_COL = 8;
const CANVAS_PER_ROW = 3;

// What a panel can be bound to (ADR-0108; the bodies beyond a live pane are
// C3 of docs/plans/matrix-canvas.md §4.2). `kind` is an open text column, so
// a kind is a validator edit on both sides and never a migration.
// "text" is the first kind that references nothing: the words are the panel
// (migration 046). Everything else names something that lives elsewhere in
// PiCode and would still exist if the canvas were deleted.
export const CANVAS_KINDS = Object.freeze(["agent", "terminal", "note", "file", "diff", "text"]);
export const CANVAS_COMPACT = Object.freeze(["vertical", "none"]);
export const CANVAS_EVENTS = Object.freeze([
  "canvas.created",
  "canvas.updated",
  "canvas.layout",
  "canvas.panel.added",
  "canvas.panel.content",
  "canvas.panel.removed",
  "canvas.edge.added",
  "canvas.edge.removed",
  "canvas.deleted",
]);

// The server's refusals, word for word (internal/store/canvas.go), so the UI
// can refuse before asking and read a 400 back as the same sentence.
const KIND_MSG = "kind must be agent, terminal, note, file, diff or text";
const REF_MSG = "ref must be <owner>:<id>:<path> with owner t, a or w";
const TEXT_REF_MSG = "a text panel takes no ref";
// Edges (internal/store/canvas_edges.go), same rule.
const EDGE_BOTH_MSG = "aPanel and bPanel are required";
const EDGE_SAME_MSG = "an edge needs two different panels";
const EDGE_DUP_MSG = "These panels are already linked";
const EDGE_KINDS_MSG = "an edge links agent or terminal panels";
// The kinds that have a mailbox at all — a note, a file or a diff is not a
// session, so it cannot hold ADR-0104's contact.
export const EDGE_KINDS = Object.freeze(["agent", "terminal"]);

const str = (v) => (typeof v === "string" ? v : "");
const nonEmpty = (v) => typeof v === "string" && v !== "";
const runes = (s) => Array.from(s).length; // code points, the way Go counts runes

// normalizeCanvas(json) -> summary | null. The summary is what the list,
// the feed and a PATCH answer carry: never panels.
export function normalizeCanvas(m) {
  if (!m || typeof m !== "object" || !nonEmpty(m.id) || !nonEmpty(m.name)) return null;
  return {
    id: m.id,
    name: m.name,
    compact: CANVAS_COMPACT.includes(m.compact) ? m.compact : "vertical",
    createdAt: str(m.createdAt),
    updatedAt: str(m.updatedAt),
    panelCount: Number.isInteger(m.panelCount) && m.panelCount >= 0 ? m.panelCount : 0,
  };
}

// ---- refs (docs/plans/matrix-canvas.md §4.2) ------------------------------
//
// What a `ref` is depends on the kind, and this is the only place that takes
// one apart or puts one together — never an inline split.
//
//   agent, terminal  the agent's or terminal's id
//   note             a pin id
//   file, diff       "<owner letter>:<owner id>:<path>"
//
// The owner letters are the desktop's own (web/desktop/src/lib/routes.js,
// ADR-0030): t terminal, a agent, w workspace. A path may hold colons of its
// own, so only the first two separate.

export const REF_OWNERS = Object.freeze({ t: "term", a: "agent", w: "workspace" });
const REF_LETTERS = Object.freeze({ term: "t", agent: "a", workspace: "w" });
const OWNED_KINDS = Object.freeze(["file", "diff"]);

// buildRef(kind, parts) -> the ref string that kind takes, or "" when the
// parts cannot make one. parts is { pinId } or { owner: { kind, id }, path }.
export function buildRef(kind, parts) {
  const p = parts || {};
  if (!OWNED_KINDS.includes(kind)) return str(p.pinId || p.id).trim();
  const letter = REF_LETTERS[p.owner && p.owner.kind];
  const id = str(p.owner && p.owner.id).trim();
  const path = str(p.path).trim();
  return letter && id && path && !id.includes(":") ? letter + ":" + id + ":" + path : "";
}

// parseRef(kind, ref) -> the binding's parts, or null when the ref is not the
// shape that kind takes:
//   note        { kind, pinId }
//   file, diff  { kind, letter, owner: { kind, id }, path }
export function parseRef(kind, ref) {
  const r = str(ref).trim();
  if (!r) return null;
  if (!OWNED_KINDS.includes(kind)) return { kind, pinId: r };
  const first = r.indexOf(":");
  const second = first < 0 ? -1 : r.indexOf(":", first + 1);
  if (first <= 0 || second < 0) return null;
  const letter = r.slice(0, first);
  const id = r.slice(first + 1, second);
  const path = r.slice(second + 1);
  if (!REF_OWNERS[letter] || !id || !path) return null;
  return { kind, letter, owner: { kind: REF_OWNERS[letter], id }, path };
}

// validateRef(kind, ref) -> "" | the server's refusal, in the server's words
// (internal/store/canvas.go).
export function validateRef(kind, ref) {
  if (!str(ref).trim()) return "ref is required";
  return parseRef(kind, ref) ? "" : REF_MSG;
}

// normalizeCanvasList(GET /api/canvases) -> summaries.
export function normalizeCanvasList(payload) {
  const list = Array.isArray(payload?.canvases) ? payload.canvases : [];
  return list.map(normalizeCanvas).filter(Boolean);
}

// normalizePanel(json) -> panel | null. A panel needs a slot id, a known
// binding and a whole rectangle; anything else cannot be placed — including
// a ref that is not the shape its kind takes, which the server refuses too.
export function normalizePanel(p) {
  if (!p || typeof p !== "object" || !nonEmpty(p.id) || !CANVAS_KINDS.includes(p.kind)) return null;
  const ref = str(p.ref).trim();
  if (!ref || validateRef(p.kind, ref)) return null;
  const { x, y, w, h } = p;
  if (![x, y, w, h].every(Number.isInteger)) return null;
  // Only a text panel carries words; the field is absent on every other
  // kind and an empty string is the honest default for a panel just born.
  return { id: p.id, kind: p.kind, ref, x, y, w, h, createdAt: str(p.createdAt), content: str(p.content) };
}

// normalizeEdge(json) -> edge | null. An edge (ADR-0116) is an undirected
// link between two panels of one canvas; the server stores the pair ordered
// (aPanel < bPanel) so the same edge drawn either way is one row. Whether
// the two panels are still on the board is edgeEndpoints' answer, not this
// one — a normalizer drops junk, never data.
export function normalizeEdge(e) {
  if (!e || typeof e !== "object" || !nonEmpty(e.id) || !nonEmpty(e.aPanel) || !nonEmpty(e.bPanel)) return null;
  if (e.aPanel === e.bPanel) return null;
  return { id: e.id, aPanel: e.aPanel, bPanel: e.bPanel, createdAt: str(e.createdAt) };
}

// normalizeEdgeList(GET /api/canvases/{id}/edges) -> edges.
export function normalizeEdgeList(payload) {
  const list = Array.isArray(payload?.edges) ? payload.edges : [];
  return list.map(normalizeEdge).filter(Boolean);
}

// normalizeCanvasDetail(GET /api/canvases/{id}) -> { canvas, panels, edges }
// | null. The count follows the panels that survived normalization; a
// payload from before ADR-0116 carries no edges and reads as none.
export function normalizeCanvasDetail(d) {
  const m = normalizeCanvas(d);
  if (!m) return null;
  const panels = (Array.isArray(d.panels) ? d.panels : []).map(normalizePanel).filter(Boolean);
  return { canvas: { ...m, panelCount: panels.length }, panels, edges: normalizeEdgeList(d) };
}

// validateName(name) -> "" | the server's refusal.
export function validateName(name) {
  const n = str(name).trim();
  if (!n) return "name is required";
  if (runes(n) > CANVAS_LIMITS.name) return `name is too long (max ${CANVAS_LIMITS.name} characters)`;
  return "";
}

// validateCompact(compact) -> "" | the server's refusal.
export function validateCompact(compact) {
  return CANVAS_COMPACT.includes(compact) ? "" : `compact must be ${CANVAS_COMPACT.join(" or ")}`;
}

// validatePlacement({x, y, w, h}) -> "" | the server's refusal (ADR-0113,
// ADR-0118): the plane is 8 px units where x and y may be negative, a panel
// is at least 32×28 units, and both are bounded so a panel cannot be lost.
// There is one set of rules and no mode to judge by. The "whole number"
// line is the client's own — the server answers a fractional coordinate
// with "invalid JSON body".
export function validatePlacement(rect) {
  const { x, y, w, h } = rect || {};
  for (const [k, v] of [["x", x], ["y", y], ["w", w], ["h", h]]) {
    if (!Number.isInteger(v)) return `${k} must be a whole number`;
  }
  const { canvasMinW, canvasMinH, canvasMax, canvasCoord } = CANVAS_LIMITS;
  if (x < -canvasCoord || x > canvasCoord) return `x must be between -${canvasCoord} and ${canvasCoord} canvas units`;
  if (y < -canvasCoord || y > canvasCoord) return `y must be between -${canvasCoord} and ${canvasCoord} canvas units`;
  if (w < canvasMinW) return `w must be at least ${canvasMinW} canvas units`;
  if (h < canvasMinH) return `h must be at least ${canvasMinH} canvas units`;
  if (w > canvasMax) return `w must be at most ${canvasMax} canvas units`;
  if (h > canvasMax) return `h must be at most ${canvasMax} canvas units`;
  return "";
}

// panelById(panels, id) -> the panel or undefined.
const panelById = (panels, id) => (Array.isArray(panels) ? panels.find((p) => p && p.id === id) : undefined);

// validateEdge({aPanel, bPanel}, panels, edges) -> "" | the server's
// refusal, in the server's words and the server's order
// (internal/store/canvas_edges.go): both ids, two different panels, both on
// this canvas, both a kind that has a mailbox, then the cap and the pair
// that is already linked. `panels` and `edges` are what the client already
// holds for that canvas; leave them out and only the two id rules are
// checked, because the rest cannot be judged without them.
//
// What an edge grants is ADR-0104's mailbox contact and nothing else — no
// transcript, ever. The UI must never say otherwise.
export function validateEdge(edge, panels, edges) {
  const a = str(edge && edge.aPanel).trim();
  const b = str(edge && edge.bPanel).trim();
  if (!a || !b) return EDGE_BOTH_MSG;
  if (a === b) return EDGE_SAME_MSG;
  if (!Array.isArray(panels)) return "";
  for (const id of [a, b]) {
    const panel = panelById(panels, id);
    if (!panel) return `panel ${id} is not on this canvas`;
    if (!EDGE_KINDS.includes(panel.kind)) return `panel ${id} is a ${panel.kind} panel and has no mailbox: ${EDGE_KINDS_MSG}`;
  }
  if (!Array.isArray(edges)) return "";
  if (edges.length >= CANVAS_LIMITS.edges) return `limit: ${CANVAS_LIMITS.edges} edges per canvas`;
  return edges.some((e) => linksSamePair(e, a, b)) ? EDGE_DUP_MSG : "";
}

// linksSamePair(edge, a, b) -> is this the same undirected link? The server
// stores the pair ordered, so a client that sends it backwards gets the
// 409; this is how the UI refuses first.
function linksSamePair(e, a, b) {
  return !!e && ((e.aPanel === a && e.bPanel === b) || (e.aPanel === b && e.bPanel === a));
}

// edgeEndpoints(edge, panels) -> { a, b } | null: the two panels an edge
// joins, in the edge's stored order, or null when either end is not on the
// board. The canvas draws from these two rectangles, and null is exactly
// the case it must not draw — an endpoint the client has not got.
export function edgeEndpoints(edge, panels) {
  const e = normalizeEdge(edge);
  if (!e) return null;
  const a = panelById(panels, e.aPanel);
  const b = panelById(panels, e.bPanel);
  return a && b ? { a, b } : null;
}

// validatePanel({kind, ref, x, y, w, h}) -> "" | the server's refusal, in
// the server's order: binding first, then the rectangle.
export function validatePanel(panel) {
  const p = panel || {};
  if (!CANVAS_KINDS.includes(p.kind)) return KIND_MSG;
  // A text panel binds to nothing and the store mints its ref, so a caller
  // that sends one is describing something that does not exist — the
  // server's own refusal, word for word.
  if (p.kind === "text") {
    if (str(p.ref).trim()) return TEXT_REF_MSG;
    return validatePlacement(p);
  }
  const bad = validateRef(p.kind, p.ref);
  if (bad) return bad;
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

// movePanels applies a subset of rectangles (a layout frame) to the panels a
// canvas holds, dropping rows the plane's rules refuse.
function movePanels(panels, subset) {
  const to = new Map();
  for (const p of Array.isArray(subset) ? subset : []) {
    if (p && nonEmpty(p.id) && !validatePlacement(p)) to.set(p.id, { x: p.x, y: p.y, w: p.w, h: p.h });
  }
  return panels.map((p) => (to.has(p.id) ? { ...p, ...to.get(p.id) } : p));
}

// edgesOf(entry) -> the edges a loaded canvas holds. A byId entry is
// { canvas, panels, edges }; `edges` arrived with ADR-0116, so an entry
// built before it reads as none.
const edgesOf = (loaded) => (loaded && Array.isArray(loaded.edges) ? loaded.edges : []);

// applyCanvasEvent(state, ev) -> next | same. state is { list, byId };
// an event for a canvas that is not loaded (no byId entry) only touches
// the summary list. Unknown types and junk return the same object. Every
// rectangle is judged by the one set of rules the plane has (ADR-0118).
// The two edge events carry ids only (ADR-0048), and canvas.panel.removed
// drops the edges that touched the panel — the store cascades them in the
// same transaction and announces no event each.
export function applyCanvasEvent(state, ev) {
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
    case "canvas.created":
    case "canvas.updated": {
      const m = normalizeCanvas(d);
      if (!m) return s;
      const nextById = loaded ? { ...byId, [id]: { canvas: { ...m, panelCount: loaded.panels.length }, panels: loaded.panels, edges: edgesOf(loaded) } } : byId;
      return finish(s, upsertSummary(list, m), nextById);
    }
    case "canvas.layout": {
      const nextList = patchSummary(list, id, stamp);
      if (!loaded) return finish(s, nextList, byId);
      const panels = movePanels(loaded.panels, d.panels);
      return finish(s, nextList, { ...byId, [id]: { canvas: stamp(loaded.canvas), panels, edges: edgesOf(loaded) } });
    }
    case "canvas.panel.added": {
      const panel = normalizePanel(d.panel);
      if (!panel) return s;
      if (!loaded) return finish(s, patchSummary(list, id, (m) => ({ ...stamp(m), panelCount: m.panelCount + 1 })), byId);
      const panels = loaded.panels.some((p) => p.id === panel.id)
        ? loaded.panels.map((p) => (p.id === panel.id ? panel : p))
        : [...loaded.panels, panel];
      const count = (m) => ({ ...stamp(m), panelCount: panels.length });
      return finish(s, patchSummary(list, id, count), { ...byId, [id]: { canvas: count(loaded.canvas), panels, edges: edgesOf(loaded) } });
    }
    case "canvas.panel.content": {
      // The words as the store now holds them, not a patch: a second browser
      // draws the string that was saved rather than replaying keystrokes.
      const panelId = str(d.panelId);
      if (!panelId || !loaded) return loaded ? s : finish(s, patchSummary(list, id, stamp), byId);
      const content = str(d.content);
      const panels = loaded.panels.map((p) => (p.id === panelId ? { ...p, content } : p));
      return finish(s, patchSummary(list, id, stamp), { ...byId, [id]: { canvas: stamp(loaded.canvas), panels, edges: edgesOf(loaded) } });
    }
    case "canvas.panel.removed": {
      const panelId = str(d.panelId);
      if (!panelId) return s;
      if (!loaded) return finish(s, patchSummary(list, id, (m) => ({ ...stamp(m), panelCount: Math.max(0, m.panelCount - 1) })), byId);
      const panels = loaded.panels.filter((p) => p.id !== panelId);
      // The store's cascade, mirrored: an edge to a panel that is gone
      // grants nothing and must not be drawn.
      const edges = edgesOf(loaded).filter((e) => e.aPanel !== panelId && e.bPanel !== panelId);
      const count = (m) => ({ ...stamp(m), panelCount: panels.length });
      return finish(s, patchSummary(list, id, count), { ...byId, [id]: { canvas: count(loaded.canvas), panels, edges } });
    }
    case "canvas.edge.added": {
      const edge = normalizeEdge(d.edge);
      if (!edge) return s;
      const nextList = patchSummary(list, id, stamp);
      if (!loaded) return finish(s, nextList, byId);
      const held = edgesOf(loaded);
      const edges = held.some((e) => e.id === edge.id) ? held.map((e) => (e.id === edge.id ? edge : e)) : [...held, edge];
      return finish(s, nextList, { ...byId, [id]: { canvas: stamp(loaded.canvas), panels: loaded.panels, edges } });
    }
    case "canvas.edge.removed": {
      const edgeId = str(d.edgeId);
      if (!edgeId) return s;
      const nextList = patchSummary(list, id, stamp);
      if (!loaded) return finish(s, nextList, byId);
      const edges = edgesOf(loaded).filter((e) => e.id !== edgeId);
      return finish(s, nextList, { ...byId, [id]: { canvas: stamp(loaded.canvas), panels: loaded.panels, edges } });
    }
    case "canvas.deleted": {
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

// A new panel is 32×42 units — 256×336 px, ≈ 58×20 characters at 1584 px
// (phase 0 study). It is the only default there is.
export const PANEL_DEFAULT_CANVAS = Object.freeze({ w: 32, h: 42 });
// Chunk loading (§4.5): a body mounts after its wrapper has been near the
// viewport for 300 ms and unmounts 5 s after it left; a suspended xterm is
// disposed past 24 instances or 10 minutes.
export const LOAD_DWELL_MS = 300;
export const UNLOAD_AFTER_MS = 5000;
export const SUSPENDED_MAX = 24;
export const SUSPENDED_TTL_MS = 10 * 60 * 1000;

const wholeRect = (p) => !!p && [p.x, p.y, p.w, p.h].every(Number.isInteger);
const overlaps = (a, b) => a.x < b.x + b.w && b.x < a.x + a.w && a.y < b.y + b.h && b.y < a.y + a.h;

// nextSlot(panels, w, h) -> {x, y}: the first free w×h slot scanning rows
// top-down and left-to-right, else the row under everything. Only a panel's
// edges can start a first-free slot (a free rectangle slides up and left
// until it touches one), so the scan visits edges, not cells. The plane has
// no column cap, so a new panel lands beside the others instead of under
// them — nothing wraps.
export function nextSlot(panels, w = PANEL_DEFAULT_CANVAS.w, h = PANEL_DEFAULT_CANVAS.h) {
  const rects = (panels || []).filter(wholeRect);
  const width = Math.max(1, w | 0);
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
  const columns = [...xs].sort((a, b) => a - b);
  for (const y of rows) {
    for (const x of columns) {
      const cand = { x, y, w: width, h: height };
      if (!rects.some((r) => overlaps(cand, r))) return { x, y };
    }
  }
  return { x: 0, y: bottom };
}

// ---- placing a panel by hand (2026-09-12) --------------------------------
//
// Until now the surface chose where a new panel went: `nextSlot` packs from
// the canvas origin outward. On an infinite plane the origin means nothing
// to the reader — with the camera parked anywhere else, a new panel is born
// off screen, which is exactly what the owner hit. So the reader draws the
// rectangle instead, and this turns that gesture into stored units.
//
// `from` and `to` are plane pixels (React Flow's coordinates, not the
// screen's). A gesture that travelled less than `slop` pixels on both axes
// is a **click**: nobody drew a rectangle, so the default size is centred on
// the point. Anything longer is the rectangle drawn, snapped to whole units,
// grown to the panel minimum if it was drawn smaller, and clamped to the
// plane. The minimum grows from the top-left corner, because that is the
// corner the reader anchored first.
export function placementRect(from, to, opts = {}) {
  const size = opts.size || PANEL_DEFAULT_CANVAS;
  const slop = Number.isFinite(opts.slop) ? opts.slop : 6;
  const { canvasMinW, canvasMinH, canvasMax, canvasCoord } = CANVAS_LIMITS;
  const ax = Number.isFinite(from && from.x) ? from.x : 0;
  const ay = Number.isFinite(from && from.y) ? from.y : 0;
  const bx = Number.isFinite(to && to.x) ? to.x : ax;
  const by = Number.isFinite(to && to.y) ? to.y : ay;

  let x;
  let y;
  let w;
  let h;
  if (Math.abs(bx - ax) < slop && Math.abs(by - ay) < slop) {
    w = Math.max(canvasMinW, size.w | 0);
    h = Math.max(canvasMinH, size.h | 0);
    x = round(ax / UNIT_PX) - Math.round(w / 2);
    y = round(ay / UNIT_PX) - Math.round(h / 2);
  } else {
    const x1 = round(Math.min(ax, bx) / UNIT_PX);
    const y1 = round(Math.min(ay, by) / UNIT_PX);
    const x2 = round(Math.max(ax, bx) / UNIT_PX);
    const y2 = round(Math.max(ay, by) / UNIT_PX);
    x = x1;
    y = y1;
    w = Math.max(canvasMinW, x2 - x1);
    h = Math.max(canvasMinH, y2 - y1);
  }
  w = clamp(w, canvasMinW, canvasMax);
  h = clamp(h, canvasMinH, canvasMax);
  // The whole rectangle stays on the plane: a panel dropped at the very edge
  // is pulled back by its own width rather than clipped or refused.
  x = clamp(x, -canvasCoord, canvasCoord - w);
  y = clamp(y, -canvasCoord, canvasCoord - h);
  return { x, y, w, h };
}

// ---- what migration 045 did, once (ADR-0113, ADR-0118) -------------------
//
// gridToCanvas is the transform the migration applied when it rewrote every
// old board's rectangles into canvas units — x*8, y*3, w*8, h*3, clamped to
// the canvas minimums — in the same transaction that dropped the mode
// column. It is kept here as the documented record of that conversion and
// has no caller. The reverse direction is gone: going back would mean
// restoring the column and packing panels into twelve columns again, and
// that is not supported.

const clamp = (v, lo, hi) => (v < lo ? lo : v > hi ? hi : v);
// Round half away from zero, the way Go's math.Round does, so a negative
// coordinate lands on the same cell on both sides.
const round = (v) => (v < 0 ? -Math.round(-v) : Math.round(v));

// gridToCanvas(panels): a cell was 8 units wide and 3 tall. A panel at the
// old 8-row minimum becomes 24 units tall, under the canvas minimum of 28,
// so it grew — and could then overlap the panel below by up to 4 units. The
// plane is free placement and allows that, which is why the migration had
// nothing to pack.
export function gridToCanvas(panels) {
  const { canvasMinW, canvasMinH, canvasMax, canvasCoord } = CANVAS_LIMITS;
  return (panels || []).filter(wholeRect).map((p) => ({
    ...p,
    x: clamp(p.x * CANVAS_PER_COL, -canvasCoord, canvasCoord),
    y: clamp(p.y * CANVAS_PER_ROW, -canvasCoord, canvasCoord),
    w: clamp(p.w * CANVAS_PER_COL, canvasMinW, canvasMax),
    h: clamp(p.h * CANVAS_PER_ROW, canvasMinH, canvasMax),
  }));
}

// layoutDiff(prev, next) -> [{id, x, y, w, h}]: the panels whose rectangle
// changed between two layouts, which is exactly what PATCH …/layout takes.
// A panel only in `next` is an add (its own POST), only in `prev` a remove;
// a rectangle the plane refuses is left out so one bad row cannot sink the
// batch.
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

// gitTouches(root, evPath) -> does a `git.updated` for `evPath` concern a
// file read through a folder at `root`? The fleet's git watcher names the
// folder it inspected, and an owner's folder is that one, one inside it, or
// one it is inside — so both directions count. It is a superset on purpose:
// an extra refetch of one diff is cheaper than a diff that goes stale.
export function gitTouches(root, evPath) {
  const a = str(root).replace(/\/+$/, "");
  const b = str(evPath).replace(/\/+$/, "");
  if (!a || !b) return false;
  return a === b || a.startsWith(b + "/") || b.startsWith(a + "/");
}

// refOwner(parsed, fleet) -> the fleet row a file's ref names, or null. The
// three owner kinds are the three the file APIs take (ADR-0030).
export function refOwner(parsed, fleet) {
  const f = fleet || {};
  if (!parsed || !parsed.owner || !parsed.owner.id) return null;
  const { kind, id } = parsed.owner;
  if (kind === "term") return (f.terminals || []).find((t) => t && t.id === id) || null;
  if (kind === "workspace") return (f.workspaces || []).find((w) => w && w.id === id) || null;
  const loc = locate(f.workspaces, f.freeAgents, id);
  return loc && loc.agent && loc.agent.id === id ? loc.agent : null;
}

// bindingState(panel, fleet) -> one row of plan §4.4:
//   terminal-running | terminal-stopped | terminal-gone
//   agent-interactive | agent-managed | agent-stopped | agent-gone
//   note-ready | note-gone | text-ready
//   file-ready | file-gone | diff-ready | diff-gone
// fleet is the host's { workspaces, freeAgents, terminals } plus, for the
// kinds that bind something else, the list that decides them: `pins` for a
// note; a file's owner is the fleet's own. A shell whose tmux session died is still "running" for the panel:
// opening it revives the shell; only a configured CLI terminal has a stopped
// state of its own.
//
// A list that has not been read yet is `null`, not `[]` — the same rule the
// fleet follows (`host.fleet.loaded`): before the read, nothing is gone.
export function bindingState(panel, fleet) {
  const f = fleet || {};
  if (!panel) return "";
  // A text panel binds to nothing, so nothing about it can go missing. It is
  // the one kind whose state does not depend on the fleet at all.
  if (panel.kind === "text") return "text-ready";
  if (panel.kind === "note") {
    if (!Array.isArray(f.pins)) return "note-ready";
    return f.pins.some((x) => x && x.id === panel.ref) ? "note-ready" : "note-gone";
  }
  if (panel.kind === "file" || panel.kind === "diff") {
    // The **owner** is what can be gone — the terminal, agent or folder the
    // file is read through. A file that is missing on disk, or a path with
    // no changes, is not a binding state: the body reports that, the way
    // FilePane reports any read failure, and the panel keeps its actions.
    return refOwner(parseRef(panel.kind, panel.ref), f) ? panel.kind + "-ready" : panel.kind + "-gone";
  }
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

// The rows whose body *is* a terminal — the only ones the zoom's pointer and
// still rules bind, because those rules are about a cell (matrix-canvas.md
// §4.3). Everything else (the four rows that answer with one line and one
// action, and every body that is not a pane at all: a note, a file, a diff)
// keeps its own chrome at every zoom above the name-plate.
export const PANE_STATES = Object.freeze(["terminal-running", "terminal-stopped", "agent-interactive"]);

// hasPane(model) -> does this panel hold an xterm? PanelBody re-exports it,
// so the components ask the domain and never a list of their own.
export function hasPane(model) {
  return !!model && !model.pending && PANE_STATES.includes(model.state);
}

// ---- what a body costs (phase 4, docs/plans/matrix-app.md §2.4/§4.5) ------
//
// There are three costs, not two. A **pane** is an xterm plus a tmux attach,
// and it is the only body with a cell, so it is the only one the zoom's
// pointer and still rules bind. A note, a file or a diff is one fetch and
// some DOM: free to leave mounted, which is why today's rule for them is
// "mount in band, unmount outside, no still". A **chat** — a managed agent's
// live conversation — is neither: it has no cell, so it never goes still,
// but it holds a WebSocket and a transcript window, so leaving it mounted
// where it cannot be read is waste. It takes the middle path, and this
// allow-list is what says which rows are on it.
export const CHAT_STATES = Object.freeze(["agent-managed"]);

// hasChat(model) -> does this panel's body hold an agent socket?
export function hasChat(model) {
  return !!model && !model.pending && CHAT_STATES.includes(model.state);
}

// How many chat sockets one canvas may hold at once. The band already
// bounds the panes at what fits in three viewport heights (§4.5: ≈ 18 on a
// 1440p screen with default-sized panels, measured 12–21 at rest), and a
// chat panel joins that budget — but a chat costs more than an attach on
// arrival: one WebSocket *and* one `…/sessions/transcript?tail=200` read per
// connect. Twelve is above what a band of default-sized panels holds on a
// 1440p screen, so the cap only bites on a canvas denser than a screenful of
// managed agents; measured on a 20-agent canvas in phase 4's QA.
export const CHAT_LIVE_MAX = 12;

// chatBudget(live, max) -> { keep, park }: which chat bodies hold a socket
// when more are in the band than the cap allows. `live` is
// [{ id, at }] — `at` is when that panel entered the band — and the **most
// recently** arrived keep their sockets, because that is where the reader
// just scrolled; the rest are parked (mounted, header live, body one muted
// line, no socket). The order is stable while a panel stays in the band, so
// nothing thrashes, and a panel that leaves frees its slot through the same
// 5 s hysteresis every other body has. The LRU shape is
// `suspendedToDispose`'s, applied to sockets instead of xterm instances.
export function chatBudget(live, max = CHAT_LIVE_MAX) {
  const list = (live || []).filter((e) => e && nonEmpty(e.id) && Number.isFinite(e.at));
  // Newest first; ties by id so two panels that arrived in the same
  // millisecond do not swap places from one tick to the next.
  list.sort((a, b) => (b.at - a.at) || (a.id < b.id ? -1 : a.id > b.id ? 1 : 0));
  const cap = Math.max(0, max);
  return { keep: list.slice(0, cap).map((e) => e.id), park: list.slice(cap).map((e) => e.id) };
}

// ---- zoom (plan docs/plans/matrix-canvas.md §4.3) -------------------------
//
// The rule that makes a big canvas cheap, and the one correctness rule the
// C0 spike found (docs/benchmarks/2026-09-10-node-canvas.md): **xterm maps a
// pointer as `cell × zoom` under a CSS transform**, because it divides the
// offset inside the transformed rect by the untransformed cell size. At 0.8
// a click aimed at column 50 row 15 arrives as column 40 row 12, and PiCode
// ships tmux `mouse on`, so that wrong coordinate reaches copy mode and
// every mouse-aware TUI. Nothing on screen shows it — the DOM renderer stays
// crisp — so the surface has to enforce it: a live pane takes a pointer only
// at zoom exactly 1.0, and a click below it snaps to 1 first.

export const CANVAS_ZOOM = Object.freeze({
  min: 0.2, // React Flow's minZoom: orientation, not reading
  max: 1.5, // maxZoom: zooming in breaks the pointer mapping exactly as out does
  live: 0.8, // at or above, a body is the live pane…
  still: 0.75, // …and below this it is a text still. The gap is the hysteresis
  plate: 0.4, // below this a still is grey texture (C0): a name-plate instead
  exact: 1, // the only zoom whose pointer lands on the cell it points at
});

// Zoom arrives from a wheel and a d3 transform, so "exactly 1.0" is a
// tolerance, not an equality.
export const ZOOM_EPS = 0.005;

// zoomBody(zoom, band, pane) -> { band, body }: what a *loaded* body renders
// and the hysteretic band to feed back next time. Live at or above 0.8,
// still below 0.75, and in between whatever it already was — so a viewer
// parked on the boundary does not thrash nine sockets (C0: crossing
// 0.79/0.80 suspended and kicked nine attaches). Below 0.4 it is a
// name-plate whatever the band says.
//
// `pane` is whether this body is a terminal (hasPane). The still is a
// picture of `term.buffer.active`, so a body with no buffer has none to
// show: a note, a file or a diff renders normally from 0.4 up — DOM text
// scales, and the pointer bug is xterm's alone — and becomes the name-plate
// below 0.4, where nothing textual is legible. The band it answers with is
// the pane band either way, because the band is what bounds the attaches.
export function zoomBody(zoom, band, pane = true) {
  const z = Number.isFinite(zoom) ? zoom : CANVAS_ZOOM.exact;
  let next = band === "still" ? "still" : "live";
  if (z >= CANVAS_ZOOM.live) next = "live";
  else if (z < CANVAS_ZOOM.still) next = "still";
  if (z < CANVAS_ZOOM.plate) return { band: next, body: "plate" };
  return { band: next, body: pane && next === "still" ? "still" : "live" };
}

// pointerAtZoom(zoom) -> may the pointer reach the pane? Only at 1.0.
export function pointerAtZoom(zoom) {
  return Math.abs((Number.isFinite(zoom) ? zoom : CANVAS_ZOOM.exact) - CANVAS_ZOOM.exact) < ZOOM_EPS;
}

// loadPolicy(entries, now, pinned, timers, view)
//   -> { load, unload, timers, wakeAt, band, bodies }
// The chunk-loading decision (§4.5), pure so every row has a test:
//   entries  [{ id, near, loaded, pane, chat }] — near: inside the viewport ±
//            the observer's margin (its word) and the surface visible; pane:
//            whether this body is a terminal (default true, so a caller that
//            says nothing gets the pane rules); chat: whether it holds an
//            agent socket (hasChat), which is what the cap counts
//   pinned   Set of ids that never unload (dragged, resized, focused,
//            maximized)
//   timers   the previous call's `timers` — the policy's only memory:
//            { id: { loadAt } | { unloadAt } }
//   view     { zoom, band } — the canvas's zoom and the band it was last in.
//            A caller that passes nothing gets zoom 1: every loaded body is
//            live.
// A wrapper loads once it has been near for LOAD_DWELL_MS; leaving cancels
// a pending load at once. A loaded body unloads UNLOAD_AFTER_MS after it
// left; coming back cancels the unload. wakeAt is the earliest deadline,
// 0 when nothing is pending — the caller sleeps until then.
// `bodies` is what each entry renders now: off (not loaded — the
// placeholder, and the attach is suspended), live, still, plate or quiet —
// the still only for an entry that has a pane, the quiet only for a chat
// over the cap. The band is screen-space, so zooming out puts everything in
// it (C0: 500 of 500 at 0.2); what bounds the attaches down there is the
// still rule for a pane and the plate for a chat, not the band.
//
// A chat body's rule, in one place (§4.5, phase 4): **its socket is open
// exactly while its body reads `live`** — in the band and at zoom ≥ 0.4,
// under the cap. Outside the band it unloads with everything else after the
// 5 s hysteresis; below 0.4 it is a name-plate, and a plate cannot show a
// conversation, so holding a socket for one is waste; over the cap it is
// quiet. Nothing about the panel's **header** is decided here — the chip is
// fed by the fleet and the feed, which is what lets a zoomed-out canvas
// still say which agent is blocked.
export function loadPolicy(entries, now, pinned, timers, view) {
  const pin = pinned || new Set();
  const prev = timers || {};
  const next = {};
  const load = [];
  const unload = [];
  const bodies = {};
  let wakeAt = 0;
  const wake = (t) => { if (t && (!wakeAt || t < wakeAt)) wakeAt = t; };
  const zoom = view && view.zoom;
  const zoomed = zoomBody(zoom, view && view.band);
  // Two rows of the same table, decided once instead of per entry: what a
  // terminal renders at this zoom, and what a body with no cell renders.
  const flat = zoomBody(zoom, zoomed.band, false).body;
  const chats = [];
  for (const e of entries || []) {
    if (!e || !nonEmpty(e.id)) continue;
    const t = prev[e.id] || {};
    let on = !!e.loaded;
    if (e.near) {
      if (!e.loaded) {
        const loadAt = t.loadAt || now + LOAD_DWELL_MS;
        if (loadAt <= now) { load.push(e.id); on = true; }
        else { next[e.id] = { loadAt }; wake(loadAt); }
      }
    } else if (e.loaded && !pin.has(e.id)) {
      const unloadAt = t.unloadAt || now + UNLOAD_AFTER_MS;
      if (unloadAt <= now) { unload.push(e.id); on = false; }
      else { next[e.id] = { unloadAt }; wake(unloadAt); }
    }
    const body = on ? (e.pane === false ? flat : zoomed.body) : "off";
    bodies[e.id] = body;
    if (e.chat && body === "live") {
      // When this chat became live is the policy's memory, like the two
      // deadlines: fixed while the panel stays in the band, dropped the
      // moment it stops being live, so coming back makes it the newest.
      const liveAt = t.liveAt || now;
      next[e.id] = { ...(next[e.id] || {}), liveAt };
      chats.push({ id: e.id, at: liveAt });
    }
  }
  if (chats.length > CHAT_LIVE_MAX) {
    for (const id of chatBudget(chats).park) bodies[id] = "quiet";
  }
  return { load, unload, timers: next, wakeAt, band: zoomed.band, bodies };
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

// ---- the canvas plane (plan docs/plans/matrix-canvas.md §4.1) ------------
//
// The store keeps units, the canvas speaks pixels, and the conversion lives
// here — nowhere else multiplies by 8.

// unitsToPx(rect) -> { x, y, width, height }: a stored rectangle as the
// pixels a React Flow node wants.
export function unitsToPx(rect) {
  const r = rect || {};
  const n = (v) => (Number.isFinite(v) ? v : 0) * UNIT_PX;
  return { x: n(r.x), y: n(r.y), width: n(r.w), height: n(r.h) };
}

// pxToUnits(box) -> { x, y, w, h }: a node's pixels back to stored units.
// A drag lands on multiples of 8 (snapGrid), a resize does not (C0: the
// resizer ignores snapGrid for size), so the rounding is here and the store
// keeps whole units either way.
export function pxToUnits(box) {
  const b = box || {};
  const n = (v) => round((Number.isFinite(v) ? v : 0) / UNIT_PX) || 0; // never -0
  return { x: n(b.x), y: n(b.y), w: n(b.width), h: n(b.height) };
}

// Tidy (plan §3, §4.4). The plane has no automatic compaction, so the pack
// is an explicit action instead of a silent one. Panels keep the size their
// owner gave them and are laid out in reading order, left to right, wrapping
// past `cols` units, each row as tall as its tallest panel; the gap is one
// 8 px unit. Deterministic, so two viewers who tidy the same canvas get the
// same plane.
export const TIDY_GAP = 1; // units (8 px, the plane's step)
export const TIDY_ACROSS = 3; // default-sized panels per row
export const TIDY_COLS = TIDY_ACROSS * PANEL_DEFAULT_CANVAS.w + (TIDY_ACROSS - 1) * TIDY_GAP; // 98 units

export function tidyCanvas(panels, cols = TIDY_COLS, gap = TIDY_GAP) {
  const rects = (panels || []).filter((p) => wholeRect(p) && nonEmpty(p.id));
  const order = [...rects].sort((a, b) => a.y - b.y || a.x - b.x || (a.id < b.id ? -1 : a.id > b.id ? 1 : 0));
  const out = [];
  let x = 0;
  let y = 0;
  let rowH = 0;
  for (const p of order) {
    // A panel wider than the row still gets a row of its own rather than
    // hanging off the end of the previous one.
    if (x > 0 && x + p.w > cols) { x = 0; y += rowH + gap; rowH = 0; }
    out.push({ id: p.id, x, y, w: p.w, h: p.h });
    x += p.w + gap;
    rowH = Math.max(rowH, p.h);
  }
  return out;
}

// The viewport is per viewer and never stored server-side (ADR-0113): a
// camera is not an edit, and two browsers must not yank each other.
// ADR-0118 renamed the key. A camera stored under the old one is **moved**,
// once, the first time that canvas is opened (Plane.jsx `readView`): a
// rename the reader did not ask for must not throw their camera away, and
// fitView on a plane they had parked somewhere is exactly that.
export const VIEWPORT_PREFIX = "picode-canvas-view:";
export const VIEWPORT_PREFIX_WAS = "picode-matrix-view:";
export function viewportKey(id) {
  return VIEWPORT_PREFIX + str(id);
}
export function legacyViewportKey(id) {
  return VIEWPORT_PREFIX_WAS + str(id);
}

// normalizeViewport(v) -> { x, y, zoom } | null: what came out of
// localStorage, or nothing — a viewer whose stored zoom is junk (or from a
// build with other bounds) gets fitView instead of an unreachable plane.
export function normalizeViewport(v) {
  if (!v || typeof v !== "object") return null;
  const { x, y, zoom } = v;
  if (![x, y, zoom].every(Number.isFinite)) return null;
  const { canvasCoord } = CANVAS_LIMITS;
  const px = canvasCoord * UNIT_PX;
  return { x: clamp(x, -px, px), y: clamp(y, -px, px), zoom: clamp(zoom, CANVAS_ZOOM.min, CANVAS_ZOOM.max) };
}

// ---- keyboard (plan §4.6 "Focus") ---------------------------------------

export const PANEL_DIRECTIONS = Object.freeze(["left", "right", "up", "down"]);

const placed = (panels) => (panels || []).filter((p) => p && nonEmpty(p.id) && wholeRect(p));

// panelOrder(panels) -> ids in reading order (top-down, then left-to-right):
// Home, End and the roving tab stop of a canvas nobody has focused yet.
export function panelOrder(panels) {
  return placed(panels).sort((a, b) => a.y - b.y || a.x - b.x).map((p) => p.id);
}

// neighborPanel(panels, fromId, dir) -> id | "": where an arrow key lands.
// The nearest panel in that direction that shares rows (left, right) or
// columns (up, down) with the focused one; ties go to the one whose near
// edge lines up best, then to reading order. Up and down fall back to the
// nearest panel in that half of the plane, so a row holding one panel is
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
