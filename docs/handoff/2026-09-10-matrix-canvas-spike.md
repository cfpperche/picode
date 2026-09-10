# 2026-09-10 — feat/matrix-canvas-spike (Matrix v2 phase C0)

Base 90f51091 · head bf8aaa46 · docs only (plan, study, this note).

## What landed

`docs/plans/matrix-canvas.md` (the v2 canvas plan, C0–C5, committed as it stood then amended with C0's numbers) and `docs/benchmarks/2026-09-10-node-canvas.md` (the study, indexed in `docs/benchmarks/README.md`).

## The gate: GO on `@xyflow/react` 12.11.6

500 nodes with nine live `top -d 1` xterm bodies on a scratch instance: 60 fps idle, panning (up to 8 000 px/s), zooming and dragging one node; 0 console and 0 page errors. `nodrag`, `nowheel` and header-drag all hold over a live pane. Chunk loading bounds the attaches under the canvas transform (9 at rest → 0 attached / 9 suspended panned away → the same 9 instances back). `NodeResizer` resized a live pane 54×16 → 83×26 and fired `onResizeEnd` once. A full-screen still costs 0.02 ms and swaps in one commit — but a capture taken at unmount is one crossing late, so the first crossing paints an empty still.

**One rule changed:** xterm divides the pointer offset inside the *transformed* rect by the *untransformed* cell size, so the cell it reports is `cell × zoom` — at 0.8 a click on (50,15) reports (40,12) and a ten-character drag returns seven. PiCode ships tmux `mouse on`, so the error reaches copy mode and every TUI. §4.3 now gives a pointer only at zoom 1.0 (live-but-inert 0.8–1.0, still below 0.75, name-plate below 0.4).

Also folded in: `onlyRenderVisibleElements` **off** (on it churned 54 socket suspends per three pan round trips); the minimap is free but unstyled for a dark app; the host is **lazy-imported** (+63 KB gzip eager vs 205 B + a 49 KB split chunk); the still is captured *before* the flip, not at unmount; `rootMargin` gains its horizontal half.

## Reverted before closing

The spike component (`MatrixCanvasSpike.jsx`), its `?spike=canvas` mount in `main.jsx`, `@xyflow/react` and the lockfile. What the component looked like — its counters and gates — is written down in the study so C2 can rebuild it.

## Debts

- Multi-panel drag: 20 selected panels ran 31 fps (18 fps in a slower run, 17 long tasks) with culling off. Unmeasured above 20; C2 needs a plan if it bites.
- Renderer CPU for nine live TUIs was noisy on WSL2: 4.2–14.0 % of one core across five samples.
- C0 never exercised agent (TUI) panels, the maximize layer, or two browsers on one canvas.
