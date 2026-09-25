import assert from "node:assert/strict";
import { test } from "node:test";
import { touches } from "./feedReducers.js";
import {
  CANVAS_ZOOM, LOAD_DWELL_MS, CANVAS_EVENTS, CANVAS_KINDS, CANVAS_LIMITS, PANE_STATES, PANEL_DEFAULT_CANVAS, PANEL_DIRECTIONS,
  SUSPENDED_MAX, SUSPENDED_TTL_MS, TIDY_COLS, TIDY_GAP, UNIT_PX, UNLOAD_AFTER_MS, VIEWPORT_PREFIX, applyCanvasEvent, bindingState,
  gridToCanvas, layoutDiff, loadPolicy, neighborPanel, nextSlot, normalizeCanvas, normalizeCanvasDetail,
  placementRect,
  normalizeCanvasList, normalizePanel, normalizeViewport, panelOrder, pointerAtZoom, pxToUnits, suspendedToDispose,
  buildRef, gitTouches, hasPane, parseRef, REF_OWNERS, tidyCanvas, validateRef, unitsToPx, validateCompact, validateName, validatePanel, validatePlacement, viewportKey, zoomBody,
  CHAT_LIVE_MAX, CHAT_STATES, chatBudget, hasChat,
  EDGE_KINDS, edgeEndpoints, normalizeEdge, normalizeEdgeList, validateEdge,
} from "./canvas.js";

const summary = (id, name, extra = {}) => ({
  id, name, compact: "vertical", createdAt: "2026-09-09T10:00:00Z", updatedAt: "2026-09-09T10:00:00Z", panelCount: 0, ...extra,
});
const panel = (id, extra = {}) => ({ id, kind: "terminal", ref: "term-" + id, x: 0, y: 0, w: 32, h: 42, createdAt: "2026-09-09T10:00:00Z", content: "", ...extra });

test("limits and event types are the server's (ADR-0108, ADR-0116, ADR-0118)", () => {
  assert.deepEqual({ ...CANVAS_LIMITS }, {
    canvases: 64, panels: 500, name: 80, text: 2000,
    canvasMinW: 32, canvasMinH: 28, canvasMax: 4096, canvasCoord: 100000, edges: 1000,
  });
  assert.equal(UNIT_PX, 8);
  assert.deepEqual([...CANVAS_EVENTS], ["canvas.created", "canvas.updated", "canvas.layout", "canvas.panel.added", "canvas.panel.content", "canvas.panel.removed", "canvas.edge.added", "canvas.edge.removed", "canvas.deleted"]);
  assert.equal(CANVAS_EVENTS.length, 9, "the mode event went with the engine it announced (ADR-0118); a text panel's words announce their own (2026-09-12)");
  for (const type of CANVAS_EVENTS) assert.equal(touches({ type }, ["canvas"]), true, type);
  // Only a session has a mailbox: a note, a file or a diff cannot hold a
  // contact, so it cannot be an endpoint (ADR-0116).
  assert.deepEqual([...EDGE_KINDS], ["agent", "terminal"]);
});

test("normalizeCanvas keeps the summary fields and drops junk", () => {
  const m = normalizeCanvas({ id: "m1", name: "Ops", compact: "none", createdAt: "a", updatedAt: "b", panelCount: 3, panels: [{}], extra: 1 });
  assert.deepEqual(m, { id: "m1", name: "Ops", compact: "none", createdAt: "a", updatedAt: "b", panelCount: 3 });
  assert.equal(normalizeCanvas({ id: "m1", name: "Ops", compact: "diagonal" }).compact, "vertical");
  assert.equal(normalizeCanvas({ id: "m1", name: "Ops", mode: "canvas" }).mode, undefined, "the dropped mode column is not a summary field (ADR-0118)");
  assert.equal(normalizeCanvas({ id: "m1", name: "Ops", panelCount: -2 }).panelCount, 0);
  assert.equal(normalizeCanvas({ id: "m1", name: "Ops", panelCount: "7" }).panelCount, 0);
  assert.equal(normalizeCanvas({ name: "Ops" }), null);
  assert.equal(normalizeCanvas({ id: "m1" }), null);
  assert.equal(normalizeCanvas(null), null);
  assert.deepEqual(normalizeCanvasList({ canvases: [summary("b", "b"), null, { id: "x" }] }), [summary("b", "b")]);
  assert.deepEqual(normalizeCanvasList({ matrices: [summary("b", "b")] }), [], "the list payload key is `canvases`, and the old one is not read (ADR-0118)");
  assert.deepEqual(normalizeCanvasList(null), []);
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
  const d = normalizeCanvasDetail({ ...summary("m1", "Ops", { panelCount: 9 }), panels: [panel("p1"), { bad: 1 }, panel("p2")] });
  assert.deepEqual(d.canvas, summary("m1", "Ops", { panelCount: 2 }));
  assert.deepEqual(d.panels.map((p) => p.id), ["p1", "p2"]);
  assert.deepEqual(normalizeCanvasDetail(summary("m1", "Ops")).panels, []);
  assert.equal(normalizeCanvasDetail({ panels: [] }), null);
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
    [{ x: 0.5, y: 0, w: 32, h: 42 }, "x must be a whole number"],
    [{ x: 0, y: 0, w: 32 }, "h must be a whole number"],
    [{ x: 64, y: 0, w: 32, h: 42 }, ""],
    [{ x: 0, y: 400, w: 96, h: 300 }, ""],
  ];
  for (const [rect, want] of cases) assert.equal(validatePlacement(rect), want, JSON.stringify(rect));
  assert.equal(validatePlacement(null), "x must be a whole number");
  assert.equal(validatePanel({ kind: "pin", ref: "x", x: 0, y: 0, w: 32, h: 42 }), "kind must be agent, terminal, note, file, diff or text");
  assert.equal(validatePanel({ kind: "agent", ref: "  ", x: 0, y: 0, w: 32, h: 42 }), "ref is required");
  assert.equal(validatePanel({ kind: "agent", ref: "a1", x: 0, y: 0, w: 31, h: 42 }), "w must be at least 32 canvas units");
  assert.equal(validatePanel({ kind: "terminal", ref: "t1", x: 64, y: 0, w: 32, h: 42 }), "");
  assert.equal(validatePanel(undefined), "kind must be agent, terminal, note, file, diff or text");
  assert.deepEqual([...CANVAS_KINDS], ["agent", "terminal", "note", "file", "diff", "text"]);
  // Text is the one kind that binds nothing: the store mints its ref, so a
  // caller that sends one is refused in the server's words.
  assert.equal(validatePanel({ kind: "text", x: 0, y: 0, w: 32, h: 42 }), "", "a text panel needs no ref");
  assert.equal(validatePanel({ kind: "text", ref: "pin-1", x: 0, y: 0, w: 32, h: 42 }), "a text panel takes no ref");
  assert.equal(validatePanel({ kind: "text", x: 0, y: 0, w: 31, h: 42 }), "w must be at least 32 canvas units", "and it is still a rectangle");
  assert.equal(validatePanel({ kind: "note", ref: "pin-1", x: 0, y: 0, w: 32, h: 42 }), "", "a note binds a pin id");
  assert.equal(validatePanel({ kind: "note", ref: " ", x: 0, y: 0, w: 32, h: 42 }), "ref is required");
  assert.equal(validatePanel({ kind: "file", ref: "t:term-1:src/main.go", x: 0, y: 0, w: 32, h: 42 }), "");
  assert.equal(validatePanel({ kind: "file", ref: "t:term-1", x: 0, y: 0, w: 32, h: 42 }), "ref must be <owner>:<id>:<path> with owner t, a or w");
  assert.equal(validatePanel({ kind: "file", ref: " ", x: 0, y: 0, w: 32, h: 42 }), "ref is required");
  assert.equal(validatePanel({ kind: "diff", ref: "w:ws-1:docs/a.md", x: 0, y: 0, w: 32, h: 42 }), "");
  assert.equal(validatePanel({ kind: "diff", ref: "docs/a.md", x: 0, y: 0, w: 32, h: 42 }), "ref must be <owner>:<id>:<path> with owner t, a or w");
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
  assert.deepEqual(parseRef("diff", "t:term-1:src/main.go").owner, { kind: "term", id: "term-1" }, "a diff takes the same shape");
  assert.equal(buildRef("diff", { owner: { kind: "term", id: "term-1" }, path: "a.md" }), "t:term-1:a.md");
  assert.equal(validateRef("note", " "), "ref is required");
});

test("created and updated keep the list sorted by name and follow a loaded canvas", () => {
  let s = { list: [summary("m2", "beta")], byId: {} };
  s = applyCanvasEvent(s, { type: "canvas.created", data: summary("m1", "Alpha") });
  assert.deepEqual(s.list.map((m) => m.id), ["m1", "m2"]);
  s = applyCanvasEvent(s, { type: "canvas.updated", data: summary("m1", "zeta", { updatedAt: "t2" }) });
  assert.deepEqual(s.list.map((m) => m.name), ["beta", "zeta"]);
  assert.equal(s.list[1].updatedAt, "t2");
  // An updated summary for a canvas the list never saw is complete: inserted.
  s = applyCanvasEvent(s, { type: "canvas.updated", data: summary("m3", "gamma") });
  assert.deepEqual(s.list.map((m) => m.name), ["beta", "gamma", "zeta"]);
  // A replayed created event does not duplicate.
  s = applyCanvasEvent(s, { type: "canvas.created", data: summary("m3", "gamma") });
  assert.equal(s.list.length, 3);
  // Same name: creation order decides.
  s = applyCanvasEvent(s, { type: "canvas.created", data: summary("m0", "Beta", { createdAt: "2026-09-09T09:00:00Z" }) });
  assert.deepEqual(s.list.map((m) => m.id), ["m0", "m2", "m3", "m1"]);
  // A loaded canvas follows; its panelCount stays what the panels say.
  const loaded = { list: [summary("m1", "Ops", { panelCount: 1 })], byId: { m1: { canvas: summary("m1", "Ops", { panelCount: 1 }), panels: [panel("p1")] } } };
  const next = applyCanvasEvent(loaded, { type: "canvas.updated", data: summary("m1", "Ops board", { compact: "none", panelCount: 7, updatedAt: "t2" }) });
  assert.equal(next.byId.m1.canvas.name, "Ops board");
  assert.equal(next.byId.m1.canvas.compact, "none");
  assert.equal(next.byId.m1.canvas.panelCount, 1);
  assert.equal(next.byId.m1.canvas.updatedAt, "t2");
  assert.deepEqual(next.byId.m1.panels, [panel("p1")]);
  assert.equal(next.list[0].name, "Ops board");
  // Junk, unknown types and other canvases leave the same object.
  assert.equal(applyCanvasEvent(loaded, { type: "canvas.created", data: { name: "no id" } }), loaded);
  assert.equal(applyCanvasEvent(loaded, { type: "pin.created", data: { id: "m1" } }), loaded);
  assert.equal(applyCanvasEvent(loaded, { type: "canvas.layout", data: { id: "other", updatedAt: "t9", panels: [] } }), loaded);
  assert.equal(applyCanvasEvent(loaded, { type: "canvas.panel.removed", data: { id: "m1", updatedAt: "t9" } }), loaded);
  assert.deepEqual(applyCanvasEvent(undefined, { type: "canvas.created", data: summary("m1", "Ops") }).list.map((m) => m.id), ["m1"]);
});

test("layout moves exactly the subset it carries", () => {
  const s = {
    list: [summary("m1", "Ops", { panelCount: 2 })],
    byId: { m1: { canvas: summary("m1", "Ops", { panelCount: 2 }), panels: [panel("p1"), panel("p2", { x: 32 })] } },
  };
  const ev = {
    type: "canvas.layout",
    data: { id: "m1", updatedAt: "t2", panels: [{ id: "p1", x: -16, y: 43, w: 64, h: 60 }, { id: "p9", x: 0, y: 0, w: 32, h: 42 }, { id: "p2", x: 200, y: 0, w: 4, h: 8 }] },
  };
  const next = applyCanvasEvent(s, ev);
  // p9 is not here (its panel.added frame comes on its own); p2's frame is
  // under the plane's minimum and cannot be honest — both are left alone.
  assert.deepEqual(next.byId.m1.panels, [panel("p1", { x: -16, y: 43, w: 64, h: 60 }), panel("p2", { x: 32 })]);
  assert.equal(next.byId.m1.canvas.updatedAt, "t2");
  assert.equal(next.list[0].updatedAt, "t2");
  assert.equal(next.list[0].panelCount, 2);
  assert.deepEqual(s.byId.m1.panels[0], panel("p1"), "the previous state is not mutated");
  // Not loaded: only the summary's updatedAt moves.
  const unloaded = applyCanvasEvent({ list: s.list, byId: {} }, ev);
  assert.equal(unloaded.list[0].updatedAt, "t2");
  assert.deepEqual(unloaded.byId, {});
});

test("panels added and removed, the counts follow on both shapes", () => {
  let s = { list: [summary("m1", "Ops"), summary("m2", "Two", { panelCount: 3 })], byId: { m1: { canvas: summary("m1", "Ops"), panels: [] } } };
  s = applyCanvasEvent(s, { type: "canvas.panel.added", data: { id: "m1", updatedAt: "t2", panel: { ...panel("p1"), junk: 1 } } });
  assert.deepEqual(s.byId.m1.panels, [panel("p1")]);
  assert.equal(s.byId.m1.canvas.panelCount, 1);
  assert.equal(s.byId.m1.canvas.updatedAt, "t2");
  assert.equal(s.list[0].panelCount, 1);
  assert.equal(s.list[0].updatedAt, "t2");
  // The same panel again (a replay) is a replacement, not a second slot.
  s = applyCanvasEvent(s, { type: "canvas.panel.added", data: { id: "m1", updatedAt: "t2", panel: panel("p1", { x: 32 }) } });
  assert.deepEqual(s.byId.m1.panels, [panel("p1", { x: 32 })]);
  assert.equal(s.list[0].panelCount, 1);
  // Not loaded: the summary count moves by one; nothing is loaded by an event.
  s = applyCanvasEvent(s, { type: "canvas.panel.added", data: { id: "m2", updatedAt: "t3", panel: panel("p5") } });
  assert.equal(s.list[1].panelCount, 4);
  assert.equal(s.list[1].updatedAt, "t3");
  assert.equal(s.byId.m2, undefined);
  s = applyCanvasEvent(s, { type: "canvas.panel.removed", data: { id: "m2", updatedAt: "t4", panelId: "p5" } });
  assert.equal(s.list[1].panelCount, 3);
  assert.equal(s.list[1].updatedAt, "t4");
  s = applyCanvasEvent(s, { type: "canvas.panel.removed", data: { id: "m1", updatedAt: "t5", panelId: "p1" } });
  assert.deepEqual(s.byId.m1.panels, []);
  assert.equal(s.byId.m1.canvas.panelCount, 0);
  assert.equal(s.list[0].panelCount, 0);
  assert.equal(s.list[0].updatedAt, "t5");
  // A panel that cannot be placed is ignored; a count never goes below zero.
  assert.equal(applyCanvasEvent(s, { type: "canvas.panel.added", data: { id: "m1", updatedAt: "t6", panel: { id: "p2" } } }), s);
  const zero = applyCanvasEvent({ list: [summary("m3", "z")], byId: {} }, { type: "canvas.panel.removed", data: { id: "m3", updatedAt: "t", panelId: "x" } });
  assert.equal(zero.list[0].panelCount, 0);
});

test("deleted drops the summary and the loaded canvas", () => {
  const s = {
    list: [summary("m1", "Ops"), summary("m2", "Two")],
    byId: { m1: { canvas: summary("m1", "Ops"), panels: [panel("p1")] }, m2: { canvas: summary("m2", "Two"), panels: [] } },
  };
  const next = applyCanvasEvent(s, { type: "canvas.deleted", data: { id: "m1" } });
  assert.deepEqual(next.list.map((m) => m.id), ["m2"]);
  assert.deepEqual(Object.keys(next.byId), ["m2"]);
  assert.equal(applyCanvasEvent(next, { type: "canvas.deleted", data: { id: "m1" } }), next);
  assert.equal(s.list.length, 2, "the previous state is not mutated");
});

// ---- the surface's arithmetic (plan §4.3–§4.6) ----------------------------

const rect = (id, x, y, w = 32, h = 42) => ({ id, x, y, w, h });

test("nextSlot: the first free slot scanning rows, and no column to wrap at", () => {
  assert.deepEqual(PANEL_DEFAULT_CANVAS, { w: 32, h: 42 });
  assert.deepEqual(nextSlot([]), { x: 0, y: 0 });
  assert.deepEqual(nextSlot([rect("a", 0, 0)]), { x: 32, y: 0 }, "beside the first panel");
  assert.deepEqual(nextSlot([rect("a", 0, 0), rect("b", 64, 0)]), { x: 32, y: 0 }, "the gap between two panels is taken first");
  assert.deepEqual(nextSlot([rect("a", 0, 0), rect("b", 32, 0), rect("c", 64, 0)]), { x: 96, y: 0 }, "a full row is not full: the plane has no column cap");
  assert.deepEqual(nextSlot([rect("a", 0, 0, 320, 42)], 96, 42), { x: 320, y: 0 }, "a wide panel is no wall either — nothing wraps under it");
  assert.deepEqual(nextSlot([rect("a", 0, 0), rect("b", 32, 0, 64, 28)], 32, 28), { x: 96, y: 0 }, "the first free x of the first row wins");
  assert.deepEqual(nextSlot([rect("a", 0, 0), { id: "junk", x: 1.5 }]), { x: 32, y: 0 }, "a panel without a whole rectangle is ignored");
  // The negative half of the plane is not a special case: the scan starts at
  // the topmost edge any panel has, which may be above the origin.
  assert.deepEqual(nextSlot([rect("a", 0, -80, 32, 42)]), { x: 0, y: -38 });
  assert.deepEqual(nextSlot([rect("a", 0, 0)], 0, 0), { x: 32, y: 0 }, "a nonsense size still lands somewhere free");
});

test("layoutDiff: the changed rectangles of panels present in both layouts", () => {
  const prev = [rect("a", 0, 0), rect("b", 33, 0), rect("c", 66, 0)];
  assert.deepEqual(layoutDiff(prev, prev), []);
  assert.deepEqual(layoutDiff(prev, [rect("a", 0, 43), rect("b", 33, 0), rect("c", 66, 0)]), [{ id: "a", x: 0, y: 43, w: 32, h: 42 }]);
  assert.deepEqual(layoutDiff(prev, [rect("a", 0, 0), rect("b", 33, 0, 64, 42), rect("c", 66, 43)]), [{ id: "b", x: 33, y: 0, w: 64, h: 42 }, { id: "c", x: 66, y: 43, w: 32, h: 42 }]);
  assert.deepEqual(layoutDiff(prev, [...prev, rect("new", 0, 43)]), [], "an added panel is its own POST");
  assert.deepEqual(layoutDiff(prev, [rect("a", 0, 0)]), [], "a removed panel is its own DELETE");
  assert.deepEqual(layoutDiff(prev, [{ ...rect("a", 0, 43), kind: "terminal", ref: "t1", extra: 1 }]), [{ id: "a", x: 0, y: 43, w: 32, h: 42 }], "only the rectangle travels");
  assert.deepEqual(layoutDiff(prev, [rect("a", 0, 43, 31, 42), rect("b", 33, 43)]), [{ id: "b", x: 33, y: 43, w: 32, h: 42 }], "an invalid rectangle is left out, the rest still goes");
  assert.deepEqual(layoutDiff(null, null), []);
});

// ---- the plane's rules, the only ones there are (ADR-0113, ADR-0118) -----

test("validate* judge every rectangle by the plane's one set of rules", () => {
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
  for (const [r, want] of cases) assert.equal(validatePlacement(r), want, JSON.stringify(r));
  // There is no second engine to fall through to, and no mode argument to
  // ask for one: a rectangle that was a legal 4×8 cell before ADR-0118 is
  // refused, and a caller that passes a mode anyway is ignored.
  assert.equal(validatePlacement({ x: 0, y: 0, w: 4, h: 8 }), "w must be at least 32 canvas units");
  assert.equal(validatePlacement({ x: 0, y: 0, w: 4, h: 8 }, "grid"), "w must be at least 32 canvas units");
  assert.equal(validatePlacement({ x: -1, y: 0, w: 32, h: 42 }), "", "a negative coordinate is the plane's, not a mistake");
  assert.equal(validatePanel({ kind: "terminal", ref: "t1", x: -40, y: -40, w: 32, h: 28 }), "");
  assert.equal(validatePanel({ kind: "terminal", ref: "t1", x: 0, y: 0, w: 4, h: 8 }), "w must be at least 32 canvas units");
  assert.equal(validatePanel({ kind: "pin", ref: "t1", x: 0, y: 0, w: 32, h: 42 }), "kind must be agent, terminal, note, file, diff or text");
  // layoutDiff applies the same rules, or every move on the plane would be
  // dropped.
  const prev = [rect("a", 0, 0, 32, 42)];
  assert.deepEqual(layoutDiff(prev, [rect("a", -8, -8, 40, 40)]), [{ id: "a", x: -8, y: -8, w: 40, h: 40 }]);
  assert.deepEqual(layoutDiff(prev, [rect("a", -8, -8, 4, 8)]), [], "a rectangle the plane refuses is left out");
});

// The arithmetic migration 045 applied once (ADR-0113, ADR-0118), kept as
// the record of that conversion — the fixtures are the store tests'. No
// caller runs it any more, and there is no way back.
test("gridToCanvas: a cell was 8 units wide and 3 tall, then the minimums", () => {
  assert.deepEqual(gridToCanvas([rect("a", 0, 0, 4, 14), rect("b", 4, 0, 8, 8), rect("c", 0, 14, 12, 40)]), [
    { id: "a", x: 0, y: 0, w: 32, h: 42 },
    { id: "b", x: 32, y: 0, w: 64, h: 28 }, // 8×3 = 24, clamped to the 28-unit minimum
    { id: "c", x: 0, y: 42, w: 96, h: 120 },
  ]);
  assert.deepEqual(gridToCanvas([{ id: "junk", x: 1.5, y: 0, w: 4, h: 8 }]), []);
  assert.deepEqual(gridToCanvas(null), []);
  // A 4×14 panel — the old default — is today's default panel, which is why
  // a converted board still looks like the board it was.
  assert.deepEqual(gridToCanvas([{ id: "p", x: 0, y: 0, w: 4, h: 14 }])[0], { id: "p", x: 0, y: 0, ...PANEL_DEFAULT_CANVAS });
  // The clamps are the plane's own limits, so everything it answers is
  // placeable.
  for (const p of gridToCanvas([rect("a", 0, 0, 4, 14), rect("b", 4, 0, 8, 8), rect("c", 0, 14, 12, 40)])) {
    assert.equal(validatePlacement(p), "", JSON.stringify(p));
  }
});

// §4.4, one assertion per row.
const fleet = {
  workspaces: [{ id: "w1", name: "App", agents: [{ id: "a-int", mode: "interactive" }, { id: "a-man", mode: "managed" }, { id: "a-off", mode: "stopped" }] }],
  freeAgents: [{ id: "a-free", mode: "managed" }, { id: "a-nomode" }],
  terminals: [{ id: "t-sh", running: true }, { id: "t-dead", running: false }, { id: "t-cli", running: true, launchCli: "claude" }, { id: "t-cli-off", running: false, launchCli: "claude" }],
};
const bound = (kind, ref) => ({ id: "p", kind, ref, x: 0, y: 0, w: 32, h: 42 });

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
  const ref = (r, kind = "file") => ({ id: "p", kind, ref: r, x: 0, y: 0, w: 32, h: 42 });
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
  // A diff answers the same rows: nothing about "does this path have
  // changes" is a binding state — the body says *No changes in this file.*
  assert.equal(bindingState(ref("t:t-sh:src/main.go", "diff"), fleet), "diff-ready");
  assert.equal(bindingState(ref("t:t-gone:src/main.go", "diff"), fleet), "diff-gone");
  assert.equal(bindingState(ref("w:w1:docs/a.md", "diff"), fleet), "diff-ready");
});

// A diff refetches on the fleet watcher's `git.updated` for its owner's
// folder, the way the Inspector does — never on a timer.
test("gitTouches: a git.updated reaches a diff whose owner works in that tree", () => {
  assert.equal(gitTouches("/repo", "/repo"), true);
  assert.equal(gitTouches("/repo/web", "/repo"), true, "the owner works inside the folder the watcher named");
  assert.equal(gitTouches("/repo", "/repo/web"), true, "and the other way round");
  assert.equal(gitTouches("/repo/", "/repo"), true, "a trailing slash is not a different folder");
  assert.equal(gitTouches("/repo-two", "/repo"), false, "a prefix is not a parent");
  assert.equal(gitTouches("/other", "/repo"), false);
  assert.equal(gitTouches("", "/repo"), false);
  assert.equal(gitTouches("/repo", ""), false);
  assert.equal(gitTouches(null, undefined), false);
});

test("hasPane: only a body that is a terminal takes the pointer and still rules", () => {
  assert.deepEqual([...PANE_STATES], ["terminal-running", "terminal-stopped", "agent-interactive"]);
  for (const state of PANE_STATES) assert.equal(hasPane({ state }), true, state);
  for (const state of ["terminal-gone", "agent-gone", "agent-stopped", "agent-managed", "note-ready", "note-gone", "file-ready", "file-gone", "diff-ready", "diff-gone"]) {
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

test("loadPolicy: a hidden canvas (every panel reported far) unloads everything after 5 s", () => {
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

test("loadPolicy: a caller that passes no view — every loaded body is live", () => {
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
  assert.equal(zoomBody(undefined, undefined).body, "live", "no zoom: the unzoomed answer");
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

// Phase 4 (docs/plans/matrix-app.md §2.4/§4.5): a chat body — a managed
// agent's live conversation — is not a pane (no cell, so no still and no
// pointer rule) and is not cheap either (a socket and a transcript window).
// One test per row of its table.
test("hasChat: a managed agent's body is the one that holds a socket", () => {
  assert.deepEqual([...CHAT_STATES], ["agent-managed"]);
  assert.equal(hasChat({ state: "agent-managed" }), true);
  assert.equal(hasPane({ state: "agent-managed" }), false, "no cell: the still and pointer rules do not bind it");
  for (const state of ["terminal-running", "terminal-stopped", "agent-interactive", "agent-stopped", "agent-gone", "note-ready", "file-ready", "diff-ready"]) {
    assert.equal(hasChat({ state }), false, state);
  }
  assert.equal(hasChat({ state: "agent-managed", pending: true }), false, "a pending wrapper has no socket yet");
  assert.equal(hasChat(null), false);
});

const chat = (id, near = true, loaded = true) => ({ id, near, loaded, pane: false, chat: true });

test("chat body: in the band at zoom 0.4 and above it is live — socket open, conversation rendered", () => {
  for (const z of [1.5, 1, 0.8, 0.74, 0.5, 0.4]) {
    assert.deepEqual(at(z, "live", [chat("c1")]).bodies, { c1: "live" }, "live at " + z);
  }
  assert.deepEqual(at(0.5, "still", [chat("c1")]).bodies, { c1: "live" }, "the pane's still band does not reach it");
  assert.deepEqual(run([chat("c1")], 1000, NONE, {}).bodies, { c1: "live" }, "no view passed: live");
});

test("chat body: below zoom 0.4 it is a name-plate — and a plate holds no socket", () => {
  assert.deepEqual(at(0.39, "live", [chat("c1")]).bodies, { c1: "plate" });
  assert.deepEqual(at(CANVAS_ZOOM.min, "still", [chat("c1")]).bodies, { c1: "plate" });
  assert.notEqual(at(0.39, "live", [chat("c1")]).bodies.c1, "live", "the socket follows the body, and this one is not live");
});

test("chat body: outside the band it unloads after the same 5 s the panes get", () => {
  const r0 = run([chat("c1", false, true)], 1000, NONE, {});
  assert.deepEqual(r0.unload, [], "the hysteresis has not run out");
  assert.equal(r0.wakeAt, 1000 + UNLOAD_AFTER_MS);
  assert.deepEqual(r0.bodies, { c1: "live" }, "still live, still connected, until it does");
  const r1 = run([chat("c1", false, true)], 6000, NONE, r0.timers);
  assert.deepEqual(r1.unload, ["c1"]);
  assert.deepEqual(r1.bodies, { c1: "off" }, "unmounted: no socket, no transcript");
  assert.deepEqual(r1.timers, {}, "and its liveAt is forgotten, so coming back makes it the newest");
});

test("chatBudget: the most recently arrived keep the sockets; the rest are parked", () => {
  const live = [{ id: "a", at: 30 }, { id: "b", at: 10 }, { id: "c", at: 20 }];
  assert.deepEqual(chatBudget(live, 2), { keep: ["a", "c"], park: ["b"] });
  assert.deepEqual(chatBudget(live, 9), { keep: ["a", "c", "b"], park: [] });
  assert.deepEqual(chatBudget(live, 0), { keep: [], park: ["a", "c", "b"] });
  assert.deepEqual(chatBudget([{ id: "y", at: 5 }, { id: "x", at: 5 }], 1), { keep: ["x"], park: ["y"] }, "a tie breaks by id, not by tick order");
  assert.deepEqual(chatBudget([{ id: "" }, null, { at: 1 }], 4), { keep: [], park: [] });
  assert.deepEqual(chatBudget(null), { keep: [], park: [] });
});

test("chat body: past the cap the oldest arrivals go quiet, and a freed slot is taken back", () => {
  assert.equal(CHAT_LIVE_MAX, 12);
  const many = [];
  for (let i = 0; i < CHAT_LIVE_MAX + 3; i++) many.push(chat("c" + i));
  // All fifteen arrive together: the tie breaks by id, so c0..c11 (string
  // order c0, c1, c10, c11, c12 …) is not the answer — the ids that sort
  // first are. What matters is that exactly three are quiet and the rest live.
  const r = run(many, 1000, NONE, {});
  const quiet = Object.keys(r.bodies).filter((k) => r.bodies[k] === "quiet");
  const liveIds = Object.keys(r.bodies).filter((k) => r.bodies[k] === "live");
  assert.equal(liveIds.length, CHAT_LIVE_MAX, "twelve sockets, never thirteen");
  assert.equal(quiet.length, 3);
  // The parked ones keep their arrival time, so the order does not thrash.
  const again = run(many, 1100, NONE, r.timers);
  assert.deepEqual(Object.keys(again.bodies).filter((k) => again.bodies[k] === "quiet"), quiet, "same three, one tick later");
  // A newcomer is newer than every one of them and takes a slot from the oldest.
  const withNew = run([...many, chat("zz")], 2000, NONE, again.timers);
  assert.equal(withNew.bodies.zz, "live", "the panel the reader just scrolled to gets the socket");
  assert.equal(Object.keys(withNew.bodies).filter((k) => withNew.bodies[k] === "live").length, CHAT_LIVE_MAX);
  const parked = Object.keys(withNew.bodies).filter((k) => withNew.bodies[k] === "quiet");
  assert.equal(parked.length, 4, "sixteen in the band, twelve sockets");
  // One panel leaves the band and unloads: its slot frees for a parked one.
  const left = Object.keys(withNew.bodies).find((k) => withNew.bodies[k] === "live" && k !== "zz");
  const after = [...many.map((e) => (e.id === left ? { id: left, near: false, loaded: false, pane: false, chat: true } : e)), chat("zz")];
  const freed = run(after, 8000, NONE, withNew.timers);
  assert.equal(freed.bodies[left], "off");
  const stillParked = Object.keys(freed.bodies).filter((k) => freed.bodies[k] === "quiet");
  assert.equal(stillParked.length, 3, "one socket freed, one parked panel took it");
  assert.ok(stillParked.every((id) => parked.includes(id)), "and it came from the parked set, not from a reshuffle");
});

test("chat body: a canvas under the cap never pays for it, and panes are counted apart", () => {
  const mixed = [chat("c1"), chat("c2"), { id: "t1", near: true, loaded: true }, { id: "n1", near: true, loaded: true, pane: false }];
  const r = run(mixed, 1000, NONE, {});
  assert.deepEqual(r.bodies, { c1: "live", c2: "live", t1: "live", n1: "live" }, "two chats, a pane and a note: nothing is capped");
  assert.deepEqual(at(0.5, "live", mixed).bodies, { c1: "live", c2: "live", t1: "still", n1: "live" }, "the still is the pane's alone");
  // A pane is never counted against the chat cap: twenty terminals are the
  // band's business, not this rule's.
  const panes = [];
  for (let i = 0; i < 20; i++) panes.push({ id: "t" + i, near: true, loaded: true });
  const p = run(panes, 1000, NONE, {});
  assert.equal(Object.values(p.bodies).filter((b) => b === "quiet").length, 0);
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
  // Ties break by id, so two viewers tidying the same canvas agree.
  assert.deepEqual(tidyCanvas([p("z", 0, 0), p("a", 0, 0)]).map((r) => r.id), ["a", "z"]);
  // What Tidy answers is a layout patch: nothing to save when nothing moved.
  assert.deepEqual(layoutDiff(out, tidyCanvas(out)), []);
});

test("viewportKey and normalizeViewport: a camera per viewer, never the store", () => {
  assert.equal(VIEWPORT_PREFIX, "picode-canvas-view:");
  assert.equal(viewportKey("m1"), "picode-canvas-view:m1");
  assert.equal(viewportKey(null), "picode-canvas-view:");
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

// The keyboard's board (plan §4.6 "Focus"): three panels across the top, a
// wide one under the first two, a stack under the third, one alone at the
// bottom left. The arrows are pure 2D arithmetic and never validate a
// rectangle, so this fixture uses small numbers to keep the picture legible;
// the test below runs the same board in canvas units.
//
//   A(0,0 4×8)  B(4,0 4×8)  C(8,0 4×8)
//   D(0,8 8×8)              E(8,8 4×4)
//                           F(8,12 4×4)
//   G(0,16 4×8)
const cell = (id, x, y, w, h) => ({ id, kind: "terminal", ref: "t-" + id, x, y, w, h });
const BOARD = [
  cell("A", 0, 0, 4, 8), cell("B", 4, 0, 4, 8), cell("C", 8, 0, 4, 8),
  cell("D", 0, 8, 8, 8), cell("E", 8, 8, 4, 4), cell("F", 8, 12, 4, 4),
  cell("G", 0, 16, 4, 8),
];

test("panelOrder: reading order, junk left out", () => {
  assert.deepEqual(panelOrder([...BOARD].reverse()), ["A", "B", "C", "D", "E", "F", "G"]);
  assert.deepEqual(panelOrder([cell("x", 4, 0, 4, 8), { id: "junk", x: 0 }, null, cell("y", 0, 0, 4, 8)]), ["y", "x"]);
  assert.deepEqual(panelOrder(null), []);
  assert.deepEqual([...PANEL_DIRECTIONS], ["left", "right", "up", "down"]);
});

test("neighborPanel: left and right stay on the row and stop at its ends", () => {
  assert.equal(neighborPanel(BOARD, "A", "right"), "B");
  assert.equal(neighborPanel(BOARD, "B", "right"), "C");
  assert.equal(neighborPanel(BOARD, "C", "right"), "", "nothing to the right of the last column");
  assert.equal(neighborPanel(BOARD, "B", "left"), "A");
  assert.equal(neighborPanel(BOARD, "A", "left"), "");
  assert.equal(neighborPanel(BOARD, "D", "right"), "E", "the wide panel shares rows with E and F; E lines up with its top");
  assert.equal(neighborPanel(BOARD, "F", "left"), "D");
  assert.equal(neighborPanel(BOARD, "G", "right"), "", "a row with one panel: right is a dead end, down and up are not");
});

test("neighborPanel: down and up follow the shared columns, ties go to the best-aligned edge", () => {
  assert.equal(neighborPanel(BOARD, "A", "down"), "D");
  assert.equal(neighborPanel(BOARD, "B", "down"), "D");
  assert.equal(neighborPanel(BOARD, "C", "down"), "E");
  assert.equal(neighborPanel(BOARD, "E", "down"), "F");
  assert.equal(neighborPanel(BOARD, "F", "up"), "E");
  assert.equal(neighborPanel(BOARD, "E", "up"), "C");
  assert.equal(neighborPanel(BOARD, "D", "up"), "A", "A and B both touch D; A lines up with its left edge");
  assert.equal(neighborPanel(BOARD, "A", "up"), "");
  assert.equal(neighborPanel(BOARD, "G", "down"), "");
});

test("neighborPanel: up and down fall back to the nearest panel in that half when no column is shared", () => {
  assert.equal(neighborPanel(BOARD, "F", "down"), "G", "F sits over nothing; G is the only panel below");
  assert.equal(neighborPanel(BOARD, "G", "up"), "D", "D touches G; A is further up");
  const twoRows = [cell("a", 8, 0, 4, 8), cell("b", 0, 8, 4, 8), cell("c", 4, 8, 4, 8)];
  assert.equal(neighborPanel(twoRows, "a", "down"), "c", "nearest column first");
  assert.equal(neighborPanel(twoRows, "b", "up"), "a");
  assert.equal(neighborPanel(twoRows, "b", "right"), "c");
  assert.equal(neighborPanel(twoRows, "c", "right"), "", "left and right never leave the row");
});

test("neighborPanel: unknown id, unknown direction and junk answer nothing", () => {
  assert.equal(neighborPanel(BOARD, "nope", "right"), "");
  assert.equal(neighborPanel(BOARD, "A", "diagonal"), "");
  assert.equal(neighborPanel(BOARD, "", "right"), "");
  assert.equal(neighborPanel(null, "A", "right"), "");
  assert.equal(neighborPanel([cell("A", 0, 0, 4, 8), { id: "B", x: 4, y: 0 }, null], "A", "right"), "", "a panel without a whole rectangle is not a neighbour");
  assert.equal(neighborPanel([cell("A", 0, 0, 4, 8)], "A", "down"), "", "alone");
});

// neighborPanel is already a 2D spatial search, so the arrows work on the
// free plane exactly as they do on the small board above — including the
// negative half of it.
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
  // And on the same board in canvas units, the way migration 045 left it.
  const moved = gridToCanvas(BOARD);
  assert.equal(moved.find((p) => p.id === "B").x, 32);
  assert.equal(neighborPanel(moved, "A", "right"), "B");
  assert.deepEqual(panelOrder(moved), ["A", "B", "C", "D", "E", "F", "G"]);
  // The conversion was lossy where the plan says it is: an 8-row panel grew
  // to the 28-unit minimum and overlaps the panel below, so `down` from C
  // skips the panel it now overlaps. The plane allows that, and there is no
  // pack to undo it (ADR-0113, ADR-0118).
  assert.equal(neighborPanel(moved, "C", "down"), "F");
});

// ---- edges (ADR-0116) ----------------------------------------------------
//
// An edge is the owner's recorded intent that two sessions may exchange
// messages. It grants exactly ADR-0104's mailbox contact — no transcript,
// ever — and the client's only jobs are to normalize it, to refuse what the
// server would refuse in the server's words, and to hand the canvas two
// endpoints.

const edge = (id, a, b, extra = {}) => ({ id, aPanel: a, bPanel: b, createdAt: "2026-09-10T10:00:00Z", ...extra });
const agentPanel = (id) => panel(id, { kind: "agent", ref: "agent-" + id });
const notePanel = (id) => panel(id, { kind: "note", ref: "pin-" + id });

test("normalizeEdge takes an id and two different panels, and drops junk", () => {
  assert.deepEqual(normalizeEdge({ ...edge("e1", "p1", "p2"), junk: 1 }), edge("e1", "p1", "p2"));
  assert.equal(normalizeEdge(edge("e1", "p1", "p1")), null, "an edge to the same panel is not an edge");
  assert.equal(normalizeEdge({ id: "e1", aPanel: "p1" }), null);
  assert.equal(normalizeEdge({ aPanel: "p1", bPanel: "p2" }), null);
  assert.equal(normalizeEdge(null), null);
  assert.equal(normalizeEdge({ ...edge("e1", "p1", "p2"), createdAt: 7 }).createdAt, "");
  assert.deepEqual(normalizeEdgeList({ edges: [edge("e1", "p1", "p2"), null, { id: "x" }] }), [edge("e1", "p1", "p2")]);
  assert.deepEqual(normalizeEdgeList(null), []);
});

test("normalizeCanvasDetail carries the edges, and a payload from before ADR-0116 reads none", () => {
  const d = normalizeCanvasDetail({
    ...summary("m1", "Ops"), panels: [panel("p1"), agentPanel("p2")], edges: [edge("e1", "p1", "p2"), { bad: 1 }],
  });
  assert.deepEqual(d.edges, [edge("e1", "p1", "p2")]);
  assert.deepEqual(normalizeCanvasDetail({ ...summary("m1", "Ops"), panels: [] }).edges, []);
});

test("validateEdge repeats the server's refusals, in the server's order", () => {
  const panels = [panel("p1"), agentPanel("p2"), notePanel("p3")];
  const edges = [edge("e1", "p1", "p2")];
  // The two id rules hold with nothing else in hand.
  assert.equal(validateEdge({ aPanel: "p1" }), "aPanel and bPanel are required");
  assert.equal(validateEdge({ aPanel: " ", bPanel: "p2" }), "aPanel and bPanel are required");
  assert.equal(validateEdge({ aPanel: "p1", bPanel: "p1" }), "an edge needs two different panels");
  assert.equal(validateEdge({ aPanel: "p1", bPanel: "p9" }), "", "without the panels there is nothing more to judge");
  // With the panels: both must be on this canvas and have a mailbox.
  assert.equal(validateEdge({ aPanel: "p1", bPanel: "p9" }, panels), "panel p9 is not on this canvas");
  assert.equal(validateEdge({ aPanel: "p1", bPanel: "p3" }, panels), "panel p3 is a note panel and has no mailbox: an edge links agent or terminal panels");
  assert.equal(validateEdge({ aPanel: "p1", bPanel: "p2" }, panels), "", "agent + terminal is the pair an edge links");
  // With the edges: the cap, then the pair that is already linked — in
  // either direction, because the server stores the pair ordered.
  assert.equal(validateEdge({ aPanel: "p1", bPanel: "p2" }, panels, edges), "These panels are already linked");
  assert.equal(validateEdge({ aPanel: "p2", bPanel: "p1" }, panels, edges), "These panels are already linked");
  assert.equal(validateEdge({ aPanel: "p1", bPanel: "p2" }, panels, []), "");
  const full = Array.from({ length: CANVAS_LIMITS.edges }, (_, i) => edge("e" + i, "a" + i, "b" + i));
  assert.equal(validateEdge({ aPanel: "p1", bPanel: "p2" }, panels, full), `limit: ${CANVAS_LIMITS.edges} edges per canvas`);
});

test("edgeEndpoints is the canvas's pair, or null when an end is not on the board", () => {
  const panels = [panel("p1"), agentPanel("p2")];
  const ends = edgeEndpoints(edge("e1", "p1", "p2"), panels);
  assert.equal(ends.a.id, "p1");
  assert.equal(ends.b.id, "p2");
  assert.equal(edgeEndpoints(edge("e1", "p1", "gone"), panels), null);
  assert.equal(edgeEndpoints(edge("e1", "gone", "p2"), panels), null);
  assert.equal(edgeEndpoints(edge("e1", "p1", "p2"), []), null);
  assert.equal(edgeEndpoints(null, panels), null);
  assert.equal(edgeEndpoints(edge("e1", "p1", "p1"), panels), null);
});

test("applyCanvasEvent reduces canvas.edge.added and canvas.edge.removed", () => {
  const entry = { canvas: summary("m1", "Ops", { panelCount: 2 }), panels: [panel("p1"), agentPanel("p2")], edges: [] };
  let s = { list: [summary("m1", "Ops", { panelCount: 2 })], byId: { m1: entry } };
  s = applyCanvasEvent(s, { type: "canvas.edge.added", data: { id: "m1", updatedAt: "t2", edge: edge("e1", "p1", "p2") } });
  assert.deepEqual(s.byId.m1.edges, [edge("e1", "p1", "p2")]);
  assert.equal(s.byId.m1.canvas.updatedAt, "t2");
  assert.deepEqual(entry.edges, [], "the previous state is not mutated");
  // The same id again replaces the row instead of doubling it.
  s = applyCanvasEvent(s, { type: "canvas.edge.added", data: { id: "m1", updatedAt: "t3", edge: edge("e1", "p1", "p2") } });
  assert.equal(s.byId.m1.edges.length, 1);
  // Junk is ignored.
  assert.equal(applyCanvasEvent(s, { type: "canvas.edge.added", data: { id: "m1", edge: { id: "e2" } } }), s);
  assert.equal(applyCanvasEvent(s, { type: "canvas.edge.removed", data: { id: "m1" } }), s);

  s = applyCanvasEvent(s, { type: "canvas.edge.removed", data: { id: "m1", updatedAt: "t4", edgeId: "e1" } });
  assert.deepEqual(s.byId.m1.edges, []);
  assert.equal(s.byId.m1.canvas.updatedAt, "t4");
  // A canvas that is not loaded only stamps its summary; the panel count is
  // an edge's business never.
  const unloaded = applyCanvasEvent({ list: [summary("m2", "Two", { panelCount: 3 })], byId: {} },
    { type: "canvas.edge.added", data: { id: "m2", updatedAt: "t9", edge: edge("e3", "p1", "p2") } });
  assert.equal(unloaded.list[0].updatedAt, "t9");
  assert.equal(unloaded.list[0].panelCount, 3);
  assert.deepEqual(unloaded.byId, {});
});

test("an edge does not outlive its panel, and every other event keeps the edges", () => {
  const held = [edge("e1", "p1", "p2"), edge("e2", "p2", "p3")];
  const base = () => ({
    list: [summary("m1", "Ops", { panelCount: 3 })],
    byId: { m1: { canvas: summary("m1", "Ops", { panelCount: 3 }), panels: [panel("p1"), agentPanel("p2"), agentPanel("p3")], edges: held } },
  });
  // The store cascades an edge away with its panel and announces no event
  // per edge, so the reducer drops exactly the edges that touched it.
  const removed = applyCanvasEvent(base(), { type: "canvas.panel.removed", data: { id: "m1", updatedAt: "t2", panelId: "p2" } });
  assert.deepEqual(removed.byId.m1.edges, []);
  const one = applyCanvasEvent(base(), { type: "canvas.panel.removed", data: { id: "m1", updatedAt: "t2", panelId: "p1" } });
  assert.deepEqual(one.byId.m1.edges, [edge("e2", "p2", "p3")]);
  // Every other event leaves them alone.
  for (const ev of [
    { type: "canvas.updated", data: { ...summary("m1", "Ops board"), updatedAt: "t2" } },
    { type: "canvas.layout", data: { id: "m1", updatedAt: "t2", panels: [{ id: "p1", x: 33, y: 0, w: 32, h: 42 }] } },
    { type: "canvas.panel.added", data: { id: "m1", updatedAt: "t2", panel: agentPanel("p4") } },
  ]) {
    assert.deepEqual(applyCanvasEvent(base(), ev).byId.m1.edges, held, ev.type);
  }
  // And a deleted canvas takes its entry, edges included.
  assert.deepEqual(Object.keys(applyCanvasEvent(base(), { type: "canvas.deleted", data: { id: "m1" } }).byId), []);
});

test("placementRect: a click centres the default size on the point", () => {
  const px = (u) => u * UNIT_PX;
  const r = placementRect({ x: px(100), y: px(50) }, { x: px(100) + 2, y: px(50) - 3 });
  assert.equal(r.w, PANEL_DEFAULT_CANVAS.w);
  assert.equal(r.h, PANEL_DEFAULT_CANVAS.h);
  // Centred: the point is the middle of the rectangle, not its corner.
  assert.equal(r.x, 100 - Math.round(PANEL_DEFAULT_CANVAS.w / 2));
  assert.equal(r.y, 50 - Math.round(PANEL_DEFAULT_CANVAS.h / 2));
});

test("placementRect: a drag is the rectangle drawn, in whole units, any direction", () => {
  const px = (u) => u * UNIT_PX;
  const want = { x: 10, y: 20, w: 60, h: 40 };
  const a = { x: px(10), y: px(20) };
  const b = { x: px(70), y: px(60) };
  assert.deepEqual(placementRect(a, b), want, "top-left to bottom-right");
  assert.deepEqual(placementRect(b, a), want, "bottom-right to top-left");
  assert.deepEqual(placementRect({ x: px(70), y: px(20) }, { x: px(10), y: px(60) }), want, "top-right to bottom-left");
});

test("placementRect: a rectangle drawn smaller than a panel grows from where it started", () => {
  const px = (u) => u * UNIT_PX;
  const r = placementRect({ x: px(4), y: px(4) }, { x: px(14), y: px(9) });
  assert.equal(r.w, CANVAS_LIMITS.canvasMinW);
  assert.equal(r.h, CANVAS_LIMITS.canvasMinH);
  assert.equal(r.x, 4, "the corner the reader anchored first stays put");
  assert.equal(r.y, 4);
});

test("placementRect: the whole rectangle stays on the plane", () => {
  const px = (u) => u * UNIT_PX;
  const far = CANVAS_LIMITS.canvasCoord;
  const r = placementRect({ x: px(far), y: px(far) }, { x: px(far + 50), y: px(far + 50) });
  assert.equal(r.x + r.w, far, "pulled back by its own width, not clipped");
  assert.equal(r.y + r.h, far);
  const back = placementRect({ x: px(-far - 100), y: px(-far - 100) }, { x: px(-far - 50), y: px(-far - 50) });
  assert.equal(back.x, -far);
  assert.equal(back.y, -far);
});

test("placementRect: nothing, or half a point, still yields a legal rectangle", () => {
  for (const [a, b] of [[null, null], [{ x: 0 }, undefined], [{ x: NaN, y: NaN }, { x: 1, y: 1 }]]) {
    const r = placementRect(a, b);
    assert.ok(r.w >= CANVAS_LIMITS.canvasMinW && r.h >= CANVAS_LIMITS.canvasMinH, "never below the minimum");
    assert.ok(Number.isFinite(r.x) && Number.isFinite(r.y), "never NaN");
  }
});
