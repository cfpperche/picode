# 2026-09-20 — feat/inspector-followups: adversarial-review followups to session-changes
Shipped: path-only worktree watcher events (fleet pills no longer inherit sibling branch); git-spelling alias keys; cached worktree sets re-listed every 10th tick; gone line for stale tabs; gap tests (prunable/bare/image/preview-mint/PUT-per-owner); plan-note row 11; changelog fragment.
Verified: `make close` green; focused Go+JS suites pass (git_watch, worktrees, preview, fileIO, feedReducers; test failed pre-fix, passes post-fix); scratch QA confirmed pills stay on anchor + stale-tab error state (screenshot read, overlayAudit ok). Blind spot: not run inside the Windows shell.
visual-review: PASS
Not done / debts: none — no new durable debts.
Merge: fast-forward ready.

Incidents:
- Inherited foreign merge node 2cac6916 (omp sync) as parent — verified content-neutral (two-dot main..HEAD is exactly the 10 own files), landed as-is since history is never rewritten; it also fooled the close owed-check, so this note is written despite "nothing owed".
- agent-browser: use `--session <name>` per QA round, never `close --all` (closes every session on the box).
