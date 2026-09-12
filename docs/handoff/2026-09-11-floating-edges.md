# 2026-09-11 — feat/floating-edges

Canvas links attach on the border facing the other panel (React Flow's
*floating edge*), instead of on one `Position.Top` handle per panel.

## What the owner reported

A screenshot of two panels joined by a link: the line left near one panel's
bottom-right corner and entered the other's top edge, left of centre. "Por
que não arrumamos os pontos de conexão? Um terminal pode ter comunicação com
diferentes outros" — and a pointer at how n8n wires nodes.

It read as arbitrary because it was. `Plane.jsx` declared one `Handle` per
panel at `Position.Top`; `Link.jsx` drew with `getSimpleBezierPath`, whose
`getControl` only distinguishes the horizontal sides. "Top" was never
honoured, so the touch point was wherever the curve landed, and a panel's
second link left from the same point as its first.

n8n is not the model: its ports carry direction (input one side, output the
other) and a Canvas link is an undirected mailbox contact (ADR-0116 §6). A
drawn arrow would promise a restriction the store does not keep.

## What shipped

`web/shared/domain/canvasAnchors.js` — pure, 28 tests in
`canvasAnchors.test.js`:

- `facingSide` / `borderPoint`: the anchor is where the centre→centre ray
  leaves the border, compared against the rectangle's own half-extents
  (`|Δx|·h` vs `|Δy|·w`), clamped 14 px off both corners.
- `spreadAlong`: anchors sharing a side are pushed to 18 px apart and no
  further — sort by the wanted position, subtract `i·gap`, pool adjacent
  violators, add it back, slide the block inside the border. Anchors already
  far enough apart do not move; a side too short shares itself evenly.
- `linkPath`: the cubic, plus `labelX`/`labelY` = B(0.5) and the normal at
  that point (¾·[(P2+P3)−(P0+P1)] turned perpendicular). Control points on
  the outward normal, half the facing distance, never under 24 px.
- `pointAnchor`: the pointer as a zero-sized rectangle, for the preview.

`Plane.jsx` computes the anchors from `nodes` (the live drag geometry) and
hands each edge its own in `data.anchor`; `Link.jsx` draws `linkPath`;
`ConnectionPreview` (exported from `Link.jsx`) is
`connectionLineComponent`, so the drag preview runs the same two functions.
The header connector stays a real `Handle` — the gesture is unchanged. The
boundary test and `shared/package.json` name the new module as the app's.

Everything ADR-0116 put on the line is unchanged: four broken states and
their colours, the word **Broken**, the reason line, the chip and its
normal offset, `interactionWidth` 22 with `non-scaling-stroke`.

## Measured (scratch `floating-edges`, 26 panels / 20 links, CDP at 60 Hz)

idle 60.0 fps. Drag: 0 links 60.1, 1 link 60.1, 4 links 60.1 at zoom 1.0
(59 distinct edge-path states over 121 frames, so the recompute is real);
60.1 at 0.6, 59.8 at 0.5. Pan: 60.5 at both 1.0 and 0.4, one distinct edge
state — a pan does not move nodes, so it never recomputes. The pass itself:
0.017 ms for this board, 0.147 ms at 200 links, 0.743 ms at the 1000-link
cap.

## Honest leftovers

- Two panels in the same row are still joined by a straight segment. That is
  geometry, and now it is the right drawing: a short perpendicular connector
  in the gap, not a diagonal across the plane.
- The preview cannot show the spread (the new link is not in the set), so a
  drop onto a crowded side moves the line by up to one gap on release.
- Pre-existing, found while measuring: **below zoom ≈0.43 a panel that keeps
  a header cannot be dragged by it.** `--cv-grab` is 12 screen px and the
  header is 28·zoom, so the resize band covers it. Panels whose body is a
  pane become plates and are unaffected; a "gone" row keeps its header and
  is not.
- The scratch fixture's panels are fabricated refs ("gone" terminals) and
  the live/broken split was produced by routing `GET /api/communication` to
  a stub — enrolment needs a real recorded session. Geometry, colours and
  the chip come from the real code path.
