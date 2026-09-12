# 2026-09-12 — the zoom readout returns; the last nextSlot caller goes

`feat/canvas-zoom-readout` → `main`. Two of the owner's nine pending items.

- **Readout on `.cv-camera`**, between `−` and **Fit**: `+`, `−`, `100%`,
  **Fit**. Pushed from the plane through `onZoom`; the surface holds it in a
  ref with a listener set, `ZoomReadout` is the only subscriber, so the plane
  never re-renders on a wheel. `subscribeZoom` now calls the listener once on
  subscribe, so a readout mounted after a zoom is not stuck at 100.
- **`addPanel`'s `nextSlot` fallback removed.** Unreachable since **Add
  panel** left: the picker only opens from a marking gesture, so `placed` is
  always a rectangle. `nextSlot` keeps only `tidyCanvas` and the migration
  record.

## Verified

Isolated instance at `/browser/`: column 36 × 146 reading `+ − 100% ⛶`;
two zoom-outs took it to 69 % and clicking the readout returned to 100 %;
arming Terminal and drawing a 280 × 220 rectangle landed a panel at exactly
that rectangle.

## Still open from that list

Text and drawing tools; the Canvas has no `docs-shots` capture; a still is
monochrome (`translateToString` drops cell attributes); v1.1 leftovers
(quick reply in the chat panel, frozen-frame placeholders, drag from the
sidebar) are backlog by the owner's call; file and diff panels stay
uncreatable, discarded by the owner.
