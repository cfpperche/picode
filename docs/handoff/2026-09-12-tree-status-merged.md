# 2026-09-12 — feat/tree-status-merged: finished trees stop looking stuck

Follow-up to ADR-0123, found by reading the first real `make worktree-status`
on main: my own merged worktree was reported as `stalled: nothing committed —
empty branch`, and merged trees sat in the board's In flight section. A branch
whose tip `main` contains is finished, not idle (`git branch --merged main`),
so it now reads "merged — `make worktree-gc` can remove it", is excluded from
the board, and a dirty tree is never called merged (its branch tip may be in
main while someone is still typing). `scripts/worktree-status.test.mjs` is new:
7 rows over ages, stall reasons and the board's flight lines.

Verified: `node --test scripts/*.test.mjs` — 74 tests, 0 fail; `make close`.

visual-review: n/a (no UI surface changed).

Not done / debts: the header row of `make worktree-status` still lists the root
checkout as a worktree named `main` (accurate, arguably noise) — owner's taste,
not mine to decide.

Merge: fast-forward ready.
