# 2026-09-20 — feat/mobile-inspector: mobile Inspector screen for agent changes and PRs
Shipped: web/shared/domain/inspector.js — the desktop rail's change-shape logic (groupChanges, sessionGroups, resolveSessionView, scope, totals) moved from web/browser/src/lib/inspector.js and shared with mobile, tests moved with it. Mobile Inspector screen #/inspector/{a|t|w}/<id>: Changes (folder-grouped sums, All | This agent scope via agentTouched fed from conversation items, multi-worktree pill/switcher/groups), Files (lazy browse doorway), PR tab (read lifted to GitPullRequest for the #n label). Glance strip on the agent screen when changes or PR exist; ProjectToolsSheet "Inspect changes" row; Work dirty chip opens it. Legacy screens/Changes.jsx + UncommittedDetail.jsx deleted; #/changes/* hashes parse onto the Inspector.
Verified: make ci-scoped PASS; make web build OK; visual-review PASS after 2 fix rounds — round 1 caught the real defect (missing mobile-git.css import); the "count mismatch" tab-badge-vs-group-header report is desktop-parity semantics (badge = all dirty checkouts, headers = per-checkout), not a bug. Not run on iOS Safari or the Windows shell.
visual-review: PASS
Benchmark note: docs/benchmarks/2026-09-20-mobile-inspector.md.
Merge: fast-forward ready (head cdf81e8e, 1 commit, 897+/504−, 32 files).

## Debts

- Servers panel deliberately desktop-only — poll-based, feed-less surface would need a reason per ADR-0048
- Inspector Files segment has no filter — the Files tool owns bounded search
- Session-scope deep link (#/inspector without visiting the agent) shows All only: touched paths live in memory (agentTouched); a persistent store is a server change
