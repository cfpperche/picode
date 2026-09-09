# 2026-09-09 — feat/shots-manifest-quiet: no manifest churn when captures are kept

Shipped: `scripts/docs-shots.mjs` keeps the previous `capturedAt` and
`gitSha` when every surface kept its image and its inputs, so a capture that
changes nothing leaves the tree clean. Before, the stamp alone made each
deploy with "stale" captures commit a two-line manifest refresh.
Verified: `make docs-shots` in this worktree right after main's capture —
four surfaces kept, tree clean; `make close`.
visual-review: n/a
Not done / debts: a fingerprint change with identical pixels still rewrites
the manifest (correct: docs-check needs the new input hashes).
Merge: fast-forward ready.
