import assert from "node:assert/strict";
import { test } from "node:test";
import { touches } from "./feedReducers.js";
import {
  LOAD_DWELL_MS, MATRIX_EVENTS, MATRIX_LIMITS, MATRIX_MODES, PANEL_DEFAULT, PANEL_DEFAULT_CANVAS, PANEL_DIRECTIONS, SUSPENDED_MAX,
  SUSPENDED_TTL_MS, UNIT_PX, UNLOAD_AFTER_MS, applyMatrixEvent, bindingState, canvasToGrid, gridToCanvas, layoutDiff, loadPolicy,
  neighborPanel, nextSlot, normalizeMatrix, normalizeMatrixDetail, normalizeMatrixList, normalizePanel, panelDefault, panelOrder,
  suspendedToDispose, validateCompact, validateMode, validateName, validatePanel, validatePlacement,
} from "./matrix.js";

const summary = (id, name, extra = {}) => ({
  id, name, compact: "vertical", mode: "grid", createdAt: "2026-09-09T10:00:00Z", updatedAt: "2026-09-09T10:00:00Z", panelCount: 0, ...extra,
});
const panel = (id, extra = {}) => ({ id, kind: "terminal", ref: "term-" + id, x: 0, y: 0, w: 4, h: 8, createdAt: "2026-09-09T10:00:00Z", ...extra });

test("limits, modes and event types are the server's (ADR-0108, ADR-0113)", () => {
  assert.deepEqual({ ...MATRIX_LIMITS }, {
    matrices: 64, panels: 500, name: 80, cols: 12, minW: 4, minH: 8,
    canvasMinW: 32, canvasMinH: 28, canvasMax: 4096, canvasCoord: 100000,
  });
  assert.deepEqual([...MATRIX_MODES], ["grid", "canvas"]);
  assert.equal(UNIT_PX, 8);
  assert.deepEqual([...MATRIX_EVENTS], ["matrix.created", "matrix.updated", "matrix.mode", "matrix.layout", "matrix.panel.added", "matrix.panel.removed", "matrix.deleted"]);
  for (const type of MATRIX_EVENTS) assert.equal(touches({ type }, ["matrix"]), true, type);
});

test("normalizeMatrix keeps the summary fields and drops junk", () => {
  const m = normalizeMatrix({ id: "m1", name: "Ops", compact: "none", mode: "canvas", createdAt: "a", updatedAt: "b", panelCount: 3, panels: [{}], extra: 1 });
  assert.deepEqual(m, { id: "m1", name: "Ops", compact: "none", mode: "canvas", createdAt: "a", updatedAt: "b", panelCount: 3 });
  assert.equal(normalizeMatrix({ id: "m1", name: "Ops", compact: "diagonal" }).compact, "vertical");
  assert.equal(normalizeMatrix({ id: "m1", name: "Ops" }).mode, "grid", "a summary from before the mode column reads grid");
  assert.equal(normalizeMatrix({ id: "m1", name: "Ops", mode: "isometric" }).mode, "grid");
  assert.equal(normalizeMatrix({ id: "m1", name: "Ops", panelCount: -2 }).panelCount, 0);
  assert.equal(normalizeMatrix({ id: "m1", name: "Ops", panelCount: "7" }).panelCount, 0);
  assert.equal(normalizeMatrix({ name: "Ops" }), null);
  assert.equal(normalizeMatrix({ id: "m1" }), null);
  assert.equal(normalizeMatrix(null), null);
  assert.deepEqual(normalizeMatrixList({ matrices: [summary("b", "b"), null, { id: "x" }] }), [summary("b", "b")]);
  assert.deepEqual(normalizeMatrixList(null), []);
});

test("normalizePanel needs a slot id, a known binding and a whole rectangle", () => {
  assert.deepEqual(normalizePanel({ ...panel("p1"), junk: true }), panel("p1"));
  assert.equal(normalizePanel({ ...panel("p1"), ref: "  term-p1 " }).ref, "term-p1");
  assert.equal(normalizePanel({ ...panel("p1"), x: 1.5 }), null);
  assert.equal(normalizePanel({ ...panel("p1"), w: "4" }), null);
  assert.equal(normalizePanel({ ...panel("p1"), kind: "pin" }), null);
  assert.equal(normalizePanel({ ...panel("p1"), ref: "" }), null);
  assert.equal(normalizePanel({ ...panel("p1"), id: 7 }), null);
  assert.equal(normalizePanel(null), null);
  const d = normalizeMatrixDetail({ ...summary("m1", "Ops", { panelCount: 9 }), panels: [panel("p1"), { bad: 1 }, panel("p2")] });
  assert.deepEqual(d.matrix, summary("m1", "Ops", { panelCount: 2 }));
  assert.deepEqual(d.panels.map((p) => p.id), ["p1", "p2"]);
  assert.deepEqual(normalizeMatrixDetail(summary("m1", "Ops")).panels, []);
  assert.equal(normalizeMatrixDetail({ panels: [] }), null);
});

test("validate* refuse in the server's words", () => {
  assert.equal(validateName("Ops"), "");
  assert.equal(validateName("  Ops  "), "");
  assert.equal(validateName("  "), "name is required");
  assert.equal(validateName(undefined), "name is required");
  assert.equal(validateName("é".repeat(80)), "");
  assert.equal(validateName("é".repeat(81)), "name is too long (max 80 characters)");
  assert.equal(validateName("😀".repeat(80)), ""); // code points, not UTF-16 units
  assert.equal(validateCompact("vertical"), "");
  assert.equal(validateCompact("none"), "");
  assert.equal(validateCompact("diagonal"), "compact must be vertical or none");
  const cases = [
    [{ x: -1, y: 0, w: 4, h: 8 }, "x must be 0 or more"],
    [{ x: 0, y: -1, w: 4, h: 8 }, "y must be 0 or more"],
    [{ x: 0, y: 0, w: 3, h: 8 }, "w must be at least 4 columns"],
    [{ x: 0, y: 0, w: 4, h: 7 }, "h must be at least 8 rows"],
    [{ x: 9, y: 0, w: 4, h: 8 }, "x + w must be at most 12 columns"],
    [{ x: 0.5, y: 0, w: 4, h: 8 }, "x must be a whole number"],
    [{ x: 0, y: 0, w: 4 }, "h must be a whole number"],
    [{ x: 8, y: 0, w: 4, h: 8 }, ""],
    [{ x: 0, y: 40, w: 12, h: 100 }, ""],
  ];
  for (const [rect, want] of cases) assert.equal(validatePlacement(rect), want, JSON.stringify(rect));
  assert.equal(validatePlacement(null), "x must be a whole number");
  assert.equal(validatePanel({ kind: "pin", ref: "x", x: 0, y: 0, w: 4, h: 8 }), "kind must be agent or terminal");
  assert.equal(validatePanel({ kind: "agent", ref: "  ", x: 0, y: 0, w: 4, h: 8 }), "ref is required");
  assert.equal(validatePanel({ kind: "agent", ref: "a1", x: 0, y: 0, w: 3, h: 8 }), "w must be at least 4 columns");
  assert.equal(validatePanel({ kind: "terminal", ref: "t1", x: 8, y: 0, w: 4, h: 8 }), "");
  assert.equal(validatePanel(undefined), "kind must be agent or terminal");
});

test("created and updated keep the list sorted by name and follow a loaded matrix", () => {
  let s = { list: [summary("m2", "beta")], byId: {} };
  s = applyMatrixEvent(s, { type: "matrix.created", data: summary("m1", "Alpha") });
  assert.deepEqual(s.list.map((m) => m.id), ["m1", "m2"]);
  s = applyMatrixEvent(s, { type: "matrix.updated", data: summary("m1", "zeta", { updatedAt: "t2" }) });
  assert.deepEqual(s.list.map((m) => m.name), ["beta", "zeta"]);
  assert.equal(s.list[1].updatedAt, "t2");
  // An updated summary for a matrix the list never saw is complete: inserted.
  s = applyMatrixEvent(s, { type: "matrix.updated", data: summary("m3", "gamma") });
  assert.deepEqual(s.list.map((m) => m.name), ["beta", "gamma", "zeta"]);
  // A replayed created event does not duplicate.
  s = applyMatrixEvent(s, { type: "matrix.created", data: summary("m3", "gamma") });
  assert.equal(s.list.length, 3);
  // Same name: creation order decides.
  s = applyMatrixEvent(s, { type: "matrix.created", data: summary("m0", "Beta", { createdAt: "2026-09-09T09:00:00Z" }) });
  assert.deepEqual(s.list.map((m) => m.id), ["m0", "m2", "m3", "m1"]);
  // A loaded matrix follows; its panelCount stays what the panels say.
  const loaded = { list: [summary("m1", "Ops", { panelCount: 1 })], byId: { m1: { matrix: summary("m1", "Ops", { panelCount: 1 }), panels: [panel("p1")] } } };
  const next = applyMatrixEvent(loaded, { type: "matrix.updated", data: summary("m1", "Ops board", { compact: "none", panelCount: 7, updatedAt: "t2" }) });
  assert.equal(next.byId.m1.matrix.name, "Ops board");
  assert.equal(next.byId.m1.matrix.compact, "none");
  assert.equal(next.byId.m1.matrix.panelCount, 1);
  assert.equal(next.byId.m1.matrix.updatedAt, "t2");
  assert.deepEqual(next.byId.m1.panels, [panel("p1")]);
  assert.equal(next.list[0].name, "Ops board");
  // Junk, unknown types and other matrices leave the same object.
  assert.equal(applyMatrixEvent(loaded, { type: "matrix.created", data: { name: "no id" } }), loaded);
  assert.equal(applyMatrixEvent(loaded, { type: "pin.created", data: { id: "m1" } }), loaded);
  assert.equal(applyMatrixEvent(loaded, { type: "matrix.layout", data: { id: "other", updatedAt: "t9", panels: [] } }), loaded);
  assert.equal(applyMatrixEvent(loaded, { type: "matrix.panel.removed", data: { id: "m1", updatedAt: "t9" } }), loaded);
  assert.deepEqual(applyMatrixEvent(undefined, { type: "matrix.created", data: summary("m1", "Ops") }).list.map((m) => m.id), ["m1"]);
});

test("layout moves exactly the subset it carries", () => {
  const s = {
    list: [summary("m1", "Ops", { panelCount: 2 })],
    byId: { m1: { matrix: summary("m1", "Ops", { panelCount: 2 }), panels: [panel("p1"), panel("p2", { x: 4 })] } },
  };
  const ev = {
    type: "matrix.layout",
    data: { id: "m1", updatedAt: "t2", panels: [{ id: "p1", x: 0, y: 8, w: 8, h: 16 }, { id: "p9", x: 0, y: 0, w: 4, h: 8 }, { id: "p2", x: 20, y: 0, w: 4, h: 8 }] },
  };
  const next = applyMatrixEvent(s, ev);
  // p9 is not here (its panel.added frame comes on its own); p2's frame is
  // outside the grid and cannot be honest — both are left alone.
  assert.deepEqual(next.byId.m1.panels, [panel("p1", { y: 8, w: 8, h: 16 }), panel("p2", { x: 4 })]);
  assert.equal(next.byId.m1.matrix.updatedAt, "t2");
  assert.equal(next.list[0].updatedAt, "t2");
  assert.equal(next.list[0].panelCount, 2);
  assert.deepEqual(s.byId.m1.panels[0], panel("p1"), "the previous state is not mutated");
  // Not loaded: only the summary's updatedAt moves.
  const unloaded = applyMatrixEvent({ list: s.list, byId: {} }, ev);
  assert.equal(unloaded.list[0].updatedAt, "t2");
  assert.deepEqual(unloaded.byId, {});
});

test("panels added and removed, the counts follow on both shapes", () => {
  let s = { list: [summary("m1", "Ops"), summary("m2", "Two", { panelCount: 3 })], byId: { m1: { matrix: summary("m1", "Ops"), panels: [] } } };
  s = applyMatrixEvent(s, { type: "matrix.panel.added", data: { id: "m1", updatedAt: "t2", panel: { ...panel("p1"), junk: 1 } } });
  assert.deepEqual(s.byId.m1.panels, [panel("p1")]);
  assert.equal(s.byId.m1.matrix.panelCount, 1);
  assert.equal(s.byId.m1.matrix.updatedAt, "t2");
  assert.equal(s.list[0].panelCount, 1);
  assert.equal(s.list[0].updatedAt, "t2");
  // The same panel again (a replay) is a replacement, not a second slot.
  s = applyMatrixEvent(s, { type: "matrix.panel.added", data: { id: "m1", updatedAt: "t2", panel: panel("p1", { x: 4 }) } });
  assert.deepEqual(s.byId.m1.panels, [panel("p1", { x: 4 })]);
  assert.equal(s.list[0].panelCount, 1);
  // Not loaded: the summary count moves by one; nothing is loaded by an event.
  s = applyMatrixEvent(s, { type: "matrix.panel.added", data: { id: "m2", updatedAt: "t3", panel: panel("p5") } });
  assert.equal(s.list[1].panelCount, 4);
  assert.equal(s.list[1].updatedAt, "t3");
  assert.equal(s.byId.m2, undefined);
  s = applyMatrixEvent(s, { type: "matrix.panel.removed", data: { id: "m2", updatedAt: "t4", panelId: "p5" } });
  assert.equal(s.list[1].panelCount, 3);
  assert.equal(s.list[1].updatedAt, "t4");
  s = applyMatrixEvent(s, { type: "matrix.panel.removed", data: { id: "m1", updatedAt: "t5", panelId: "p1" } });
  assert.deepEqual(s.byId.m1.panels, []);
  assert.equal(s.byId.m1.matrix.panelCount, 0);
  assert.equal(s.list[0].panelCount, 0);
  assert.equal(s.list[0].updatedAt, "t5");
  // A panel that cannot be placed is ignored; a count never goes below zero.
  assert.equal(applyMatrixEvent(s, { type: "matrix.panel.added", data: { id: "m1", updatedAt: "t6", panel: { id: "p2" } } }), s);
  const zero = applyMatrixEvent({ list: [summary("m3", "z")], byId: {} }, { type: "matrix.panel.removed", data: { id: "m3", updatedAt: "t", panelId: "x" } });
  assert.equal(zero.list[0].panelCount, 0);
});

test("deleted drops the summary and the loaded matrix", () => {
  const s = {
    list: [summary("m1", "Ops"), summary("m2", "Two")],
    byId: { m1: { matrix: summary("m1", "Ops"), panels: [panel("p1")] }, m2: { matrix: summary("m2", "Two"), panels: [] } },
  };
  const next = applyMatrixEvent(s, { type: "matrix.deleted", data: { id: "m1" } });
  assert.deepEqual(next.list.map((m) => m.id), ["m2"]);
  assert.deepEqual(Object.keys(next.byId), ["m2"]);
  assert.equal(applyMatrixEvent(next, { type: "matrix.deleted", data: { id: "m1" } }), next);
  assert.equal(s.list.length, 2, "the previous state is not mutated");
});

// ---- the surface's arithmetic (plan §4.3–§4.6) ----------------------------

const rect = (id, x, y, w = 4, h = 14) => ({ id, x, y, w, h });

test("nextSlot: the first free slot scanning rows, then the bottom", () => {
  assert.deepEqual(PANEL_DEFAULT, { w: 4, h: 14 });
  assert.deepEqual(nextSlot([]), { x: 0, y: 0 });
  assert.deepEqual(nextSlot([rect("a", 0, 0)]), { x: 4, y: 0 }, "beside the first panel");
  assert.deepEqual(nextSlot([rect("a", 0, 0), rect("b", 8, 0)]), { x: 4, y: 0 }, "a gap in the first row is taken first");
  assert.deepEqual(nextSlot([rect("a", 0, 0), rect("b", 4, 0), rect("c", 8, 0)]), { x: 0, y: 14 }, "a full row: the next row");
  assert.deepEqual(nextSlot([rect("a", 0, 0), rect("b", 4, 0, 8, 8)]), { x: 4, y: 8 }, "under a short panel, beside a tall one");
  assert.deepEqual(nextSlot([rect("a", 0, 0)], 12, 14), { x: 0, y: 14 }, "a full-width panel goes under everything");
  assert.deepEqual(nextSlot([rect("a", 0, 0, 12, 8)], 4, 8), { x: 0, y: 8 });
  assert.deepEqual(nextSlot([rect("a", 0, 0), rect("b", 4, 0, 4, 8), rect("c", 8, 0)], 4, 8), { x: 4, y: 8 }, "a pocket the exact size counts");
  assert.deepEqual(nextSlot([rect("a", 0, 0), rect("b", 4, 0, 4, 8), rect("c", 8, 0), rect("d", 4, 12)], 4, 8), { x: 0, y: 14 }, "a pocket with a panel under it is too short");
  assert.deepEqual(nextSlot([rect("a", 0, 0), { id: "junk", x: 1.5 }]), { x: 4, y: 0 }, "a panel without a whole rectangle is ignored");
  // A canvas has no column cap: a new panel lands beside the others, never
  // wrapped into a 12-column row.
  assert.deepEqual(nextSlot([], 32, 42, "canvas"), { x: 0, y: 0 });
  assert.deepEqual(nextSlot([rect("a", 0, 0, 32, 42)], 32, 42, "canvas"), { x: 32, y: 0 }, "beside, not under");
  assert.deepEqual(nextSlot([rect("a", 0, 0, 96, 42), rect("b", 96, 0, 96, 42)], 96, 42, "canvas"), { x: 192, y: 0 });
  assert.deepEqual(nextSlot([rect("a", 0, 0, 96, 42)], 96, 42), { x: 0, y: 42 }, "the same panels in grid mode wrap");
  assert.deepEqual(PANEL_DEFAULT_CANVAS, { w: 32, h: 42 });
  assert.deepEqual(panelDefault("canvas"), PANEL_DEFAULT_CANVAS);
  assert.deepEqual(panelDefault(), PANEL_DEFAULT);
});

test("layoutDiff: the changed rectangles of panels present in both layouts", () => {
  const prev = [rect("a", 0, 0), rect("b", 4, 0), rect("c", 8, 0)];
  assert.deepEqual(layoutDiff(prev, prev), []);
  assert.deepEqual(layoutDiff(prev, [rect("a", 0, 14), rect("b", 4, 0), rect("c", 8, 0)]), [{ id: "a", x: 0, y: 14, w: 4, h: 14 }]);
  assert.deepEqual(layoutDiff(prev, [rect("a", 0, 0), rect("b", 4, 0, 8, 14), rect("c", 8, 14)]), [{ id: "b", x: 4, y: 0, w: 8, h: 14 }, { id: "c", x: 8, y: 14, w: 4, h: 14 }]);
  assert.deepEqual(layoutDiff(prev, [...prev, rect("new", 0, 14)]), [], "an added panel is its own POST");
  assert.deepEqual(layoutDiff(prev, [rect("a", 0, 0)]), [], "a removed panel is its own DELETE");
  assert.deepEqual(layoutDiff(prev, [{ ...rect("a", 0, 14), kind: "terminal", ref: "t1", extra: 1 }]), [{ id: "a", x: 0, y: 14, w: 4, h: 14 }], "only the rectangle travels");
  assert.deepEqual(layoutDiff(prev, [rect("a", 0, 14, 3, 14), rect("b", 4, 14)]), [{ id: "b", x: 4, y: 14, w: 4, h: 14 }], "an invalid rectangle is left out, the rest still goes");
  assert.deepEqual(layoutDiff(null, null), []);
});

// ---- canvas mode (ADR-0113) ----------------------------------------------

test("validate* judge a rectangle by the mode it belongs to", () => {
  assert.equal(validateMode("grid"), "");
  assert.equal(validateMode("canvas"), "");
  assert.equal(validateMode("isometric"), "mode must be grid or canvas");
  assert.equal(validateMode(undefined), "mode must be grid or canvas");
  const cases = [
    [{ x: 0, y: 0, w: 31, h: 42 }, "w must be at least 32 canvas units"],
    [{ x: 0, y: 0, w: 32, h: 27 }, "h must be at least 28 canvas units"],
    [{ x: 0, y: 0, w: 4097, h: 42 }, "w must be at most 4096 canvas units"],
    [{ x: 0, y: 0, w: 32, h: 4097 }, "h must be at most 4096 canvas units"],
    [{ x: 100001, y: 0, w: 32, h: 42 }, "x must be between -100000 and 100000 canvas units"],
    [{ x: -100001, y: 0, w: 32, h: 42 }, "x must be between -100000 and 100000 canvas units"],
    [{ x: 0, y: -100001, w: 32, h: 42 }, "y must be between -100000 and 100000 canvas units"],
    [{ x: 0.5, y: 0, w: 32, h: 42 }, "x must be a whole number"],
    [{ x: -4000, y: -2500, w: 32, h: 28 }, ""], // negative is the plane's, not a mistake
    [{ x: 100000, y: -100000, w: 4096, h: 4096 }, ""],
  ];
  for (const [r, want] of cases) assert.equal(validatePlacement(r, "canvas"), want, JSON.stringify(r));
  // The same rectangles under the other mode, and the grid rules unchanged
  // when no mode is passed at all.
  assert.equal(validatePlacement({ x: 0, y: 0, w: 32, h: 42 }, "grid"), "x + w must be at most 12 columns");
  assert.equal(validatePlacement({ x: 0, y: 0, w: 32, h: 42 }), "x + w must be at most 12 columns");
  assert.equal(validatePlacement({ x: 0, y: 0, w: 4, h: 8 }, "canvas"), "w must be at least 32 canvas units");
  assert.equal(validatePlacement({ x: -1, y: 0, w: 4, h: 8 }), "x must be 0 or more");
  assert.equal(validatePanel({ kind: "terminal", ref: "t1", x: -40, y: -40, w: 32, h: 28 }, "canvas"), "");
  assert.equal(validatePanel({ kind: "terminal", ref: "t1", x: 0, y: 0, w: 4, h: 8 }, "canvas"), "w must be at least 32 canvas units");
  assert.equal(validatePanel({ kind: "pin", ref: "t1", x: 0, y: 0, w: 32, h: 42 }, "canvas"), "kind must be agent or terminal");
  // layoutDiff follows the mode, or every canvas move would be dropped.
  const prev = [rect("a", 0, 0, 32, 42)];
  assert.deepEqual(layoutDiff(prev, [rect("a", -8, -8, 40, 40)], "canvas"), [{ id: "a", x: -8, y: -8, w: 40, h: 40 }]);
  assert.deepEqual(layoutDiff(prev, [rect("a", -8, -8, 40, 40)]), [], "the grid rules refuse it");
});

// The transform the store applies (internal/store/matrix.go), repeated here
// so the UI can preview a switch — the fixtures are the store tests'.
test("gridToCanvas: a cell is 8 units wide and 3 tall, then the minimums", () => {
  assert.deepEqual(gridToCanvas([rect("a", 0, 0, 4, 14), rect("b", 4, 0, 8, 8), rect("c", 0, 14, 12, 40)]), [
    { id: "a", x: 0, y: 0, w: 32, h: 42 },
    { id: "b", x: 32, y: 0, w: 64, h: 28 }, // 8×3 = 24, clamped to the 28-unit minimum
    { id: "c", x: 0, y: 42, w: 96, h: 120 },
  ]);
  assert.deepEqual(gridToCanvas([{ id: "junk", x: 1.5, y: 0, w: 4, h: 8 }]), []);
  assert.deepEqual(gridToCanvas(null), []);
  // The default panel is the same panel in both modes.
  assert.deepEqual(gridToCanvas([{ id: "p", x: 0, y: 0, ...PANEL_DEFAULT }])[0], { id: "p", x: 0, y: 0, ...PANEL_DEFAULT_CANVAS });
});

test("canvasToGrid: divide, round, clamp, then pack in reading order", () => {
  // a covers (0,0)–(4,14); b and c round onto it; d's negative x clamps to
  // the left edge and its row is full, so it goes under.
  const packed = canvasToGrid([
    rect("a", 0, 0, 32, 42),
    rect("b", 12, 2, 32, 42),
    rect("c", 28, 5, 33, 43),
    rect("d", -600, 90, 40, 30),
  ]);
  assert.deepEqual(packed, [
    { id: "a", x: 0, y: 0, w: 4, h: 14 },
    { id: "b", x: 4, y: 0, w: 4, h: 14 },
    { id: "c", x: 8, y: 0, w: 4, h: 14 },
    { id: "d", x: 0, y: 14, w: 5, h: 10 },
  ]);
  for (const p of packed) assert.equal(validatePlacement(p), "", JSON.stringify(p));
  // Nothing is lost and nothing overlaps.
  const overlaps = (a, b) => a.x < b.x + b.w && b.x < a.x + a.w && a.y < b.y + b.h && b.y < a.y + a.h;
  for (let i = 0; i < packed.length; i++) {
    for (let j = i + 1; j < packed.length; j++) assert.equal(overlaps(packed[i], packed[j]), false, `${packed[i].id} × ${packed[j].id}`);
  }
  // A round trip is legal and disjoint, never identical: the 8-row minimum
  // leaves as 28 units (24 clamped up) and comes back 9 rows tall.
  const start = [rect("a", 0, 0, 4, 8), rect("b", 4, 0, 4, 8), rect("c", 0, 8, 12, 8)];
  const back = canvasToGrid(gridToCanvas(start));
  assert.equal(back.length, 3);
  for (const p of back) assert.equal(validatePlacement(p), "", JSON.stringify(p));
  for (let i = 0; i < back.length; i++) {
    for (let j = i + 1; j < back.length; j++) assert.equal(overlaps(back[i], back[j]), false, `${back[i].id} × ${back[j].id}`);
  }
  assert.deepEqual(back, [
    { id: "a", x: 0, y: 0, w: 4, h: 9 },
    { id: "b", x: 4, y: 0, w: 4, h: 9 },
    { id: "c", x: 0, y: 9, w: 12, h: 9 },
  ]);
  assert.deepEqual(canvasToGrid([{ x: 0, y: 0, w: 32, h: 42 }]), [], "a rectangle with no panel id cannot be placed");
});

test("matrix.mode carries the new summary and the panels it moved", () => {
  const gridPanels = [panel("p1"), panel("p2", { x: 4 })];
  const s = {
    list: [summary("m1", "Ops", { panelCount: 2 }), summary("m2", "Two")],
    byId: { m1: { matrix: summary("m1", "Ops", { panelCount: 2 }), panels: gridPanels } },
  };
  const ev = {
    type: "matrix.mode",
    data: {
      ...summary("m1", "Ops", { mode: "canvas", panelCount: 2, updatedAt: "t2" }),
      panels: [{ id: "p1", x: 0, y: 0, w: 32, h: 28 }, { id: "p2", x: 32, y: 0, w: 32, h: 28 }],
    },
  };
  const next = applyMatrixEvent(s, ev);
  assert.equal(next.byId.m1.matrix.mode, "canvas");
  assert.equal(next.byId.m1.matrix.updatedAt, "t2");
  assert.equal(next.byId.m1.matrix.panelCount, 2);
  assert.deepEqual(next.byId.m1.panels, [panel("p1", { w: 32, h: 28 }), panel("p2", { x: 32, w: 32, h: 28 })]);
  assert.equal(next.list[0].mode, "canvas");
  assert.deepEqual(s.byId.m1.panels, gridPanels, "the previous state is not mutated");
  // The rectangles are judged in the mode the frame carries: a grid-sized
  // row inside a canvas frame is not honest and is left alone.
  const half = applyMatrixEvent(s, {
    type: "matrix.mode",
    data: { ...summary("m1", "Ops", { mode: "canvas", updatedAt: "t3" }), panels: [{ id: "p1", x: 0, y: 0, w: 4, h: 8 }] },
  });
  assert.deepEqual(half.byId.m1.panels, gridPanels);
  assert.equal(half.byId.m1.matrix.mode, "canvas");
  // A layout frame that follows is judged by the mode the matrix now has.
  const moved = applyMatrixEvent(next, { type: "matrix.layout", data: { id: "m1", updatedAt: "t4", panels: [{ id: "p1", x: -80, y: -8, w: 40, h: 30 }] } });
  assert.deepEqual(moved.byId.m1.panels[0], panel("p1", { x: -80, y: -8, w: 40, h: 30 }));
  // Not loaded: the summary is complete, so the list follows and nothing loads.
  const unloaded = applyMatrixEvent({ list: s.list, byId: {} }, ev);
  assert.equal(unloaded.list[0].mode, "canvas");
  assert.deepEqual(unloaded.byId, {});
  // A summary a list has never seen is inserted, the way created/updated do.
  const fresh = applyMatrixEvent({ list: [], byId: {} }, ev);
  assert.deepEqual(fresh.list.map((m) => m.id), ["m1"]);
  assert.equal(applyMatrixEvent(s, { type: "matrix.mode", data: { panels: [] } }), s);
});

// §4.4, one assertion per row.
const fleet = {
  workspaces: [{ id: "w1", name: "App", agents: [{ id: "a-int", mode: "interactive" }, { id: "a-man", mode: "managed" }, { id: "a-off", mode: "stopped" }] }],
  freeAgents: [{ id: "a-free", mode: "managed" }, { id: "a-nomode" }],
  terminals: [{ id: "t-sh", running: true }, { id: "t-dead", running: false }, { id: "t-cli", running: true, launchCli: "claude" }, { id: "t-cli-off", running: false, launchCli: "claude" }],
};
const bound = (kind, ref) => ({ id: "p", kind, ref, x: 0, y: 0, w: 4, h: 14 });

test("bindingState: terminal rows of §4.4", () => {
  assert.equal(bindingState(bound("terminal", "t-sh"), fleet), "terminal-running");
  assert.equal(bindingState(bound("terminal", "t-cli"), fleet), "terminal-running");
  assert.equal(bindingState(bound("terminal", "t-dead"), fleet), "terminal-running", "a plain shell revives on open");
  assert.equal(bindingState(bound("terminal", "t-cli-off"), fleet), "terminal-stopped", "only a CLI terminal has a stopped state");
  assert.equal(bindingState(bound("terminal", "t-gone"), fleet), "terminal-gone");
});

test("bindingState: agent rows of §4.4", () => {
  assert.equal(bindingState(bound("agent", "a-int"), fleet), "agent-interactive");
  assert.equal(bindingState(bound("agent", "a-man"), fleet), "agent-managed");
  assert.equal(bindingState(bound("agent", "a-free"), fleet), "agent-managed", "free agents count");
  assert.equal(bindingState(bound("agent", "a-off"), fleet), "agent-stopped");
  assert.equal(bindingState(bound("agent", "a-nomode"), fleet), "agent-stopped");
  assert.equal(bindingState(bound("agent", "a-gone"), fleet), "agent-gone");
  assert.equal(bindingState(bound("agent", "w1"), fleet), "agent-gone", "a workspace id is not an agent");
  assert.equal(bindingState(bound("pin", "x"), fleet), "");
  assert.equal(bindingState(null, fleet), "");
  assert.equal(bindingState(bound("terminal", "t-sh"), null), "terminal-gone");
});

// §4.5, one test per row. `near` is the observer's word plus the surface
// being visible; the caller runs the policy on every change and at wakeAt.
const NONE = new Set();
const run = (entries, now, pinned, timers) => loadPolicy(entries, now, pinned, timers);

test("loadPolicy: a wrapper loads after it stays near for 300 ms, not before", () => {
  assert.equal(LOAD_DWELL_MS, 300);
  let r = run([{ id: "p1", near: true, loaded: false }], 1000, NONE, {});
  assert.deepEqual(r.load, []);
  assert.equal(r.wakeAt, 1300, "sleep until the dwell ends");
  r = run([{ id: "p1", near: true, loaded: false }], 1100, NONE, r.timers);
  assert.deepEqual(r.load, [], "still dwelling");
  assert.equal(r.wakeAt, 1300, "the deadline does not slide");
  r = run([{ id: "p1", near: true, loaded: false }], 1300, NONE, r.timers);
  assert.deepEqual(r.load, ["p1"]);
  assert.deepEqual(r.timers, {}, "loaded: nothing pending");
  assert.equal(r.wakeAt, 0);
});

test("loadPolicy: leaving the margin cancels a pending load at once (a fly-over loads nothing)", () => {
  let r = run([{ id: "p1", near: true, loaded: false }], 1000, NONE, {});
  r = run([{ id: "p1", near: false, loaded: false }], 1100, NONE, r.timers);
  assert.deepEqual(r.load, []);
  assert.deepEqual(r.timers, {});
  assert.equal(r.wakeAt, 0);
  r = run([{ id: "p1", near: true, loaded: false }], 1200, NONE, r.timers);
  assert.equal(r.wakeAt, 1500, "coming back starts a fresh dwell");
});

test("loadPolicy: a loaded body unloads 5 s after it left, and coming back cancels that", () => {
  assert.equal(UNLOAD_AFTER_MS, 5000);
  let r = run([{ id: "p1", near: false, loaded: true }], 1000, NONE, {});
  assert.deepEqual(r.unload, []);
  assert.equal(r.wakeAt, 6000);
  r = run([{ id: "p1", near: false, loaded: true }], 4000, NONE, r.timers);
  assert.deepEqual(r.unload, [], "hysteresis: still inside the 5 s");
  r = run([{ id: "p1", near: true, loaded: true }], 4500, NONE, r.timers);
  assert.deepEqual(r.timers, {}, "back in view: the unload is dropped");
  assert.equal(r.wakeAt, 0);
  r = run([{ id: "p1", near: false, loaded: true }], 5000, NONE, r.timers);
  r = run([{ id: "p1", near: false, loaded: true }], 10000, NONE, r.timers);
  assert.deepEqual(r.unload, ["p1"]);
  assert.deepEqual(r.timers, {});
});

test("loadPolicy: a pinned panel never unloads; unpinning starts the 5 s from then", () => {
  const pinned = new Set(["p1"]);
  let r = run([{ id: "p1", near: false, loaded: true }], 1000, pinned, {});
  assert.deepEqual(r.unload, []);
  assert.deepEqual(r.timers, {}, "no unload timer while pinned");
  r = run([{ id: "p1", near: false, loaded: true }], 60000, pinned, r.timers);
  assert.deepEqual(r.unload, [], "a minute out of view, still pinned");
  r = run([{ id: "p1", near: false, loaded: true }], 60000, NONE, r.timers);
  assert.equal(r.wakeAt, 65000, "unpinned: the clock starts now");
});

test("loadPolicy: a hidden matrix (every panel reported far) unloads everything after 5 s", () => {
  const far = ["p1", "p2", "p3"].map((id) => ({ id, near: false, loaded: true }));
  let r = run(far, 1000, NONE, {});
  assert.deepEqual(r.unload, []);
  r = run(far, 6000, NONE, r.timers);
  assert.deepEqual(r.unload, ["p1", "p2", "p3"]);
});

test("loadPolicy: mixed panels — wakeAt is the earliest deadline, junk is skipped", () => {
  let r = run([{ id: "a", near: true, loaded: false }, { id: "b", near: false, loaded: true }, { id: "c", near: true, loaded: true }, { near: true }, null], 1000, NONE, {});
  assert.equal(r.wakeAt, 1300);
  assert.deepEqual(Object.keys(r.timers).sort(), ["a", "b"]);
  r = run([{ id: "a", near: true, loaded: true }, { id: "b", near: false, loaded: true }], 1300, NONE, r.timers);
  assert.equal(r.wakeAt, 6000, "a loaded, b still counting down");
  assert.deepEqual(run([], 5, NONE, {}), { load: [], unload: [], timers: {}, wakeAt: 0 });
  assert.deepEqual(run(null, 5, null, null), { load: [], unload: [], timers: {}, wakeAt: 0 });
});

test("suspendedToDispose: past 24 instances the oldest go, past 10 min any goes", () => {
  assert.equal(SUSPENDED_MAX, 24);
  assert.equal(SUSPENDED_TTL_MS, 600000);
  const at = (n) => ({ id: "t" + n, at: 1000 + n });
  const list = Array.from({ length: 26 }, (_, i) => at(i));
  assert.deepEqual(suspendedToDispose(list, 2000), ["t0", "t1"], "two over the cap: the two oldest");
  assert.deepEqual(suspendedToDispose(list.slice(0, 24), 2000), []);
  assert.deepEqual(suspendedToDispose([{ id: "old", at: 0 }, { id: "fresh", at: 500000 }], 600000), ["old"]);
  assert.deepEqual(suspendedToDispose([{ id: "b", at: 20 }, { id: "a", at: 10 }], 30, 1), ["a"], "sorted by age, not by order");
  assert.deepEqual(suspendedToDispose([{ id: "x" }, null, { at: 1 }], 5), []);
});

// The keyboard's grid (plan §4.6 "Focus"): three panels across the top,
// a wide one under the first two, a stack under the third, one alone at
// the bottom left.
//
//   A(0,0 4×8)  B(4,0 4×8)  C(8,0 4×8)
//   D(0,8 8×8)              E(8,8 4×4)
//                           F(8,12 4×4)
//   G(0,16 4×8)
const cell = (id, x, y, w, h) => ({ id, kind: "terminal", ref: "t-" + id, x, y, w, h });
const GRID = [
  cell("A", 0, 0, 4, 8), cell("B", 4, 0, 4, 8), cell("C", 8, 0, 4, 8),
  cell("D", 0, 8, 8, 8), cell("E", 8, 8, 4, 4), cell("F", 8, 12, 4, 4),
  cell("G", 0, 16, 4, 8),
];

test("panelOrder: reading order, junk left out", () => {
  assert.deepEqual(panelOrder([...GRID].reverse()), ["A", "B", "C", "D", "E", "F", "G"]);
  assert.deepEqual(panelOrder([cell("x", 4, 0, 4, 8), { id: "junk", x: 0 }, null, cell("y", 0, 0, 4, 8)]), ["y", "x"]);
  assert.deepEqual(panelOrder(null), []);
  assert.deepEqual([...PANEL_DIRECTIONS], ["left", "right", "up", "down"]);
});

test("neighborPanel: left and right stay on the row and stop at its ends", () => {
  assert.equal(neighborPanel(GRID, "A", "right"), "B");
  assert.equal(neighborPanel(GRID, "B", "right"), "C");
  assert.equal(neighborPanel(GRID, "C", "right"), "", "nothing to the right of the last column");
  assert.equal(neighborPanel(GRID, "B", "left"), "A");
  assert.equal(neighborPanel(GRID, "A", "left"), "");
  assert.equal(neighborPanel(GRID, "D", "right"), "E", "the wide panel shares rows with E and F; E lines up with its top");
  assert.equal(neighborPanel(GRID, "F", "left"), "D");
  assert.equal(neighborPanel(GRID, "G", "right"), "", "a row with one panel: right is a dead end, down and up are not");
});

test("neighborPanel: down and up follow the shared columns, ties go to the best-aligned edge", () => {
  assert.equal(neighborPanel(GRID, "A", "down"), "D");
  assert.equal(neighborPanel(GRID, "B", "down"), "D");
  assert.equal(neighborPanel(GRID, "C", "down"), "E");
  assert.equal(neighborPanel(GRID, "E", "down"), "F");
  assert.equal(neighborPanel(GRID, "F", "up"), "E");
  assert.equal(neighborPanel(GRID, "E", "up"), "C");
  assert.equal(neighborPanel(GRID, "D", "up"), "A", "A and B both touch D; A lines up with its left edge");
  assert.equal(neighborPanel(GRID, "A", "up"), "");
  assert.equal(neighborPanel(GRID, "G", "down"), "");
});

test("neighborPanel: up and down fall back to the nearest panel in that half when no column is shared", () => {
  assert.equal(neighborPanel(GRID, "F", "down"), "G", "F sits over nothing; G is the only panel below");
  assert.equal(neighborPanel(GRID, "G", "up"), "D", "D touches G; A is further up");
  const twoRows = [cell("a", 8, 0, 4, 8), cell("b", 0, 8, 4, 8), cell("c", 4, 8, 4, 8)];
  assert.equal(neighborPanel(twoRows, "a", "down"), "c", "nearest column first");
  assert.equal(neighborPanel(twoRows, "b", "up"), "a");
  assert.equal(neighborPanel(twoRows, "b", "right"), "c");
  assert.equal(neighborPanel(twoRows, "c", "right"), "", "left and right never leave the row");
});

test("neighborPanel: unknown id, unknown direction and junk answer nothing", () => {
  assert.equal(neighborPanel(GRID, "nope", "right"), "");
  assert.equal(neighborPanel(GRID, "A", "diagonal"), "");
  assert.equal(neighborPanel(GRID, "", "right"), "");
  assert.equal(neighborPanel(null, "A", "right"), "");
  assert.equal(neighborPanel([cell("A", 0, 0, 4, 8), { id: "B", x: 4, y: 0 }, null], "A", "right"), "", "a panel without a whole rectangle is not a neighbour");
  assert.equal(neighborPanel([cell("A", 0, 0, 4, 8)], "A", "down"), "", "alone");
});
