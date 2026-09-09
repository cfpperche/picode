import assert from "node:assert/strict";
import { test } from "node:test";
import { touches } from "./feedReducers.js";
import {
  MATRIX_EVENTS, MATRIX_LIMITS, applyMatrixEvent, normalizeMatrix, normalizeMatrixDetail, normalizeMatrixList,
  normalizePanel, validateCompact, validateName, validatePanel, validatePlacement,
} from "./matrix.js";

const summary = (id, name, extra = {}) => ({
  id, name, compact: "vertical", createdAt: "2026-09-09T10:00:00Z", updatedAt: "2026-09-09T10:00:00Z", panelCount: 0, ...extra,
});
const panel = (id, extra = {}) => ({ id, kind: "terminal", ref: "term-" + id, x: 0, y: 0, w: 4, h: 8, createdAt: "2026-09-09T10:00:00Z", ...extra });

test("limits and event types are the server's (ADR-0108)", () => {
  assert.deepEqual({ ...MATRIX_LIMITS }, { matrices: 64, panels: 500, name: 80, cols: 12, minW: 4, minH: 8 });
  assert.deepEqual([...MATRIX_EVENTS], ["matrix.created", "matrix.updated", "matrix.layout", "matrix.panel.added", "matrix.panel.removed", "matrix.deleted"]);
  for (const type of MATRIX_EVENTS) assert.equal(touches({ type }, ["matrix"]), true, type);
});

test("normalizeMatrix keeps the summary fields and drops junk", () => {
  const m = normalizeMatrix({ id: "m1", name: "Ops", compact: "none", createdAt: "a", updatedAt: "b", panelCount: 3, panels: [{}], extra: 1 });
  assert.deepEqual(m, { id: "m1", name: "Ops", compact: "none", createdAt: "a", updatedAt: "b", panelCount: 3 });
  assert.equal(normalizeMatrix({ id: "m1", name: "Ops", compact: "diagonal" }).compact, "vertical");
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
