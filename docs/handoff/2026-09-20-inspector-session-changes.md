# 2026-09-20 — feat/inspector-session-changes: Inspector Changes follows dirty linked worktrees
Shipped: Following pill + Back, View switcher, per-checkout groups in InspectorChanges; `gitstatus` worktrees[]; `?worktree=` on text/save/preview reads; watcher covers linked worktrees; tabs persist checkout (picode-file-worktrees). Paseo Changes shape extended; ADR-0078 one-worktree-per-task observation.
Verified: `make close` green; Go tests (7 new) + JS pure tests pass; scratch QA on isolated fixture verified follow/switcher/groups/center-diff/reload-persistence/live-refetch — not run against production tmux or the Windows shell.
visual-review: PASS (5 screenshots read, overlayAudit ok twice, 2 nits fixed)
Not done / debts: cross-repo session-touched following (needs per-CLI session parsing); nested non-linked clones invisible; pre-existing git-menu hint truncation (see open/inspector.md).
Merge: fast-forward ready.

## Debts

- Cross-repo session-touched following still future work — durable in open/inspector.md
