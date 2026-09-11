// Canvas link anchors (ADR-0116; docs/architecture/canvas.md, "Where a link
// touches"). Pure — no React, no React Flow, no DOM. Two rectangles and a
// count are all this needs, so the rule the plane draws is the rule
// `canvasAnchors.test.js` proves.
//
// The defect this replaces: the plane declared one `Handle` per panel at
// `Position.Top` (the link knob in the header) and drew with
// `getSimpleBezierPath`, whose control points only distinguish the
// horizontal sides. "Top" was never honoured, so the line touched wherever
// the curve happened to land — a capture of the shipped build has it leaving
// near one panel's bottom-right corner and entering the other's top edge,
// left of centre. And with a single anchor per panel, every link a panel
// carried stacked on that one point whatever direction its target lay in.
//
// The fix is what React Flow calls a **floating edge**: the endpoint is
// computed on the border facing the other node instead of read off a fixed
// handle. Two rules, in this order:
//
//   1. **Direction.** The anchor sits where the ray from this panel's centre
//      to the other panel's centre leaves this panel's border. A panel to
//      the right is joined right-edge to left-edge, one below bottom to top,
//      and a link can never cross the panel it belongs to because it starts
//      on the border already pointing at its target. A diagonal neighbour
//      picks the side its centre actually lies beyond, which is the panel's
//      own aspect talking: a wide, short panel hands a target 45° away to
//      its top or bottom, a tall one to its left or right.
//   2. **Spread.** Every link a panel carries repeats step 1 independently,
//      so two targets in nearly the same direction want nearly the same
//      point. Anchors sharing a side are pushed apart to `ANCHOR_GAP` and no
//      further: each stays as near the point that faces its own target as
//      the gap allows (`spreadAlong`, an isotonic fit — see there), the
//      order along the border matches the order of the targets, so the lines
//      do not cross, and a panel with one link is untouched by the rule.
//
// The drag preview runs the same two functions with the pointer standing in
// for the far rectangle, so the line you drag is the line you get.
//
// The curve is ours (`linkPath`) rather than React Flow's, because the chip
// needs the curve's own midpoint and normal and those are only knowable to
// whoever placed the control points. It is still symmetric and still has no
// arrowhead: an edge is undirected (ADR-0116 §6) and a marker would promise
// a restriction the store does not keep.

// Sides, as this module names them. The plane maps these to React Flow's
// `Position`; nothing here knows that type exists.
export const SIDES = Object.freeze(["top", "right", "bottom", "left"]);

// How close to a corner an anchor may sit. A link that touches the very
// corner reads as touching neither side, and the rounded border (8 px) has
// no line there to touch.
export const ANCHOR_MARGIN = 14; // px

// The least distance between two anchors on one side. 18 px is a little more
// than twice the painted line's 2 px at zoom 1 plus its rounded cap, so four
// links fanning out of one side read as four lines and not as a brush.
export const ANCHOR_GAP = 18; // px

// The shortest control-point offset. A pair of anchors almost touching still
// leaves and enters perpendicular to its border, so the tie reads as
// *attached* rather than as a chord clipped across a corner.
export const CURVE_MIN = 24; // px

const clamp = (v, lo, hi) => (v < lo ? lo : v > hi ? hi : v);

// The outward unit normal of each side, and the axis a spread runs along.
const OUT = Object.freeze({
  top: Object.freeze({ x: 0, y: -1 }),
  right: Object.freeze({ x: 1, y: 0 }),
  bottom: Object.freeze({ x: 0, y: 1 }),
  left: Object.freeze({ x: -1, y: 0 }),
});
const VERTICAL = Object.freeze({ left: true, right: true });

function centre(r) {
  return { x: (r.x || 0) + (r.width || 0) / 2, y: (r.y || 0) + (r.height || 0) / 2 };
}

// facingSide — which of `rect`'s four borders points at `other`.
//
// The comparison is `|dx| · h` against `|dy| · w`, which is `|dx| / (w/2)`
// against `|dy| / (h/2)`: the ray is measured against the rectangle's own
// half-extents, so the diagonal that switches sides is the rectangle's
// corner and not a fixed 45°. Ties and two concentric rectangles answer
// "right", deterministically — an overlapping pair has no facing side and
// the drawing has to pick one.
export function facingSide(rect, other) {
  const a = centre(rect);
  const b = centre(other);
  const dx = b.x - a.x;
  const dy = b.y - a.y;
  const w = Math.max(rect.width || 0, 1);
  const h = Math.max(rect.height || 0, 1);
  if (Math.abs(dx) * h >= Math.abs(dy) * w) return dx < 0 ? "left" : "right";
  return dy < 0 ? "top" : "bottom";
}

// borderPoint — where that ray leaves the border, clamped `ANCHOR_MARGIN`
// off both corners. This is the anchor a link would get if it were the only
// one on its side; `spreadAlong` is what happens when it is not.
export function borderPoint(rect, other) {
  const side = facingSide(rect, other);
  const a = centre(rect);
  const b = centre(other);
  const dx = b.x - a.x;
  const dy = b.y - a.y;
  const w = rect.width || 0;
  const h = rect.height || 0;
  if (VERTICAL[side]) {
    const x = side === "right" ? (rect.x || 0) + w : rect.x || 0;
    const reach = dx === 0 ? 0 : (dy * (w / 2)) / Math.abs(dx);
    const [lo, hi] = spanOf(rect, side);
    return { side, x, y: clamp(a.y + reach, lo, hi) };
  }
  const y = side === "bottom" ? (rect.y || 0) + h : rect.y || 0;
  const reach = dy === 0 ? 0 : (dx * (h / 2)) / Math.abs(dy);
  const [lo, hi] = spanOf(rect, side);
  return { side, x: clamp(a.x + reach, lo, hi), y };
}

// The usable run of one border: the side, less a corner margin at each end.
// A panel too small to hold both margins collapses to its midpoint rather
// than inverting.
export function spanOf(rect, side) {
  const vertical = VERTICAL[side];
  const lo = vertical ? rect.y || 0 : rect.x || 0;
  const len = vertical ? rect.height || 0 : rect.width || 0;
  if (len <= 2 * ANCHOR_MARGIN) {
    const mid = lo + len / 2;
    return [mid, mid];
  }
  return [lo + ANCHOR_MARGIN, lo + len - ANCHOR_MARGIN];
}

// spreadAlong — place `n` anchors on one border, each as near the point that
// faces its own target as a minimum gap allows.
//
// Returned in the caller's order. The rule, stated once:
//
//   * Sort by the wanted position. That ordering is the order of the targets
//     along the border, so honouring it is what keeps two links from one
//     panel from crossing each other on their way out.
//   * Subtract `i · gap` from the i-th wanted position. The min-gap
//     constraint `p[i+1] − p[i] ≥ gap` becomes "non-decreasing", and the
//     nearest non-decreasing sequence in the least-squares sense is the
//     pool-adjacent-violators fit: walk left to right, and whenever a value
//     would sit below the block behind it, merge the two and give both their
//     mean. Add `i · gap` back. Anchors that were already far enough apart
//     do not move at all — which is the property that matters, because it is
//     what lets a link keep pointing at its target.
//   * Slide the whole block back inside the border if it overhangs.
//   * A side that cannot hold `n` anchors at `gap` shares itself evenly
//     instead; the gap shrinks rather than the anchors leaving the panel.
export function spreadAlong(wanted, lo, hi, gap = ANCHOR_GAP) {
  const n = wanted.length;
  if (n === 0) return [];
  if (n === 1) return [clamp(wanted[0], lo, hi)];
  const span = hi - lo;
  const order = wanted.map((v, i) => i).sort((i, j) => (wanted[i] - wanted[j]) || (i - j));
  const out = new Array(n);
  if (span <= 0 || (n - 1) * gap > span) {
    const step = n > 1 ? span / (n - 1) : 0;
    order.forEach((idx, k) => { out[idx] = lo + k * step; });
    return out;
  }
  // Pool adjacent violators on q[k] = sorted[k] − k·gap.
  const blocks = []; // { sum, count }
  for (let k = 0; k < n; k++) {
    let sum = clamp(wanted[order[k]], lo, hi) - k * gap;
    let count = 1;
    while (blocks.length) {
      const last = blocks[blocks.length - 1];
      if (last.sum / last.count <= sum / count) break;
      blocks.pop();
      sum += last.sum;
      count += last.count;
    }
    blocks.push({ sum, count });
  }
  const pos = new Array(n);
  let k = 0;
  for (const b of blocks) {
    const mean = b.sum / b.count;
    for (let j = 0; j < b.count; j++, k++) pos[k] = mean + k * gap;
  }
  // The block is never wider than the span (the sorted wanted positions are
  // themselves inside it), so one shift always fits it back in.
  let shift = 0;
  if (pos[0] < lo) shift = lo - pos[0];
  if (pos[n - 1] + shift > hi) shift = hi - pos[n - 1];
  order.forEach((idx, j) => { out[idx] = pos[j] + shift; });
  return out;
}

// anchorLinks — both ends of every link, direction then spread.
//
// `rects` is id → { x, y, width, height } in plane pixels; `links` is rows of
// { id, source, target }. A link whose either end has no rectangle is left
// out of the answer rather than drawn from nowhere — that is `edgeEndpoints`'
// case (a panel not on the board), and it must not draw.
//
// Returns a Map: link id → { sourceX, sourceY, sourceSide, targetX, targetY,
// targetSide }.
export function anchorLinks(rects, links) {
  const get = (id) => (rects instanceof Map ? rects.get(id) : rects ? rects[id] : undefined);
  const out = new Map();
  const groups = new Map(); // `${panelId}|${side}` → entries
  for (const link of links || []) {
    if (!link || !link.id) continue;
    const a = get(link.source);
    const b = get(link.target);
    if (!a || !b) continue;
    const pa = borderPoint(a, b);
    const pb = borderPoint(b, a);
    out.set(link.id, {
      sourceX: pa.x, sourceY: pa.y, sourceSide: pa.side,
      targetX: pb.x, targetY: pb.y, targetSide: pb.side,
    });
    for (const end of [
      { panel: link.source, rect: a, point: pa, at: "source" },
      { panel: link.target, rect: b, point: pb, at: "target" },
    ]) {
      const key = end.panel + "|" + end.point.side;
      let g = groups.get(key);
      if (!g) { g = { rect: end.rect, side: end.point.side, ends: [] }; groups.set(key, g); }
      g.ends.push({ id: link.id, at: end.at, point: end.point });
    }
  }
  for (const g of groups.values()) {
    if (g.ends.length < 2) continue; // one link on a side keeps its own point
    const vertical = VERTICAL[g.side];
    const [lo, hi] = spanOf(g.rect, g.side);
    const placed = spreadAlong(g.ends.map((e) => (vertical ? e.point.y : e.point.x)), lo, hi);
    g.ends.forEach((e, i) => {
      const a = out.get(e.id);
      if (!a) return;
      if (vertical) a[e.at === "source" ? "sourceY" : "targetY"] = placed[i];
      else a[e.at === "source" ? "sourceX" : "targetX"] = placed[i];
    });
  }
  return out;
}

// linkPath — the cubic between two anchored ends, plus the two things the
// chip needs and only this function can answer.
//
// The control point of each end sits along that border's outward normal, half
// the facing distance out and never less than `CURVE_MIN`, so the curve is
// symmetric (swap the ends and you get the same shape, which is what an
// undirected edge must draw) and leaves each border at a right angle.
//
// `labelX` / `labelY` are B(0.5) — the curve's own centre, not the chord's
// midpoint, which stops being on the line the moment the line bows.
// `nx` / `ny` are the unit normal of the curve **at that point**: the
// derivative of a cubic at t = 0.5 is ¾·[(P2 + P3) − (P0 + P1)], so a normal
// taken from the chord would slide the chip along the curve instead of off
// it. The sign is settled the same way it always was — always up, and right
// for a vertical line — because which end is "source" is the sorted pair
// order, which is nothing a viewer can see.
export function linkPath({ sourceX, sourceY, sourceSide, targetX, targetY, targetSide }) {
  const us = OUT[sourceSide] || OUT.right;
  const ut = OUT[targetSide] || OUT.left;
  const dx = targetX - sourceX;
  const dy = targetY - sourceY;
  const os = Math.max(CURVE_MIN, 0.5 * (dx * us.x + dy * us.y));
  const ot = Math.max(CURVE_MIN, 0.5 * (-dx * ut.x + -dy * ut.y));
  const c1x = sourceX + us.x * os;
  const c1y = sourceY + us.y * os;
  const c2x = targetX + ut.x * ot;
  const c2y = targetY + ut.y * ot;
  const labelX = (sourceX + 3 * c1x + 3 * c2x + targetX) / 8;
  const labelY = (sourceY + 3 * c1y + 3 * c2y + targetY) / 8;
  const tx = c2x + targetX - sourceX - c1x;
  const ty = c2y + targetY - sourceY - c1y;
  const len = Math.hypot(tx, ty) || 1;
  let nx = ty / len;
  let ny = -tx / len;
  if (ny > 0 || (ny === 0 && nx < 0)) { nx = -nx; ny = -ny; }
  const r = (v) => Math.round(v * 100) / 100;
  return {
    path: `M${r(sourceX)},${r(sourceY)} C${r(c1x)},${r(c1y)} ${r(c2x)},${r(c2y)} ${r(targetX)},${r(targetY)}`,
    labelX, labelY, nx, ny,
  };
}

// pointAnchor — the moving end of a drag. The pointer has no rectangle, so it
// stands in as a zero-sized one: the fixed end still picks the border facing
// it, by the same `borderPoint`, and the preview cannot promise an anchor the
// finished edge would not use. The one thing the preview does not know is the
// spread — the link it is drawing is not in the set yet — so a new link into
// a crowded side lands on its facing point and joins the fan on the drop.
export function pointAnchor(x, y) {
  return { x, y, width: 0, height: 0 };
}
