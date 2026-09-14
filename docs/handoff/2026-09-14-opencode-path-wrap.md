# 2026-09-14 — opencode-path-wrap: wrapped path rows stop reading as drafts

Shipped: OpenCode's editor guard accepts as many footer continuation rows as the cwd's length implies (each matched against a strict path-only pattern, cap 8); non-path rows still refuse. Found during the onboarding matrix: an 80-col pane with a deep worktree path wrapped twice and the guard read it as a draft, silently blocking delivery.
Verified: `TestPeerOpenCodeSidebarAndWrappedFooter` — new rows (wraps-twice accepts; footer-text and bare-junk continuations refuse) plus all previous negatives; `make close` next.
visual-review: n/a (guard logic, covered by parser fixtures)
Not done / debts: none from this branch.
Merge: fast-forward ready after this close.
