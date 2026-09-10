import assert from "node:assert/strict";
import { test } from "node:test";
import { touches } from "./feedReducers.js";
import {
  CANVAS_ZOOM, LOAD_DWELL_MS, MATRIX_EVENTS, MATRIX_KINDS, MATRIX_LIMITS, MATRIX_MODES, PANE_STATES, PANEL_DEFAULT, PANEL_DEFAULT_CANVAS, PANEL_DIRECTIONS,
  SUSPENDED_MAX, SUSPENDED_TTL_MS, TIDY_COLS, TIDY_GAP, UNIT_PX, UNLOAD_AFTER_MS, VIEWPORT_PREFIX, applyMatrixEvent, bindingState,
  canvasToGrid, gridToCanvas, layoutDiff, loadPolicy, neighborPanel, nextSlot, normalizeMatrix, normalizeMatrixDetail,
  normalizeMatrixList, normalizePanel, normalizeViewport, panelDefault, panelOrder, pointerAtZoom, pxToUnits, suspendedToDispose,
  buildRef, hasPane, parseRef, REF_OWNERS, tidyCanvas, validateRef, unitsToPx, validateCompact, validateMode, validateName, validatePanel, validatePlacement, viewportKey, zoomBody,
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
  assert.equal(normalizePanel({ ...panel("p1"), kind: "note", ref: "pin-1" }).kind, "note");
  assert.equal(normalizePanel({ ...panel("p1"), kind: "file", ref: "t:term-1:src/main.go" }).ref, "t:term-1:src/main.go");
  assert.equal(normalizePanel({ ...panel("p1"), kind: "file", ref: "src/main.go" }), null, "a ref the server would refuse is not a panel");
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
  assert.equal(validatePanel({ kind: "pin", ref: "x", x: 0, y: 0, w: 4, h: 8 }), "kind must be agent, terminal, note or file");
  assert.equal(validatePanel({ kind: "agent", ref: "  ", x: 0, y: 0, w: 4, h: 8 }), "ref is required");
  assert.equal(validatePanel({ kind: "agent", ref: "a1", x: 0, y: 0, w: 3, h: 8 }), "w must be at least 4 columns");
  assert.equal(validatePanel({ kind: "terminal", ref: "t1", x: 8, y: 0, w: 4, h: 8 }), "");
  assert.equal(validatePanel(undefined), "kind must be agent, terminal, note or file");
  assert.deepEqual([...MATRIX_KINDS], ["agent", "terminal", "note", "file"]);
  assert.equal(validatePanel({ kind: "note", ref: "pin-1", x: 0, y: 0, w: 4, h: 8 }), "", "a note binds a pin id");
  assert.equal(validatePanel({ kind: "note", ref: " ", x: 0, y: 0, w: 4, h: 8 }), "ref is required");
  assert.equal(validatePanel({ kind: "file", ref: "t:term-1:src/main.go", x: 0, y: 0, w: 4, h: 8 }), "");
  assert.equal(validatePanel({ kind: "file", ref: "t:term-1", x: 0, y: 0, w: 4, h: 8 }), "ref must be <owner>:<id>:<path> with owner t, a or w");
  assert.equal(validatePanel({ kind: "file", ref: " ", x: 0, y: 0, w: 4, h: 8 }), "ref is required");
});

// C3 (docs/plans/matrix-canvas.md §4.2): one place takes a ref apart and one
// place puts it together, so no caller ever splits on ":".
test("parseRef and buildRef: a pin id, or <owner>:<id>:<path> with the desktop's own letters", () => {
  assert.deepEqual(REF_OWNERS, { t: "term", a: "agent", w: "workspace" });
  assert.deepEqual(parseRef("note", " pin-1 "), { kind: "note", pinId: "pin-1" });
  assert.equal(parseRef("note", "  "), null);
  assert.deepEqual(parseRef("file", "t:term-1:src/main.go"), {
    kind: "file", letter: "t", owner: { kind: "term", id: "term-1" }, path: "src/main.go",
  });
  assert.deepEqual(parseRef("file", "a:agent-1:README.md").owner, { kind: "agent", id: "agent-1" });
  assert.deepEqual(parseRef("file", "w:ws-1:docs/a.md").owner, { kind: "workspace", id: "ws-1" });
  // A path keeps its own colons: only the first two separate.
  assert.equal(parseRef("file", "t:term-1:weird:name.txt").path, "weird:name.txt");
  for (const bad of ["t:term-1", "t:term-1:", "t::main.go", "x:term-1:main.go", "src/main.go", ":a:b", ""]) {
    assert.equal(parseRef("file", bad), null, bad);
  }
  assert.equal(buildRef("note", { pinId: "pin-1" }), "pin-1");
  assert.equal(buildRef("file", { owner: { kind: "term", id: "term-1" }, path: "src/main.go" }), "t:term-1:src/main.go");
  assert.equal(buildRef("file", { owner: { kind: "workspace", id: "ws-1" }, path: "a.md" }), "w:ws-1:a.md");
  assert.equal(buildRef("file", { owner: { kind: "term", id: "term-1" }, path: "" }), "");
  assert.equal(buildRef("file", { owner: { kind: "nope", id: "x" }, path: "a.md" }), "");
  assert.equal(buildRef("file", { owner: { kind: "term", id: "has:colon" }, path: "a.md" }), "", "an id with a colon cannot round-trip");
  // Round trip, both ways.
  const ref = buildRef("file", { owner: { kind: "agent", id: "agent-1" }, path: "docs/a:b.md" });
  assert.deepEqual(parseRef("file", ref), { kind: "file", letter: "a", owner: { kind: "agent", id: "agent-1" }, path: "docs/a:b.md" });
  assert.equal(validateRef("file", ref), "");
  assert.equal(validateRef("file", "nope"), "ref must be <owner>:<id>:<path> with owner t, a or w");
  assert.equal(validateRef("note", " "), "ref is required");
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
  assert.equal(validatePanel({ kind: "pin", ref: "t1", x: 0, y: 0, w: 32, h: 42 }, "canvas"), "kind must be agent, terminal, note or file");
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

// C3 (docs/plans/matrix-canvas.md §4.2): a note's binding is a pin, and the
// gone row is the pin's deletion — the store never hears of it (ADR-0108).
test("bindingState: note rows follow the pins list, and nothing is gone before it is read", () => {
  const pins = [{ id: "pin-1", title: "Release notes" }];
  assert.equal(bindingState(bound("note", "pin-1"), { ...fleet, pins }), "note-ready");
  assert.equal(bindingState(bound("note", "pin-2"), { ...fleet, pins }), "note-gone");
  assert.equal(bindingState(bound("note", "pin-2"), { ...fleet, pins: [] }), "note-gone");
  assert.equal(bindingState(bound("note", "pin-2"), fleet), "note-ready", "no pins list yet: not read is not gone");
  assert.equal(bindingState(bound("note", "pin-2"), { ...fleet, pins: null }), "note-ready");
});

test("bindingState: a file's owner is what can be gone, never the file on disk", () => {
  const ref = (r) => ({ id: "p", kind: "file", ref: r, x: 0, y: 0, w: 4, h: 8 });
  assert.equal(bindingState(ref("t:t-sh:src/main.go"), fleet), "file-ready");
  assert.equal(bindingState(ref("a:a-int:README.md"), fleet), "file-ready");
  assert.equal(bindingState(ref("a:a-off:README.md"), fleet), "file-ready", "a stopped agent still owns its folder");
  assert.equal(bindingState(ref("w:w1:docs/a.md"), fleet), "file-ready");
  assert.equal(bindingState(ref("t:t-gone:src/main.go"), fleet), "file-gone");
  assert.equal(bindingState(ref("a:a-gone:README.md"), fleet), "file-gone");
  assert.equal(bindingState(ref("w:w-gone:a.md"), fleet), "file-gone");
  assert.equal(bindingState(ref("t:t-sh:never/written.txt"), fleet), "file-ready", "a missing file is the body's news, not a binding state");
  assert.equal(bindingState(ref("nonsense"), fleet), "file-gone");
  assert.equal(bindingState(ref("t:t-sh:a.md"), null), "file-gone");
});

test("hasPane: only a body that is a terminal takes the pointer and still rules", () => {
  assert.deepEqual([...PANE_STATES], ["terminal-running", "terminal-stopped", "agent-interactive"]);
  for (const state of PANE_STATES) assert.equal(hasPane({ state }), true, state);
  for (const state of ["terminal-gone", "agent-gone", "agent-stopped", "agent-managed", "note-ready", "note-gone", "file-ready", "file-gone"]) {
    assert.equal(hasPane({ state }), false, state);
  }
  assert.equal(hasPane({ state: "terminal-running", pending: true }), false, "a pending wrapper has no pane yet");
  assert.equal(hasPane(null), false);
});

// §4.5, one test per row. `near` is the observer's word plus the surface
// being visible; the caller runs the policy on every change and at wakeAt.
const NONE = new Set();
const run = (entries, now, pinned, timers, view) => loadPolicy(entries, now, pinned, timers, view);

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
  assert.deepEqual(run([], 5, NONE, {}), { load: [], unload: [], timers: {}, wakeAt: 0, band: "live", bodies: {} });
  assert.deepEqual(run(null, 5, null, null), { load: [], unload: [], timers: {}, wakeAt: 0, band: "live", bodies: {} });
});

test("loadPolicy: grid mode passes no view — every loaded body is live, as before canvas mode", () => {
  const r = run([{ id: "a", near: true, loaded: true }, { id: "b", near: false, loaded: false }], 1000, NONE, {});
  assert.deepEqual(r.bodies, { a: "live", b: "off" });
  assert.equal(r.band, "live");
});

// The zoom table (plan docs/plans/matrix-canvas.md §4.3), one test per row.
// `view` is { zoom, band }: the canvas's zoom and the band the last call
// answered with, which is the hysteresis's only memory.
const IN = [{ id: "p1", near: true, loaded: true }];
const OUT = [{ id: "p1", near: false, loaded: false }];
const at = (zoom, band = "live", entries = IN) => run(entries, 1000, NONE, {}, { zoom, band });

test("zoom table: in the band at zoom 1.0 — a live pane, and the only body that takes a pointer", () => {
  assert.deepEqual({ ...CANVAS_ZOOM }, { min: 0.2, max: 1.5, live: 0.8, still: 0.75, plate: 0.4, exact: 1 });
  const r = at(1);
  assert.deepEqual(r.bodies, { p1: "live" });
  assert.equal(r.band, "live");
  assert.equal(pointerAtZoom(1), true);
  assert.equal(pointerAtZoom(1.0001), true, "a d3 transform never lands on exactly 1.0");
  assert.equal(pointerAtZoom(0.99), false);
});

test("zoom table: in the band from 0.8 to 1.0 and above — live and readable, but no pointer", () => {
  for (const z of [0.8, 0.9, 0.99, 1.2, 1.5]) {
    assert.deepEqual(at(z).bodies, { p1: "live" }, "live at " + z);
    assert.equal(pointerAtZoom(z), false, "no pointer at " + z);
  }
  // Zooming *in* breaks xterm's mapping exactly as zooming out does (C0: a
  // ten-character drag at 1.5 returned thirteen), so 1.5 is pointer-inert too.
  assert.equal(zoomBody(1.5, "live").body, "live");
});

test("zoom table: in the band below 0.75 — a text still, no attach", () => {
  assert.deepEqual(at(0.74).bodies, { p1: "still" });
  assert.deepEqual(at(0.5).bodies, { p1: "still" });
  assert.equal(at(0.74).band, "still");
  assert.equal(pointerAtZoom(0.5), false);
});

test("zoom table: below 0.4 — a name-plate, because a still is grey texture down there", () => {
  assert.deepEqual(at(0.39).bodies, { p1: "plate" });
  assert.deepEqual(at(CANVAS_ZOOM.min).bodies, { p1: "plate" });
  assert.equal(at(0.2).band, "still");
  assert.equal(zoomBody(0.3, "live").body, "plate", "the plate wins whatever band the viewer came from");
});

test("zoom table: outside the band at any zoom — the placeholder, and the socket suspends", () => {
  for (const z of [1, 0.8, 0.5, 0.2]) assert.deepEqual(at(z, "live", OUT).bodies, { p1: "off" }, "off at " + z);
  // The 5 s hysteresis is the unload's, unchanged by the zoom.
  let r = run([{ id: "p1", near: false, loaded: true }], 1000, NONE, {}, { zoom: 1, band: "live" });
  assert.deepEqual(r.unload, []);
  assert.deepEqual(r.bodies, { p1: "live" }, "still live while the 5 s runs");
  r = run([{ id: "p1", near: false, loaded: true }], 6000, NONE, r.timers, { zoom: 1, band: "live" });
  assert.deepEqual(r.unload, ["p1"]);
  assert.deepEqual(r.bodies, { p1: "off" }, "the body goes in the same pass that unloads it");
});

test("zoom table: the hysteresis — live above 0.8, still below 0.75, in between whatever it was", () => {
  assert.equal(zoomBody(0.78, "live").band, "live", "coming down from live: still live at 0.78");
  assert.equal(zoomBody(0.78, "still").band, "still", "coming up from still: still still at 0.78");
  assert.equal(zoomBody(0.75, "live").band, "live");
  assert.equal(zoomBody(0.749, "live").band, "still", "below 0.75 it flips");
  assert.equal(zoomBody(0.8, "still").band, "live", "at 0.8 it flips back");
  assert.equal(zoomBody(0.79, "still").band, "still", "0.79 does not: C0 measured nine sockets thrashing there");
  assert.equal(zoomBody(1, undefined).band, "live", "no memory: live");
  assert.equal(zoomBody(undefined, undefined).body, "live", "no zoom: grid mode's answer");
  assert.equal(zoomBody(NaN, "still").band, "live", "junk zoom reads as 1.0");
});

// C3 (docs/plans/matrix-canvas.md §4.2): a body that is not a pane has no
// `term.buffer` to capture, so the still row never applies to it — it renders
// normally from 0.4 up and becomes the name-plate below it. The band it
// reports is still the pane band: the band is what bounds the attaches.
test("zoom table: a body with no pane never goes still — it reads normally from 0.4 up", () => {
  const flat = [{ id: "n1", near: true, loaded: true, pane: false }];
  for (const z of [1.5, 1, 0.8, 0.74, 0.5, 0.4]) {
    assert.deepEqual(at(z, "live", flat).bodies, { n1: "live" }, "readable at " + z);
  }
  assert.deepEqual(at(0.5, "still", flat).bodies, { n1: "live" }, "the still band does not reach it");
  assert.equal(at(0.5, "still", flat).band, "still", "the pane band is unchanged by it");
  assert.equal(zoomBody(0.5, "still", false).body, "live");
  assert.equal(zoomBody(0.5, "still", true).body, "still");
});

test("zoom table: a body with no pane is still a name-plate below 0.4, and still obeys the band", () => {
  const flat = (near, loaded) => [{ id: "n1", near, loaded, pane: false }];
  assert.deepEqual(at(0.39, "still", flat(true, true)).bodies, { n1: "plate" });
  assert.deepEqual(at(CANVAS_ZOOM.min, "live", flat(true, true)).bodies, { n1: "plate" });
  // Outside the band it unmounts like any other body: there is no socket to
  // suspend, and mounting again is a fetch.
  assert.deepEqual(at(1, "live", flat(false, false)).bodies, { n1: "off" });
  const r = run(flat(false, true), 6000, NONE, { n1: { unloadAt: 5000 } }, { zoom: 1, band: "live" });
  assert.deepEqual(r.unload, ["n1"]);
  assert.deepEqual(r.bodies, { n1: "off" });
});

test("zoom table: panes and non-panes on one plane are decided per row", () => {
  const mixed = [{ id: "t1", near: true, loaded: true }, { id: "n1", near: true, loaded: true, pane: false }];
  assert.deepEqual(at(0.5, "live", mixed).bodies, { t1: "still", n1: "live" });
  assert.deepEqual(at(1, "live", mixed).bodies, { t1: "live", n1: "live" });
  assert.deepEqual(at(0.3, "live", mixed).bodies, { t1: "plate", n1: "plate" });
});

test("zoom table: focused and engaged — the pointer rule is what the snap to 1 exists for", () => {
  // The surface animates to 1 before the pane takes keys; the pure part is
  // the question it asks: may this zoom take a pointer?
  assert.equal(pointerAtZoom(0.8), false, "a click here snaps to 1 first");
  assert.equal(pointerAtZoom(CANVAS_ZOOM.exact), true, "and only then does the pointer reach the pane");
});

test("unitsToPx and pxToUnits: one conversion, both ways, whole units either side", () => {
  assert.equal(UNIT_PX, 8);
  assert.deepEqual(unitsToPx({ x: 3, y: -2, w: 32, h: 42 }), { x: 24, y: -16, width: 256, height: 336 });
  assert.deepEqual(unitsToPx(null), { x: 0, y: 0, width: 0, height: 0 });
  assert.deepEqual(pxToUnits({ x: 24, y: -16, width: 256, height: 336 }), { x: 3, y: -2, w: 32, h: 42 });
  assert.deepEqual(pxToUnits({ x: 3, y: -3, width: 260, height: 337 }), { x: 0, y: 0, w: 33, h: 42 }, "a resize does not snap: rounded here");
  assert.deepEqual(pxToUnits({ x: -20, y: -20, width: 256, height: 224 }), { x: -3, y: -3, w: 32, h: 28 }, "half away from zero, like the store");
  assert.deepEqual(pxToUnits(undefined), { x: 0, y: 0, w: 0, h: 0 });
  const rect = { x: -7, y: 11, w: 40, h: 30 };
  assert.deepEqual(pxToUnits(unitsToPx(rect)), rect, "a round trip through pixels is identity");
});

test("tidyCanvas: reading order, sizes kept, three default panels across", () => {
  assert.equal(TIDY_COLS, 98, "three 32-unit panels and the two gutters between them");
  assert.equal(TIDY_GAP, 1);
  const p = (id, x, y, w = 32, h = 42) => ({ id, kind: "terminal", ref: "t-" + id, x, y, w, h });
  const out = tidyCanvas([p("c", 400, 0), p("a", 0, 0), p("b", 200, 0), p("d", 0, 300)]);
  assert.deepEqual(out, [
    { id: "a", x: 0, y: 0, w: 32, h: 42 },
    { id: "b", x: 33, y: 0, w: 32, h: 42 },
    { id: "c", x: 66, y: 0, w: 32, h: 42 },
    { id: "d", x: 0, y: 43, w: 32, h: 42 },
  ], "three across at the default size, then a new row a gap below the tallest");
  const kept = tidyCanvas([p("a", 0, 0, 60, 50), p("b", 100, 0, 32, 28)]);
  assert.deepEqual(kept, [{ id: "a", x: 0, y: 0, w: 60, h: 50 }, { id: "b", x: 61, y: 0, w: 32, h: 28 }], "Tidy arranges, it never resizes");
  const wide = tidyCanvas([p("a", 0, 0, 32, 42), p("wide", 100, 0, 120, 42)]);
  assert.deepEqual(wide[1], { id: "wide", x: 0, y: 43, w: 120, h: 42 }, "a panel wider than the row gets its own row");
  assert.deepEqual(tidyCanvas([]), []);
  assert.deepEqual(tidyCanvas(null), []);
  assert.deepEqual(tidyCanvas([{ id: "junk", x: 0 }, null]), [], "a panel without a whole rectangle cannot be placed");
  // Ties break by id, so two viewers tidying the same matrix agree.
  assert.deepEqual(tidyCanvas([p("z", 0, 0), p("a", 0, 0)]).map((r) => r.id), ["a", "z"]);
  // What Tidy answers is a layout patch: nothing to save when nothing moved.
  assert.deepEqual(layoutDiff(out, tidyCanvas(out), "canvas"), []);
});

test("viewportKey and normalizeViewport: a camera per viewer, never the store", () => {
  assert.equal(VIEWPORT_PREFIX, "picode-matrix-view:");
  assert.equal(viewportKey("m1"), "picode-matrix-view:m1");
  assert.equal(viewportKey(null), "picode-matrix-view:");
  assert.deepEqual(normalizeViewport({ x: -120, y: 40, zoom: 0.5 }), { x: -120, y: 40, zoom: 0.5 });
  assert.deepEqual(normalizeViewport({ x: 0, y: 0, zoom: 9 }), { x: 0, y: 0, zoom: 1.5 }, "clamped into minZoom/maxZoom");
  assert.deepEqual(normalizeViewport({ x: 0, y: 0, zoom: 0.01 }), { x: 0, y: 0, zoom: 0.2 });
  assert.equal(normalizeViewport({ x: 0, y: 0 }), null, "junk: fitView instead");
  assert.equal(normalizeViewport({ x: "1", y: 2, zoom: 1 }), null);
  assert.equal(normalizeViewport(null), null);
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

// The keyboard needs no canvas of its own: neighborPanel is already a 2D
// spatial search, so the arrows work on a free plane exactly as they do on
// the grid — including the negative half of it.
//
//   L(-99,0 32×42)  P(0,0 32×42)  Q(33,0 32×42)
//                   R(0,43 65×42)
const plane = (id, x, y, w = 32, h = 42) => ({ id, kind: "terminal", ref: "t-" + id, x, y, w, h });
const PLANE = [plane("L", -99, 0), plane("P", 0, 0), plane("Q", 33, 0), plane("R", 0, 43, 65, 42)];

test("neighborPanel and panelOrder read canvas units unchanged (the arrows are already 2D)", () => {
  assert.equal(neighborPanel(PLANE, "P", "right"), "Q");
  assert.equal(neighborPanel(PLANE, "P", "left"), "L", "a negative coordinate is the plane's, not a mistake");
  assert.equal(neighborPanel(PLANE, "L", "left"), "");
  assert.equal(neighborPanel(PLANE, "P", "down"), "R");
  assert.equal(neighborPanel(PLANE, "Q", "down"), "R", "R shares columns with both");
  assert.equal(neighborPanel(PLANE, "R", "up"), "P", "P lines up with R's left edge");
  assert.equal(neighborPanel(PLANE, "L", "down"), "R", "no column shared: the nearest panel in that half");
  assert.deepEqual(panelOrder(PLANE), ["L", "P", "Q", "R"]);
  // And through the switch transform: a cell is 8 units wide and 3 tall.
  const moved = gridToCanvas(GRID);
  assert.equal(moved.find((p) => p.id === "B").x, 32);
  assert.equal(neighborPanel(moved, "A", "right"), "B");
  assert.deepEqual(panelOrder(moved), ["A", "B", "C", "D", "E", "F", "G"]);
  // The transform is lossy where the plan says it is: an 8-row panel grows
  // to the 28-unit minimum and overlaps the panel below, so `down` from C
  // skips the panel it now overlaps. Canvas mode allows that; the switch
  // back packs it out (ADR-0113).
  assert.equal(neighborPanel(moved, "C", "down"), "F");
});
