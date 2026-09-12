# 2026-09-12 — Canvas: one toolbar on the bottom edge

`feat/canvas-toolbar` → `main` (84bd369c). Owner's ask: nodeterm's bar —
minimap bottom-right, everything else merged into one horizontal dock.

## What changed

`.cv-toolbar` is centred on the bottom edge and holds two `.cv-cluster`
button groups: the canvas (switcher, **Add panel** filled, `⋯`) and the
camera (−, readout, +, Fit). `.cv-chrome` and `.cv-zoom` are gone;
`.cv-cluster` lost `position: absolute` and is now purely the group
primitive. `FIT_TOP` 60 → 24 (nothing floats at the top any more).

## The one structural change

The camera had to leave `Plane.jsx` to sit in the same dock, and the zoom
readout with it. The plane pushes a percentage through a new `onZoom` prop;
the surface keeps it in a ref plus a listener set (`subscribeZoom`), and
`ZoomReadout` is the only subscriber. Surface state would have re-rendered
the plane on every notch of the wheel.

## Verified

Isolated instance, one terminal panel: readout 100 → 83 → 100 driven
through the plane handle; `⋯` opens above its trigger (menu bottom 579 vs
trigger top 585); the minimap is `display: none` at a 700 px stage and back
at 900; 28 `[data-align-row]` rows on the page, none misaligned.

## Watch for

The switcher keeps a 104 px floor. It looks like slack beside a short name,
but in a centred dock any width change moves every other button under the
pointer — that floor is load-bearing now, not cosmetic.
