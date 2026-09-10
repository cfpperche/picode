// Matrix (ADR-0108, ADR-0113): the client side of the store's contract.
// Pure — no React, no fetch. normalize* turn the API's JSON into the shapes the
// surface holds (junk dropped, the way contracts/appPrimitives.js does);
// validate* refuse what the server would refuse, in the server's words, so
// the UI never asks a question it knows the answer to; applyMatrixEvent
// reduces the seven feed events over { list: [summaries], byId: { id:
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
  cols: 12, // grid mode
  minW: 4, // columns
  minH: 8, // rows
  // Canvas mode (ADR-0113): the unit is 8 px, x and y may be negative, and
  // the plane is bounded only so a panel cannot be lost.
  canvasMinW: 32, // units (256 px)
  canvasMinH: 28, // units (224 px)
  canvasMax: 4096, // units, w and h
  canvasCoord: 100000, // units, |x| and |y|
});

// One canvas unit in CSS pixels — the grid's own 8 px margin, so snapGrid
// falls out of the coordinate system (plan §4.1).
export const UNIT_PX = 8;

// The switch transform's factors: a grid cell is 8 canvas units wide
// (colWidth / 8) and 3 tall (rowHeight 24 px / 8).
const CANVAS_PER_COL = 8;
const CANVAS_PER_ROW = 3;

// What a panel can be bound to (ADR-0108; the bodies beyond a live pane are
// C3 of docs/plans/matrix-canvas.md §4.2). `kind` is an open text column, so
// a kind is a validator edit on both sides and never a migration.
export const MATRIX_KINDS = Object.freeze(["agent", "terminal", "note"]);
export const MATRIX_COMPACT = Object.freeze(["vertical", "none"]);
// The layout mode of a matrix: what its panels' x/y/w/h mean. The first is
// the default a summary falls back to.
export const MATRIX_MODES = Object.freeze(["grid", "canvas"]);
export const MATRIX_EVENTS = Object.freeze([
  "matrix.created",
  "matrix.updated",
  "matrix.mode",
  "matrix.layout",
  "matrix.panel.added",
  "matrix.panel.removed",
  "matrix.deleted",
]);

// The server's refusals, word for word (internal/store/matrix.go), so the UI
// can refuse before asking and read a 400 back as the same sentence.
const KIND_MSG = "kind must be agent, terminal or note";

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
    mode: MATRIX_MODES.includes(m.mode) ? m.mode : "grid",
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

// validateMode(mode) -> "" | the server's refusal.
export function validateMode(mode) {
  return MATRIX_MODES.includes(mode) ? "" : `mode must be ${MATRIX_MODES.join(" or ")}`;
}

// validatePlacement({x, y, w, h}, mode) -> "" | the server's refusal, in the
// rules of that mode (ADR-0113): grid is 12 columns and panels of at least
// 4×8 cells, rows without end; canvas is a plane of 8 px units where x and y
// may be negative, a panel is at least 32×28 units, and both are bounded so
// a panel cannot be lost. The mode defaults to grid, so every v1 caller
// keeps working unchanged. The "whole number" line is the client's own —
// the server answers a fractional cell with "invalid JSON body".
export function validatePlacement(rect, mode = "grid") {
  const { x, y, w, h } = rect || {};
  for (const [k, v] of [["x", x], ["y", y], ["w", w], ["h", h]]) {
    if (!Number.isInteger(v)) return `${k} must be a whole number`;
  }
  const { cols, minW, minH, canvasMinW, canvasMinH, canvasMax, canvasCoord } = MATRIX_LIMITS;
  if (mode === "canvas") {
    if (x < -canvasCoord || x > canvasCoord) return `x must be between -${canvasCoord} and ${canvasCoord} canvas units`;
    if (y < -canvasCoord || y > canvasCoord) return `y must be between -${canvasCoord} and ${canvasCoord} canvas units`;
    if (w < canvasMinW) return `w must be at least ${canvasMinW} canvas units`;
    if (h < canvasMinH) return `h must be at least ${canvasMinH} canvas units`;
    if (w > canvasMax) return `w must be at most ${canvasMax} canvas units`;
    if (h > canvasMax) return `h must be at most ${canvasMax} canvas units`;
    return "";
  }
  if (x < 0) return "x must be 0 or more";
  if (y < 0) return "y must be 0 or more";
  if (w < minW) return `w must be at least ${minW} columns`;
  if (h < minH) return `h must be at least ${minH} rows`;
  if (x + w > cols) return `x + w must be at most ${cols} columns`;
  return "";
}

// validatePanel({kind, ref, x, y, w, h}, mode) -> "" | the server's refusal,
// in the server's order: binding first, then the rectangle of that mode.
export function validatePanel(panel, mode = "grid") {
  const p = panel || {};
  if (!MATRIX_KINDS.includes(p.kind)) return KIND_MSG;
  if (!str(p.ref).trim()) return "ref is required";
  return validatePlacement(p, mode);
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

// movePanels applies a subset of rectangles (a layout or a mode frame) to
// the panels a matrix holds, dropping rows that are not legal in `mode`.
function movePanels(panels, subset, mode) {
  const to = new Map();
  for (const p of Array.isArray(subset) ? subset : []) {
    if (p && nonEmpty(p.id) && !validatePlacement(p, mode)) to.set(p.id, { x: p.x, y: p.y, w: p.w, h: p.h });
  }
  return panels.map((p) => (to.has(p.id) ? { ...p, ...to.get(p.id) } : p));
}

// applyMatrixEvent(state, ev) -> next | same. state is { list, byId };
// an event for a matrix that is not loaded (no byId entry) only touches
// the summary list. Unknown types and junk return the same object. A
// rectangle is judged by the mode it belongs to: matrix.mode carries the
// new mode with the panels it moved, matrix.layout the mode the loaded
// matrix already has.
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
      const panels = movePanels(loaded.panels, d.panels, loaded.matrix.mode);
      return finish(s, nextList, { ...byId, [id]: { matrix: stamp(loaded.matrix), panels } });
    }
    // The switch answers with the summary (carrying the new mode) plus
    // exactly the panels it moved, so one frame is one reducer step: the
    // summary is complete, the rectangles are already in the new mode's
    // units.
    case "matrix.mode": {
      const m = normalizeMatrix(d);
      if (!m) return s;
      if (!loaded) return finish(s, upsertSummary(list, m), byId);
      const panels = movePanels(loaded.panels, d.panels, m.mode);
      return finish(s, upsertSummary(list, m), { ...byId, [id]: { matrix: { ...m, panelCount: panels.length }, panels } });
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
// minW/minH are the store's 4×8. On a canvas it is the same panel through
// the switch transform — 32×42 units, which is the same 256×336 px.
export const PANEL_DEFAULT = Object.freeze({ w: 4, h: 14 });
export const PANEL_DEFAULT_CANVAS = Object.freeze({ w: 32, h: 42 });

// panelDefault(mode) -> the default rectangle of that mode.
export function panelDefault(mode = "grid") {
  return mode === "canvas" ? PANEL_DEFAULT_CANVAS : PANEL_DEFAULT;
}
// Chunk loading (§4.5): a body mounts after its wrapper has been near the
// viewport for 300 ms and unmounts 5 s after it left; a suspended xterm is
// disposed past 24 instances or 10 minutes.
export const LOAD_DWELL_MS = 300;
export const UNLOAD_AFTER_MS = 5000;
export const SUSPENDED_MAX = 24;
export const SUSPENDED_TTL_MS = 10 * 60 * 1000;

const wholeRect = (p) => !!p && [p.x, p.y, p.w, p.h].every(Number.isInteger);
const overlaps = (a, b) => a.x < b.x + b.w && b.x < a.x + a.w && a.y < b.y + b.h && b.y < a.y + a.h;

// nextSlot(panels, w, h, mode) -> {x, y}: the first free w×h slot scanning
// rows top-down and left-to-right, else the row under everything. Only a
// panel's edges can start a first-free slot (a free rectangle slides up and
// left until it touches one), so the scan visits edges, not cells. Grid mode
// wraps at 12 columns; a canvas has no column cap, so a new panel lands
// beside the others instead of under them.
export function nextSlot(panels, w = PANEL_DEFAULT.w, h = PANEL_DEFAULT.h, mode = "grid") {
  const cols = mode === "canvas" ? Infinity : MATRIX_LIMITS.cols;
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

// ---- the switch transform (ADR-0113) -------------------------------------
//
// The same conversion the store applies, so the UI can preview a switch
// before asking for it: gridToCanvas scales, canvasToGrid divides, rounds,
// clamps and packs. Both take the panels and answer the same panels with
// new rectangles, in the order they came in.

const clamp = (v, lo, hi) => (v < lo ? lo : v > hi ? hi : v);
// Round half away from zero, the way Go's math.Round does, so a negative
// coordinate lands on the same cell on both sides.
const round = (v) => (v < 0 ? -Math.round(-v) : Math.round(v));

// gridToCanvas(panels): a cell is 8 units wide and 3 tall. A panel at the
// 8-row minimum becomes 24 units tall, under the canvas minimum of 28, so it
// grows — and can then overlap the panel below by up to 4 units. Canvas mode
// is free placement and allows that; switching back packs it out.
export function gridToCanvas(panels) {
  const { canvasMinW, canvasMinH, canvasMax, canvasCoord } = MATRIX_LIMITS;
  return (panels || []).filter(wholeRect).map((p) => ({
    ...p,
    x: clamp(p.x * CANVAS_PER_COL, -canvasCoord, canvasCoord),
    y: clamp(p.y * CANVAS_PER_ROW, -canvasCoord, canvasCoord),
    w: clamp(p.w * CANVAS_PER_COL, canvasMinW, canvasMax),
    h: clamp(p.h * CANVAS_PER_ROW, canvasMinH, canvasMax),
  }));
}

// canvasToGrid(panels): divide by the same factors, round, clamp into the
// 12-column rules, then place each panel in reading order (ties by id) at
// the first free slot. Rounding alone leaves panels overlapping, and overlap
// after a switch is a bug, not a tolerance — the pack is what makes the
// switch total.
export function canvasToGrid(panels) {
  const { cols, minW, minH } = MATRIX_LIMITS;
  const rounded = (panels || []).filter((p) => wholeRect(p) && nonEmpty(p.id)).map((p) => {
    const w = clamp(round(p.w / CANVAS_PER_COL), minW, cols);
    return {
      ...p,
      w,
      h: Math.max(round(p.h / CANVAS_PER_ROW), minH),
      x: clamp(round(p.x / CANVAS_PER_COL), 0, cols - w),
      y: Math.max(round(p.y / CANVAS_PER_ROW), 0),
    };
  });
  const placed = [];
  const at = new Map();
  for (const p of [...rounded].sort((a, b) => a.y - b.y || a.x - b.x || (a.id < b.id ? -1 : a.id > b.id ? 1 : 0))) {
    const slot = nextSlot(placed, p.w, p.h);
    const moved = { ...p, x: slot.x, y: slot.y };
    placed.push(moved);
    at.set(p.id, moved);
  }
  return rounded.map((p) => at.get(p.id));
}

// layoutDiff(prev, next, mode) -> [{id, x, y, w, h}]: the panels whose
// rectangle changed between two layouts, which is exactly what PATCH
// …/layout takes. A panel only in `next` is an add (its own POST), only in
// `prev` a remove; a rectangle the matrix's mode refuses is left out so one
// bad row cannot sink the batch.
export function layoutDiff(prev, next, mode = "grid") {
  const before = new Map((prev || []).filter((p) => p && nonEmpty(p.id)).map((p) => [p.id, p]));
  const out = [];
  for (const p of next || []) {
    const b = p && before.get(p.id);
    if (!b || !wholeRect(p) || validatePlacement(p, mode)) continue;
    if (b.x !== p.x || b.y !== p.y || b.w !== p.w || b.h !== p.h) out.push({ id: p.id, x: p.x, y: p.y, w: p.w, h: p.h });
  }
  return out;
}

// bindingState(panel, fleet) -> one row of plan §4.4:
//   terminal-running | terminal-stopped | terminal-gone
//   agent-interactive | agent-managed | agent-stopped | agent-gone
//   note-ready | note-gone
// fleet is the host's { workspaces, freeAgents, terminals } plus, for the
// kinds that bind something else, the list that decides them: `pins` for a
// note. A shell whose tmux session died is still "running" for the panel:
// opening it revives the shell; only a configured CLI terminal has a stopped
// state of its own.
//
// A list that has not been read yet is `null`, not `[]` — the same rule the
// fleet follows (`host.fleet.loaded`): before the read, nothing is gone.
export function bindingState(panel, fleet) {
  const f = fleet || {};
  if (!panel) return "";
  if (panel.kind === "note") {
    if (!Array.isArray(f.pins)) return "note-ready";
    return f.pins.some((x) => x && x.id === panel.ref) ? "note-ready" : "note-gone";
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
//   entries  [{ id, near, loaded, pane }] — near: inside the viewport ± the
//            observer's margin (its word) and the surface visible; pane:
//            whether this body is a terminal (default true, so grid mode and
//            every v1 caller are unchanged)
//   pinned   Set of ids that never unload (dragged, resized, focused,
//            maximized)
//   timers   the previous call's `timers` — the policy's only memory:
//            { id: { loadAt } | { unloadAt } }
//   view     { zoom, band } — the canvas's zoom and the band it was last in.
//            Grid mode passes nothing and gets zoom 1: every loaded body is
//            live, exactly as it was before canvas mode.
// A wrapper loads once it has been near for LOAD_DWELL_MS; leaving cancels
// a pending load at once. A loaded body unloads UNLOAD_AFTER_MS after it
// left; coming back cancels the unload. wakeAt is the earliest deadline,
// 0 when nothing is pending — the caller sleeps until then.
// `bodies` is what each entry renders now: off (not loaded — the
// placeholder, and the attach is suspended), live, still or plate — the
// still only for an entry that has a pane. The band is screen-space, so
// zooming out puts everything in it (C0: 500 of 500 at 0.2); what bounds the
// attaches down there is the still rule, not the band.
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
    bodies[e.id] = on ? (e.pane === false ? flat : zoomed.body) : "off";
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

// Tidy (plan §3, §4.4). Canvas mode has no automatic compaction —
// react-grid-layout's job in grid mode — so the pack is an explicit action
// instead of a silent one. Panels keep the size their owner gave them and
// are laid out in reading order, left to right, wrapping past `cols` units,
// each row as tall as its tallest panel; the gap is the grid's own 8 px
// gutter. Deterministic, so two viewers who tidy the same matrix get the
// same plane.
export const TIDY_GAP = 1; // units (8 px, the grid's margin)
export const TIDY_ACROSS = 3; // default-sized panels per row — what the 12-column grid shows
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
export const VIEWPORT_PREFIX = "picode-matrix-view:";
export function viewportKey(id) {
  return VIEWPORT_PREFIX + str(id);
}

// normalizeViewport(v) -> { x, y, zoom } | null: what came out of
// localStorage, or nothing — a viewer whose stored zoom is junk (or from a
// build with other bounds) gets fitView instead of an unreachable plane.
export function normalizeViewport(v) {
  if (!v || typeof v !== "object") return null;
  const { x, y, zoom } = v;
  if (![x, y, zoom].every(Number.isFinite)) return null;
  const { canvasCoord } = MATRIX_LIMITS;
  const px = canvasCoord * UNIT_PX;
  return { x: clamp(x, -px, px), y: clamp(y, -px, px), zoom: clamp(zoom, CANVAS_ZOOM.min, CANVAS_ZOOM.max) };
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
