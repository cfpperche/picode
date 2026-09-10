# 2026-09-09 — matrix-surface: the Matrix surface, keyboard and maximize

Shipped (plan phase 3, both sessions): the native Matrix app (ADR-0108/0109) at
`#/app/matrix/<id>` — grid, chunk loading, picker, switcher, empty states, save/409 —
plus session 2's keyboard (one focused panel, arrows, Home/End, Enter engages,
Shift+Esc leaves, Delete with Undo, roving tabindex), maximize as a layer over the grid,
pane disposal on `terminal.deleted`/`agent.deleted`, a fleet-loaded gate, and
`docs-site/guide/matrix.md` in the nav. Rules: `docs/architecture/matrix.md`.
Verified: `make close` PASS. Scratch QA (`qa-scratch.sh`, port 8473, two agent-browser
sessions): drag, resize and reload keep the layout; the agent's tab re-claims the pane
(tmux 38x22 → 121x52 → 38x22) and the panel gets it back; maximize moves the same xterm
(58x29 → 120x47) and Esc restores; 40 panels — 12 loaded at rest, 21 mid-scroll (the
three-viewport band), suspended instances kept and reattached as the same objects, 8
mounts on a 16 000 px/s pass; a delete gives the gone row and disposes the pane at once;
the second browser followed add, layout and delete; both 409s (duplicate binding, stale
layout) showed the server's message and left no wrapper behind.
Budgets (vs `docs/benchmarks/2026-09-09-matrix-live-grid.md`): nine live TUIs at 60.0 fps
in-page, 0 long tasks, 4.2 % of one core in the page's renderer (0.1 % idle); drag
60.3–60.9 fps with those nine live.
visual-review: PASS — 12 captures in dark and light (empty states, 6 panels, 40
mid-scroll, picker with `__picodeOverlayAudit` ok, maximize, gone row); card 5/5 after
fixing the two defects the captures showed: 15 s of "That terminal is gone." on every
panel while the fleet loaded, and a placeholder action with no fill.
Not done / debts: no docs-shots capture (needs a `desktop-matrix` profile plus a fixture
that seeds a matrix); the app's toast covers the surface's Close; the dialog has no scrim
and little elevation in dark; after a grow-resize an idle pane's cursor stays in its old
cell until the next redraw (xterm reflow, pre-existing).
Merge: fast-forward ready. No deploy.
