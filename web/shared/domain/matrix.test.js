import assert from "node:assert/strict";
import { test } from "node:test";
import { touches } from "./feedReducers.js";
import {
  LOAD_DWELL_MS, MATRIX_EVENTS, MATRIX_LIMITS, PANEL_DEFAULT, SUSPENDED_MAX, SUSPENDED_TTL_MS, UNLOAD_AFTER_MS,
  applyMatrixEvent, bindingState, layoutDiff, loadPolicy, nextSlot, normalizeMatrix, normalizeMatrixDetail,
  normalizeMatrixList, normalizePanel, suspendedToDispose, validateCompact, validateName, validatePanel, validatePlacement,
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
  assert.deepEqual(nextSlot([rect("a", 0, 0)], 6, 14, 8), { x: 0, y: 14 }, "cols bounds the candidates");
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
