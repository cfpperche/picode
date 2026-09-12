# 2026-09-12 — the reader draws where a panel goes

`feat/canvas-element-tools` → `main` (ded13e57). Replaces **Add panel**.

## Shape

Arm (`.cv-tool`, `aria-pressed`) → mark (`.cv-mark-layer` in `Plane.jsx`) →
pick (`PanelPicker only={kind}`). `placementRect` in
`web/shared/domain/canvas.js` turns two plane points into stored units;
`CANVAS_TOOLS` in `CanvasSurface.jsx` is the row — add a kind there and it
gets the same three steps. Chrome moved: switcher + `⋯` top-left, tools
bottom-centre, camera bottom-left, minimap bottom-right.

## Two traps, both found in the browser and not on paper

- **The opening `fitView` steals a placed camera.** First panel on a fresh
  canvas (no stored view) → `fitted.current` was false → the camera jumped
  the moment the rectangle was released. `onMarkUp` now sets `fitted.current`
  before reporting: drawing on a plane settles its camera.
- **A narrowed dialog needs its own count.** `total` is every kind summed, so
  with one agent and no terminals the terminal dialog said "every terminal is
  already on this canvas". `kindTotal` per kind now.

## Notes for the next QA

- `element.click()` from `eval` does **not** arm the tool — use the CLI's real
  `click`. The low-level `agent-browser mouse move/down/up` is what draws the
  rectangle, and it needs ~0.3 s between steps or the events outrun React.
- `setPointerCapture` throws on a synthetic PointerEvent; it is wrapped in
  try/catch, which is also right for real pointers the browser stops tracking.

## Left undone

Text and drawings are not panels yet, so they have no tool. Files and changes
deliberately keep none — `⋯` → **Add panel…** is their way in, and the only
remaining caller of `nextSlot` for new panels.
