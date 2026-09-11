# 2026-09-11 — the Canvas chrome floats, and light stops being white (`feat/canvas-chrome`)

**Session one.** The Canvas surface has no header: the plane fills the stage and the
chrome floats as two `.cv-cluster` groups — switcher / Add panel / `⋯` top-left, zoom
+ minimap bottom-right (`docs/architecture/canvas.md` "Chrome"; nodeterm,
`docs/benchmarks/2026-09-10-node-canvas.md`).

**Session two, the plane's texture is the reader's.** **Plain / Dots / Grid / Cross**
in Preferences → Appearance, each card showing its own pattern (`PatternSwatch.jsx`).
`web/shared/domain/canvasPattern.js` holds it: key `picode-canvas-pattern`, safe
default, a window event so open planes *and* the page re-read without a reload. The
colour is CSS — `canvas.css` keys `--xy-background-pattern-color` off `data-bg` — so a
theme switch re-tints on the next paint. Scope is the plane only; canvas.md says so.

**And light stopped being pure white.** It was inverted (page `#ffffff`, panels grey),
so a card sank into its own page. Now ground `#f0f2f7`, panel `#fbfcfe`, elevated
`#ffffff` — split from panel in both themes, dark `#1c1c23` — hover `#e4e9f2`, border
`#dde2ec` / `#b3bccd`, accent `#2a62d6`. Measured: primary text 15.9:1 on ground and
17.3:1 on panel, secondary 5.3 / 5.8, accent 4.4 → 5.4. `.dlg` took `--bg-base` and
now takes elevated plus a shadow; light's minimap mask went 45 % → 16 % black.

**Verified.** `make close` green. Scratch `canvas-bg` (two terminals, an agent, a
three-panel canvas), 24 screenshots read one by one in both themes: dashboard
first-run and populated, terminal, agent, file tree, git graph, sidebar, Preferences,
all four patterns, a menu and a dialog over a panel, mobile Now and a terminal. Reload
keeps the pattern, theme re-tints live, `__picodeOverlayAudit()` stayed `ok`.
visual-review: PASS.

**Debts.** Inspector header chips truncate to `· fe… ·…` in both themes (pre-existing,
not palette); `App.jsx` has a duplicate `onLaunchAction` attribute (`f71fd2fd`,
harmless); the canvas panel's **Run** reads as a disabled chip beside the agent tab's
accent button; the terminal's own light theme is still `#ffffff` glass, which is
deliberate (it is the paper, not the page) but never re-measured; still no docs-shots
capture for the Canvas.

**Merge.** Fast-forward ready.
