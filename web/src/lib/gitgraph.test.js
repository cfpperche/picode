import assert from "node:assert/strict";
import { test } from "node:test";
import {
  layout, layoutUncommitted, UNCOMMITTED, simplifyLines, branchPath,
  colourAt, COLOURS, GRID, repoNameFromKey,
} from "./gitgraph.js";

const commit = (hash, ...parents) => ({ hash, parents });

test("empty history lays out to nothing", () => {
  assert.deepEqual(layout([]), { columns: 0, vertices: [], branches: [] });
  assert.deepEqual(layout(null), { columns: 0, vertices: [], branches: [] });
});

test("a linear history is a single column", () => {
  const g = layout([commit("c", "b"), commit("b", "a"), commit("a")]);
  assert.equal(g.columns, 1);
  assert.deepEqual(g.vertices.map((v) => v.x), [0, 0, 0]);
  assert.deepEqual(g.vertices.map((v) => v.colour), [0, 0, 0]);
});

test("a branch and its merge occupy two columns", () => {
  //  A   merge of B and C
  //  |\
  //  B C
  //  |/
  //  D
  const g = layout([
    commit("A", "B", "C"),
    commit("B", "D"),
    commit("C", "D"),
    commit("D"),
  ]);
  assert.equal(g.columns, 2);
  assert.deepEqual(g.vertices.map((v) => v.x), [0, 0, 1, 0]);
  // The side branch gets its own colour; the trunk keeps the first.
  assert.equal(g.vertices[0].colour, 0);
  assert.equal(g.vertices[2].colour, 1);
});

test("a parent outside the loaded window still draws a line off the bottom", () => {
  const g = layout([commit("b", "a-not-loaded")]);
  assert.equal(g.vertices.length, 1);
  assert.equal(g.branches.length, 1);
  assert.ok(g.branches[0].lines.length >= 0);
});

test("a colour is reused once its branch has ended above the new one", () => {
  // The rule is strict: getAvailableColour reuses colour i only when the new
  // branch starts *below* the row where branch i ended. The first fork closes
  // at row 3, so the fork opening at row 5 gets its colour back.
  //
  //   A   fork
  //   |\
  //   B C
  //   |/
  //   D  E  (the first fork has closed by here)
  //   |
  //   F   second fork
  //   |\
  //   G H
  //   |/
  //   I
  const g = layout([
    commit("A", "B", "C"),
    commit("B", "D"),
    commit("C", "D"),
    commit("D", "E"),
    commit("E", "F"),
    commit("F", "G", "H"),
    commit("G", "I"),
    commit("H", "I"),
    commit("I"),
  ]);
  const used = new Set(g.vertices.map((v) => v.colour));
  assert.deepEqual([...used].sort(), [0, 1], `colour not recycled: ${[...used]}`);
  assert.equal(g.vertices[2].colour, 1, "first fork takes colour 1");
  assert.equal(g.vertices[7].colour, 1, "second fork gets colour 1 back");
});

test("overlapping forks do not share a colour", () => {
  // The counterpart: while the first fork is still open, a second one must
  // take a colour of its own, even though only two are ever on screen.
  const g = layout([
    commit("A", "B", "C"),
    commit("B", "D"),
    commit("C", "D"),
    commit("D", "E", "F"),
    commit("E", "G"),
    commit("F", "G"),
    commit("G"),
  ]);
  const used = new Set(g.vertices.map((v) => v.colour));
  assert.equal(used.size, 3, `expected three distinct colours, got ${[...used]}`);
});

test("collinear vertical segments collapse into one", () => {
  const straight = [
    { p1: { x: 0, y: 0 }, p2: { x: 0, y: 1 }, lockedFirst: true },
    { p1: { x: 0, y: 1 }, p2: { x: 0, y: 2 }, lockedFirst: true },
    { p1: { x: 0, y: 2 }, p2: { x: 0, y: 3 }, lockedFirst: true },
  ];
  const merged = simplifyLines(straight);
  assert.equal(merged.length, 1);
  assert.deepEqual(merged[0].p1, { x: 0, y: 0 });
  assert.deepEqual(merged[0].p2, { x: 0, y: 3 });
  // The input is not mutated: the caller may redraw from it.
  assert.equal(straight.length, 3);
});

test("a sideways step breaks the run", () => {
  const bent = [
    { p1: { x: 0, y: 0 }, p2: { x: 0, y: 1 }, lockedFirst: true },
    { p1: { x: 0, y: 1 }, p2: { x: 1, y: 2 }, lockedFirst: false },
    { p1: { x: 1, y: 2 }, p2: { x: 1, y: 3 }, lockedFirst: true },
  ];
  assert.equal(simplifyLines(bent).length, 3);
});

test("vertical runs draw as straight lines, sideways as a curve", () => {
  const vertical = branchPath([{ p1: { x: 0, y: 0 }, p2: { x: 0, y: 2 }, lockedFirst: true }]);
  assert.match(vertical, /^M\d+,\d+(\.\d)?L\d+,\d+(\.\d)?$/);
  assert.ok(!vertical.includes("C"), "a vertical line needs no curve");

  const sideways = branchPath([{ p1: { x: 0, y: 0 }, p2: { x: 1, y: 1 }, lockedFirst: false }]);
  assert.ok(sideways.includes("C"), "rounded style uses a cubic");

  const angular = branchPath([{ p1: { x: 0, y: 0 }, p2: { x: 1, y: 1 }, lockedFirst: false }], {
    style: "angular",
  });
  assert.ok(!angular.includes("C"), "angular style uses no curve");
  assert.equal(angular.match(/L/g).length, 2, "angular turns with two segments");
});

test("grid coordinates become pixels through the grid config", () => {
  const path = branchPath([{ p1: { x: 0, y: 0 }, p2: { x: 0, y: 1 }, lockedFirst: true }]);
  assert.ok(path.startsWith(`M${GRID.offsetX},${GRID.offsetY.toFixed(1)}`), path);
  assert.ok(path.endsWith(`L${GRID.offsetX},${(GRID.offsetY + GRID.y).toFixed(1)}`), path);
});

test("a real branch layout produces a drawable path", () => {
  const g = layout([
    commit("A", "B", "C"),
    commit("B", "D"),
    commit("C", "D"),
    commit("D"),
  ]);
  for (const branch of g.branches) {
    const path = branchPath(branch.lines);
    assert.ok(path.startsWith("M"), `path must open with a move: ${path}`);
    assert.ok(!/NaN|undefined/.test(path), `path has holes: ${path}`);
  }
});

test("colours wrap and never return undefined", () => {
  assert.equal(colourAt(0), COLOURS[0]);
  assert.equal(colourAt(COLOURS.length), COLOURS[0]);
  assert.equal(colourAt(COLOURS.length + 2), COLOURS[2]);
  assert.equal(colourAt(-1), COLOURS[COLOURS.length - 1]);
  for (let i = -20; i < 40; i++) assert.ok(colourAt(i), `colourAt(${i}) is empty`);
});

test("the repository name comes out of the key, matching the server", () => {
  assert.equal(repoNameFromKey("/home/u/picode/.git"), "picode");
  assert.equal(repoNameFromKey("/home/u/picode/.git/"), "picode");
  assert.equal(repoNameFromKey("/srv/mirrors/picode.git"), "picode");
  assert.equal(repoNameFromKey(""), "");
  assert.equal(repoNameFromKey(null), "");
});

// expandAt/expandY came with the port and had never been called: they shift
// everything drawn below an expanded row so an inline panel can open there.
// These tests run that code for the first time, before any pixel depends on it.

test("a vertical line crossing the expanded row stretches only its lower end", () => {
  const line = [{ p1: { x: 0, y: 0 }, p2: { x: 0, y: 2 }, lockedFirst: true }];
  const plain = branchPath(line);
  const opened = branchPath(line, { expandAt: 1, expandY: 100 });
  assert.equal(plain, `M${GRID.offsetX},${GRID.offsetY.toFixed(1)}L${GRID.offsetX},${(GRID.offsetY + 2 * GRID.y).toFixed(1)}`);
  assert.equal(opened, `M${GRID.offsetX},${GRID.offsetY.toFixed(1)}L${GRID.offsetX},${(GRID.offsetY + 2 * GRID.y + 100).toFixed(1)}`);
});

test("a line entirely below the expanded row shifts both ends", () => {
  const line = [{ p1: { x: 0, y: 2 }, p2: { x: 0, y: 3 }, lockedFirst: true }];
  const opened = branchPath(line, { expandAt: 1, expandY: 50 });
  assert.equal(
    opened,
    `M${GRID.offsetX},${(GRID.offsetY + 2 * GRID.y + 50).toFixed(1)}L${GRID.offsetX},${(GRID.offsetY + 3 * GRID.y + 50).toFixed(1)}`,
  );
});

test("a line entirely above the expanded row does not move", () => {
  const line = [{ p1: { x: 0, y: 0 }, p2: { x: 0, y: 1 }, lockedFirst: true }];
  assert.equal(branchPath(line, { expandAt: 1, expandY: 300 }), branchPath(line));
});

test("a branch stays continuous across the expansion boundary", () => {
  // A vertical run into a sideways merge curve. Wherever the boundary falls,
  // the path must stay one stroke: a second M means the shifted start of one
  // segment no longer meets the shifted end of the one before it.
  const lines = [
    { p1: { x: 0, y: 0 }, p2: { x: 0, y: 1 }, lockedFirst: true },
    { p1: { x: 0, y: 1 }, p2: { x: 1, y: 2 }, lockedFirst: false },
    { p1: { x: 1, y: 2 }, p2: { x: 1, y: 3 }, lockedFirst: true },
  ];
  for (let at = 0; at <= 3; at++) {
    const path = branchPath(lines, { expandAt: at, expandY: 120 });
    assert.equal(path.match(/M/g).length, 1, `expandAt ${at} split the stroke: ${path}`);
    assert.ok(!/NaN|undefined/.test(path), `expandAt ${at} has holes: ${path}`);
  }
});

test("expanding at the last row is a no-op for every line", () => {
  const lines = [
    { p1: { x: 0, y: 0 }, p2: { x: 0, y: 1 }, lockedFirst: true },
    { p1: { x: 0, y: 1 }, p2: { x: 1, y: 2 }, lockedFirst: false },
  ];
  assert.equal(branchPath(lines, { expandAt: 2, expandY: 500 }), branchPath(lines));
});

// The Uncommitted Changes row (ADR-0038): a pseudo-commit through the ordinary
// allocator, with its trail split out for dashed drawing.

test("a dirty tree over HEAD at the top yields a one-row dashed trail", () => {
  const { placed, rows, dashed } = layoutUncommitted(
    [commit("h"), commit("g")], // g is unreachable from h here; keep it simple
    "h",
  );
  assert.equal(rows.length, 3);
  assert.equal(rows[0].hash, UNCOMMITTED);
  assert.equal(placed.vertices.length, 3);
  assert.ok(dashed, "a trail must come back");
  assert.equal(dashed.lines.length, 1);
  assert.deepEqual(dashed.lines[0].p1, { x: 0, y: 0 });
  assert.deepEqual(dashed.lines[0].p2, { x: 0, y: 1 });
});

test("the dashed trail reaches a HEAD further down and stops there", () => {
  //  *      uncommitted
  //  A      a branch tip above HEAD
  //  H      HEAD
  //  B
  const { placed, rows, dashed } = layoutUncommitted(
    [commit("A", "B"), commit("H", "B"), commit("B")],
    "H",
  );
  assert.equal(rows.length, 4);
  const headRow = rows.findIndex((r) => r.hash === "H");
  assert.equal(headRow, 2);
  assert.ok(dashed.lines.length >= 1);
  const last = dashed.lines[dashed.lines.length - 1];
  assert.equal(last.p2.y, headRow, "the trail must end at the HEAD row");
  for (const l of dashed.lines) {
    assert.ok(l.p2.y <= headRow, "no dashed line may pass HEAD");
  }
  // The solid remainder of that branch continues below HEAD (H → B).
  const first = placed.branches[0];
  for (const l of first.lines) {
    assert.ok(l.p2.y > headRow, "solid lines start below the HEAD row");
  }
  assert.ok(first.lines.length >= 1, "history below HEAD stays solid");
});

test("a HEAD outside the loaded window means no pseudo row at all", () => {
  const commits = [commit("A", "B"), commit("B")];
  const { placed, rows, dashed } = layoutUncommitted(commits, "not-loaded");
  assert.equal(rows, commits);
  assert.equal(dashed, null);
  assert.equal(placed.vertices.length, 2);
  const noHead = layoutUncommitted(commits, "");
  assert.equal(noHead.dashed, null);
});

test("the pseudo row does not disturb where the history lands", () => {
  const commits = [
    commit("A", "B", "C"),
    commit("B", "D"),
    commit("C", "D"),
    commit("D"),
  ];
  const plain = layout(commits);
  const { placed } = layoutUncommitted(commits, "A");
  // Same columns for every real commit, one row down.
  for (let i = 0; i < commits.length; i++) {
    assert.equal(placed.vertices[i + 1].x, plain.vertices[i].x, `commit ${commits[i].hash} moved`);
  }
});
