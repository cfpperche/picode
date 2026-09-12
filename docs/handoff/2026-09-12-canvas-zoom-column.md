# 2026-09-12 — Canvas: the camera moves, the still keeps its ground

`feat/canvas-zoom-column` → `main` (cc346113). Three owner asks, one pass.

## What changed

- **Camera controls bottom-left, as a column**; the minimap stays
  bottom-right. Following React Flow's own `<Controls>`. **Fit** is the
  `Maximize2` icon — the word set the column's width. `FIT_BOTTOM` drops
  from 204 to 162: the corners stand side by side now, so the band is the
  taller of the two, not one stacked on the other.
- **A still wears the terminal's ground.** `--cv-term-bg` / `--cv-term-fg`
  on the plane root from `xtermTheme(readTermTheme())`, refreshed on
  `picode-term-theme`. Crossing 0.75 no longer turns a black pane white.
- **Three resize targets**, not eight: bottom, right, bottom-right, through
  `NodeResizeControl` (`RESIZE_CONTROLS`, `Plane.jsx`). The dead CSS for the
  other five went with them.

## Verified

On an isolated instance (`qa-scratch.sh`, two terminal panels): the column
36 × 146 at 12 px from both edges and the minimap opposite it; at 58 % both
stills computed `rgb(14,14,17)` on `rgb(236,236,241)`; at 33 % the
name-plates still take over; three controls per node, `bottom line`,
`right line`, `bottom right handle`.

## Watch for

A still cannot show **colour in its text** — `translateToString` drops cell
attributes, so a coloured prompt reads monochrome below 0.75. Recovering it
means per-cell capture, which is the cost the band exists to avoid.
