import assert from "node:assert/strict";
import { test } from "node:test";
import {
  ANCHOR_GAP, ANCHOR_MARGIN, CURVE_MIN, SIDES,
  anchorLinks, borderPoint, facingSide, linkPath, pointAnchor, spanOf, spreadAlong,
} from "./canvasAnchors.js";

// A panel the size the plane actually makes: 32 × 42 units at 8 px.
const panel = (x, y, width = 256, height = 336) => ({ x, y, width, height });
const near = (a, b, eps = 0.001) => assert.ok(Math.abs(a - b) <= eps, `${a} ≉ ${b}`);

test("the four sides are the four sides", () => {
  assert.deepEqual([...SIDES].sort(), ["bottom", "left", "right", "top"]);
});

// ---- rule 1: direction -----------------------------------------------------

test("facingSide answers the border the other panel's centre lies beyond", () => {
  const a = panel(0, 0);
  assert.equal(facingSide(a, panel(600, 0)), "right");
  assert.equal(facingSide(a, panel(-600, 0)), "left");
  assert.equal(facingSide(a, panel(0, -600)), "top");
  assert.equal(facingSide(a, panel(0, 600)), "bottom");
});

test("facingSide measures the ray against the panel's own corner, not 45°", () => {
  // Equal centre offsets. A tall panel (256 × 336) hands that direction to
  // its side; a wide one (336 × 256) hands the same direction to its top or
  // bottom. The switch is the rectangle's diagonal.
  const d = { x: 300, y: 300, width: 256, height: 336 };
  assert.equal(facingSide(panel(0, 0, 256, 336), d), "right");
  assert.equal(facingSide(panel(0, 0, 336, 256), d), "bottom");
});

test("facingSide is deterministic for two panels on the same centre", () => {
  assert.equal(facingSide(panel(0, 0), panel(0, 0)), "right");
});

test("a panel to the right joins right-edge to left-edge", () => {
  const a = panel(0, 0);
  const b = panel(600, 0);
  const pa = borderPoint(a, b);
  const pb = borderPoint(b, a);
  assert.equal(pa.side, "right");
  assert.equal(pb.side, "left");
  assert.equal(pa.x, 256);
  assert.equal(pb.x, 600);
  near(pa.y, 168);
  near(pb.y, 168);
});

test("a panel below joins bottom-edge to top-edge", () => {
  const a = panel(0, 0);
  const b = panel(0, 600);
  const pa = borderPoint(a, b);
  const pb = borderPoint(b, a);
  assert.equal(pa.side, "bottom");
  assert.equal(pb.side, "top");
  assert.equal(pa.y, 336);
  assert.equal(pb.y, 600);
});

test("a diagonal neighbour anchors off-centre along the facing side", () => {
  const a = panel(0, 0);
  const b = panel(600, 300);
  const pa = borderPoint(a, b);
  const pb = borderPoint(b, a);
  assert.equal(pa.side, "right");
  assert.equal(pb.side, "left");
  // The ray leaves a's right border below a's own midline, and enters b's
  // left border above b's — each end points at the other.
  assert.ok(pa.y > 168, "source anchor below its midline");
  assert.ok(pb.y < 468, "target anchor above its midline");
});

test("an anchor never sits on a corner", () => {
  const a = panel(0, 0);
  // Exactly along the panel's own diagonal (336 / 256), which is where the
  // ray leaves through the corner itself.
  const p = borderPoint(a, { x: 1000, y: 1312.5, width: 256, height: 336 });
  const [lo, hi] = spanOf(a, "right");
  assert.equal(p.side, "right");
  assert.equal(p.y, hi);
  assert.equal(hi, 336 - ANCHOR_MARGIN);
  assert.equal(lo, ANCHOR_MARGIN);
});

test("a panel too small for two margins anchors on its midpoint", () => {
  assert.deepEqual(spanOf({ x: 10, y: 10, width: 20, height: 20 }, "left"), [20, 20]);
  assert.deepEqual(spanOf({ x: 0, y: 0, width: 256, height: 336 }, "top"), [ANCHOR_MARGIN, 256 - ANCHOR_MARGIN]);
});

// ---- rule 2: spread --------------------------------------------------------

test("one link on a side keeps exactly the point that faces its target", () => {
  assert.deepEqual(spreadAlong([200], 0, 336), [200]);
});

test("anchors already a gap apart are not moved", () => {
  const wanted = [40, 120, 200, 300];
  assert.deepEqual(spreadAlong(wanted, 14, 322), wanted);
});

test("anchors that would stack are pushed to exactly the gap, centred on what they wanted", () => {
  const out = spreadAlong([168, 168, 168], 14, 322);
  assert.deepEqual(out.map((v) => Math.round(v)), [168 - ANCHOR_GAP, 168, 168 + ANCHOR_GAP]);
});

test("the spread keeps the order of the targets, so two links do not cross", () => {
  // Wanted positions in the caller's order are scrambled; the answer keeps
  // each entry's rank.
  const out = spreadAlong([300, 100, 305, 95], 14, 322);
  assert.ok(out[1] < out[3] === 100 < 95 || out[3] < out[1], "the lower rank stays lower");
  assert.ok(out[3] < out[1], "95 stays below 100");
  assert.ok(out[0] < out[2], "300 stays below 305");
  assert.ok(out[1] < out[0], "the lower pair stays below the upper pair");
  for (let i = 1; i < 4; i++) {
    const sorted = [...out].sort((a, b) => a - b);
    assert.ok(sorted[i] - sorted[i - 1] >= ANCHOR_GAP - 0.001, "every neighbour a gap apart");
  }
});

test("the spread slides back inside the border rather than overhanging it", () => {
  const out = spreadAlong([320, 322, 322], 14, 322);
  assert.ok(out.every((v) => v >= 14 - 0.001 && v <= 322 + 0.001), JSON.stringify(out));
  assert.equal(Math.round(Math.max(...out)), 322);
  assert.equal(Math.round(Math.min(...out)), 322 - 2 * ANCHOR_GAP);
});

test("a side that cannot hold them at the gap shares itself evenly", () => {
  const out = spreadAlong([10, 20, 30, 40, 50], 0, 40, ANCHOR_GAP);
  assert.deepEqual(out, [0, 10, 20, 30, 40]);
});

test("a side with no run at all stacks them on the midpoint", () => {
  assert.deepEqual(spreadAlong([5, 5, 5], 20, 20), [20, 20, 20]);
});

// ---- both rules together ---------------------------------------------------

test("anchorLinks joins each relative position on the opposing borders", () => {
  const rects = { c: panel(0, 0), r: panel(600, 0), l: panel(-600, 0), u: panel(0, -600), d: panel(0, 600), q: panel(600, 600) };
  const got = anchorLinks(rects, [
    { id: "e-r", source: "c", target: "r" },
    { id: "e-l", source: "c", target: "l" },
    { id: "e-u", source: "c", target: "u" },
    { id: "e-d", source: "c", target: "d" },
    { id: "e-q", source: "c", target: "q" },
  ]);
  assert.deepEqual(
    [...got].map(([id, a]) => [id, a.sourceSide, a.targetSide]),
    [["e-r", "right", "left"], ["e-l", "left", "right"], ["e-u", "top", "bottom"], ["e-d", "bottom", "top"], ["e-q", "right", "left"]],
  );
});

test("four links out of one panel do not stack", () => {
  // Four targets all to the right, at four heights: one side, four anchors.
  const rects = { hub: panel(0, 0) };
  const links = [];
  for (let i = 0; i < 4; i++) {
    rects["t" + i] = panel(700, i * 90 - 40);
    links.push({ id: "e" + i, source: "hub", target: "t" + i });
  }
  const got = anchorLinks(rects, links);
  const ys = links.map((l) => got.get(l.id).sourceY);
  assert.ok(links.every((l) => got.get(l.id).sourceSide === "right"), "all four leave the right border");
  const sorted = [...ys].sort((a, b) => a - b);
  for (let i = 1; i < sorted.length; i++) {
    assert.ok(sorted[i] - sorted[i - 1] >= ANCHOR_GAP - 0.001, `stacked at ${sorted}`);
  }
  const [lo, hi] = spanOf(rects.hub, "right");
  assert.ok(ys.every((y) => y >= lo - 0.001 && y <= hi + 0.001), `${ys} inside [${lo}, ${hi}]`);
  // Order follows the targets: the lowest target takes the lowest anchor.
  assert.deepEqual(ys, [...ys].sort((a, b) => a - b));
});

test("links in different directions are not spread against each other", () => {
  const rects = { hub: panel(0, 0), r: panel(700, 0), d: panel(0, 700) };
  const got = anchorLinks(rects, [
    { id: "a", source: "hub", target: "r" },
    { id: "b", source: "hub", target: "d" },
  ]);
  near(got.get("a").sourceY, 168); // alone on the right border
  near(got.get("b").sourceX, 128); // alone on the bottom border
});

test("a link with an end that is not on the board is not drawn", () => {
  const got = anchorLinks({ a: panel(0, 0) }, [{ id: "e", source: "a", target: "gone" }]);
  assert.equal(got.size, 0);
});

test("the anchors are the same shape whichever end the store called source", () => {
  const rects = { a: panel(0, 0), b: panel(600, 300) };
  const ab = anchorLinks(rects, [{ id: "e", source: "a", target: "b" }]).get("e");
  const ba = anchorLinks(rects, [{ id: "e", source: "b", target: "a" }]).get("e");
  assert.deepEqual(
    { x: ab.sourceX, y: ab.sourceY, s: ab.sourceSide },
    { x: ba.targetX, y: ba.targetY, s: ba.targetSide },
  );
});

// ---- the curve -------------------------------------------------------------

test("linkPath leaves and enters perpendicular to the borders it touches", () => {
  const g = linkPath({ sourceX: 256, sourceY: 168, sourceSide: "right", targetX: 600, targetY: 468, targetSide: "left" });
  const m = g.path.match(/^M([-\d.]+),([-\d.]+) C([-\d.]+),([-\d.]+) ([-\d.]+),([-\d.]+) ([-\d.]+),([-\d.]+)$/);
  assert.ok(m, g.path);
  const [, x0, y0, c1x, c1y, c2x, c2y, x3] = m.map(Number);
  assert.equal(y0, Number(c1y), "the first control point is level with the right border");
  assert.ok(c1x > x0, "and outside it");
  assert.equal(Number(c2y), 468, "the second is level with the left border");
  assert.ok(Number(c2x) < x3, "and outside it");
});

test("linkPath never draws a control point closer than CURVE_MIN", () => {
  const g = linkPath({ sourceX: 0, sourceY: 0, sourceSide: "right", targetX: 4, targetY: 0, targetSide: "left" });
  assert.match(g.path, new RegExp(`C${CURVE_MIN},0 ${4 - CURVE_MIN},0`));
});

test("linkPath is symmetric: swapping the ends draws the same shape", () => {
  const a = linkPath({ sourceX: 256, sourceY: 168, sourceSide: "right", targetX: 600, targetY: 468, targetSide: "left" });
  const b = linkPath({ sourceX: 600, sourceY: 468, sourceSide: "left", targetX: 256, targetY: 168, targetSide: "right" });
  near(a.labelX, b.labelX);
  near(a.labelY, b.labelY);
  assert.deepEqual([a.nx, a.ny], [b.nx, b.ny]);
});

test("the label is the curve's own centre, not the chord's midpoint", () => {
  // Two opposing sides draw a symmetric S, whose centre *is* the chord's.
  const s = linkPath({ sourceX: 0, sourceY: 0, sourceSide: "bottom", targetX: 400, targetY: 400, targetSide: "top" });
  near(s.labelX, 200);
  near(s.labelY, 200);
  // Two sides at a right angle do not, and this is the case a chord midpoint
  // would put the chip off the line it belongs to.
  const g = linkPath({ sourceX: 128, sourceY: 336, sourceSide: "bottom", targetX: 600, targetY: 468, targetSide: "left" });
  near(g.labelX, 275.5);
  near(g.labelY, 426.75);
  assert.ok(Math.abs(g.labelX - (128 + 600) / 2) > 1, "off the chord in x");
  assert.ok(Math.abs(g.labelY - (336 + 468) / 2) > 1, "off the chord in y");
});

test("the chip's normal always points up, and right for a vertical line", () => {
  for (const s of [
    { sourceX: 0, sourceY: 0, sourceSide: "right", targetX: 400, targetY: 0, targetSide: "left" },
    { sourceX: 400, sourceY: 0, sourceSide: "left", targetX: 0, targetY: 0, targetSide: "right" },
    { sourceX: 0, sourceY: 0, sourceSide: "bottom", targetX: 0, targetY: 400, targetSide: "top" },
    { sourceX: 0, sourceY: 400, sourceSide: "top", targetX: 0, targetY: 0, targetSide: "bottom" },
  ]) {
    const g = linkPath(s);
    near(Math.hypot(g.nx, g.ny), 1);
    assert.ok(g.ny < 0 || (g.ny === 0 && g.nx > 0), JSON.stringify(g));
  }
});

test("two panels in the same row still join with a straight segment — now border to border", () => {
  const rects = { a: panel(0, 0), b: panel(600, 0) };
  const a = anchorLinks(rects, [{ id: "e", source: "a", target: "b" }]).get("e");
  const g = linkPath(a);
  // 256 → 600 at a constant y: the shortest tie between the facing borders,
  // and it crosses neither panel.
  assert.equal(g.path, "M256,168 C428,168 428,168 600,168");
  near(g.labelY, 168);
});

// ---- the drag preview ------------------------------------------------------

test("the pointer stands in as a zero-sized rectangle, so the preview anchors like the edge", () => {
  const a = panel(0, 0);
  const p = pointAnchor(900, 168);
  assert.deepEqual(pointAnchor(900, 168), { x: 900, y: 168, width: 0, height: 0 });
  assert.equal(borderPoint(a, p).side, "right");
  assert.equal(borderPoint(a, pointAnchor(0, 900)).side, "bottom");
  assert.equal(borderPoint(a, pointAnchor(-900, 168)).side, "left");
});
