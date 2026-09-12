# 2026-09-12 — Canvas: three anchors, a context menu, a Radix switcher

`feat/canvas-context-menu` → `main` (b639a29e). Four owner calls in one pass.

## What changed

- **Camera column bottom-left** (`.cv-camera`), toolbar centred, minimap
  right. `FIT_TOP`/`FIT_BOTTOM` unchanged — the bottom band is still the
  150 px minimap, which is taller than either of the other two.
- **No zoom readout.** See the doc: the recovery (click an inert body → snap
  to 1) is intact, the *warning* is what is gone.
- **`canvasChrome.js`** (shared, per viewer): one switch hides the camera,
  the toolbar and the minimap. Only the literal `"hidden"` hides.
- **Plane context menu**, same body as `⋯` via `canvasMenu(M)` — Radix's
  context and dropdown menus are one primitive under two names.
- **Switcher is a `DropdownMenu.RadioGroup`**, not a native `<select>`.
- New dependency: `@radix-ui/react-context-menu` in the desktop workspace.
  `npm install --workspace desktop`, not the web root — the first attempt put
  it in `web/package.json`, which is the workspace root and holds no deps.

## Two collisions the menu had to dodge

- Right-drag **pans** (`panOnDrag={[1, 2]}`) and also ends in a `contextmenu`
  event, so every pan would open a menu. The press is recorded on
  `pointerdown`; more than `PAN_SLOP` (4 px) of travel drops the menu.
- A right-click inside a panel belongs to the panel (terminals have their own
  menu) — stopped in the capture phase before Radix's trigger sees it.

## Verified

Isolated instance: camera 36 × 110 at 12 px from the left and bottom, dock
centred at 12 px, minimap at 12 px; the switcher stays 104 px with a
long-named second canvas (the `<select>` grew to 240); the menu lists both
canvases with a tick on the current one; **Show controls** removes all three
and the right-click still opens with the toggle unchecked; a synthetic
right-drag of 50 px opens nothing, a 1 px one opens the menu.

## Adding to `web/shared/domain`

A new shared module needs a line in `web/shared/package.json` `exports` or
the build fails with `private shared import`. Cost me one build.
