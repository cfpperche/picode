# 2026-09-20 — feat/inspector-polish: exact git-menu hints + plain worktree cap note
Shipped: Inspector git-menu hints carry exact prepared commands as hover titles; push expands real branch; upstream check follows the pill ([Inspector.jsx](/home/goat/picode/.worktrees/inspector-polish/web/browser/src/components/Inspector.jsx)). Cap note reworded to plain "+N more copies of the project" with teaching tooltip ([InspectorSessionChanges.jsx](/home/goat/picode/.worktrees/inspector-polish/web/browser/src/components/InspectorSessionChanges.jsx)). Paid two debts in docs/handoff/open/inspector.md.
Verified: `make close` green in worktree. Scratch QA with 9 dirty worktrees (8 groups + cap note); menu titles asserted via DOM. Blind spot: not run inside the Windows shell.
visual-review: PASS (screenshots read in subagent; overlayAudit ok)
Not done / debts: none new.
Merge: fast-forward ready (main merged into branch; rerun `make close` then ff from root).
