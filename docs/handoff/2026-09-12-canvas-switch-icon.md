# 2026-09-12 — Canvas toolbar: one group, one switcher shape

`feat/canvas-switch-icon` → `main` (3e14e5a2). Four owner calls, three of
them reversing decisions taken earlier the same day.

## What changed

- **One `.cv-cluster`**, not two with a gap. Order: switcher, **Add panel**,
  − / readout / + / **Fit**, `⋯`. The menu ends the row and opens
  `side="top" align="end"`.
- **No accent fill** on Add panel.
- **Fit** is `IconFit` (lucide `Maximize`, four corner brackets) — React
  Flow's own glyph. `IconExpand` (`Maximize2`) stays for everything else.
- **The switcher is always the `<select>`**, with `IconCanvas` before it. The
  label-only shape for a single canvas (2026-09-11) is gone: on a row of
  identical segments a label reads as a disabled control, and the one-canvas
  reader is the one who has not learnt canvases are plural.

## A measurement that corrected a comment

I had written that the 104 px floor keeps the dock from resizing when a
second canvas appears. It does not: a native `<select>` sizes to its
**longest option**. One canvas named `aaa` is 104 px; a second with a long
name takes the same box to 240. The dock therefore re-centres when the canvas
list changes — a deliberate act, never mid-gesture, which is the invariant a
centred dock needs. CSS and `docs/architecture/canvas.md` now say that.

## Verified

Isolated instance: one group with seven segments in the asked order; the menu
opens above its trigger and flush with its right edge; every
`[data-align-row]` on the page even; the switcher 104 px with one canvas and
240 px with a long-named second.
