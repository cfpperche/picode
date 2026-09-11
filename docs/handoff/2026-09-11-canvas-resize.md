# 2026-09-11 — feat/canvas-resize: grabbable edges, the plane's texture, curved links

Shipped on the branch (not merged, not deployed).

**Resize targets** (`canvas.css` "Resize targets", `Plane.jsx` `autoScale={false}`): every control is sized off `--cv-px` = `calc(1px / var(--cv-zoom))`, so it is constant in *screen* pixels. At zoom 1.0 / 0.8 / 0.4 — before: edge band **1.0 / 0.8 / 0.4** px, corner **3.5 / 3.3 / 2.9** effective (`.cv-panel` clips the outward half); after: band **12** and corner **20 × 20** at all three. Hit area and paint are separate rectangles (the `::after` keeps the grid's 24×3 bar and corner L in `--text-secondary`); the CSS says so, because the next reader will merge them. Drags at 1.0: E +64 w · S +48 h · W −40 x/+40 w · N −40 y/+40 h · NW −32/−32 → +32/+32 · SE +48/+48, each **one** pointerdown on a control and **one** `PATCH …/layout`. Header still drags; tmux refit 36×11 → 55×11 → 55×17.

**Pattern token:** `--canvas-pattern` in both theme blocks, all three variants keyed off it. Light `#a9b3c6` = **1.88:1** on `#f0f2f7`, dark `#4a4a58` = **2.21:1** on `#0e0e11` (was `--border`: 1.16:1 / 1.23:1, and Grid took the *weaker* token). Geometry untouched — the gap is four cells and panels snap to it. The `⋯` menu gained **Background…**, which navigates to `#/preferences`; Preferences stays the only writer.

**Links curve:** `getSimpleBezierPath` + `ConnectionLineType.SimpleBezier`, so the preview is the result. `getBezierPath` hooks over itself with both handles at `Position.Top`; `getSmoothStepPath` reads as directed wiring. Aligned pairs still draw straight — that is what an S between two aligned points is. `interactionWidth` stays 22 with `non-scaling-stroke`: 22 screen px at every zoom, not 8.8 at 0.4. No parallel-edge rule to fix — a duplicate pair is refused three times over.

**Three recorded defects:** duplicate `onLaunchAction` gone (esbuild warned before, silent after); the Inspector chip draws one string, the branch, with the worktree moved into the `title` (+1 test); placeholder rule — the action that starts work is accented (`Run`, `Try again`, `Resume last session`), the one that navigates or tidies is not.

Debts: a resize in progress has no panel-level feedback — `.cv-panel.resizing` and `.react-draggable-dragging` in `canvas.css` are dead react-grid-layout selectors. The bands take the outer 12 screen px of a live pane's edges (the top one takes 12 of the 28 px header).
