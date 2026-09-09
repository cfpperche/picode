# 2026-09-09 — feat/matrix-spike: Matrix phase 0, react-grid-layout measured under React 19

Shipped: phase 0 of the Matrix app (`docs/plans/matrix-app.md`, accepted by
the owner 2026-09-09). Two commits: the plan itself (03ca0e4c, on main) and
the study `docs/benchmarks/2026-09-09-matrix-live-grid.md` plus the plan
updates it forced (c7695107). Folded into the plan: load dwell 300 ms /
unload 5 s; default panel 4×14 cells with minW 4 and minH 8; `react-resizable`
declared directly; no lazy import. Written down: attach teardown after a page
reload waits for the bridge's 60 s pongWait.
Verified: scratch instance with a throwaway component that did not merge —
react-grid-layout 2.2.4 under React 19.1, 300 wrappers, nine live xterm
panels: drags at 58–60 fps, zero console warnings, +24 KB gzip, about 4 %
renderer CPU for nine TUIs; chunk loading suspends sockets out of view and
resumes the same xterm instances; a 300 ms load dwell cut a fly-over from
341 body loads to 6. Scratch instance, its twelve terminals and the browser
session were removed.
visual-review: n/a (docs only; the spike component did not merge)
Verdict: GO for react-grid-layout; dockview only if v2 wants tabs inside panels.
Not done / debts: no changelog fragment (nothing user-visible); the later
phases remain (below).
Next: phases 1 (`feat/apps-native-surface`) and 2 (`feat/matrix-store`) can
run in parallel worktrees; phase 3 after both merge.
Merge: fast-forward ready.
