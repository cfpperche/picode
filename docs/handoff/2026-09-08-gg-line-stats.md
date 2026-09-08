# 2026-09-08 — feat/gg-line-stats: per-commit +/- column in the git graph listing
Shipped: each commit row in the desktop git graph shows its own diff's
+added/−removed (`internal/gitgraph/gitgraph.go` — `Commit.Add/Del`, one extra
`--shortstat -m --first-parent` walk over the same window the graph loads;
row and commit detail reconcile exactly, merges included). Column renders in
`GitGraph.jsx` reusing `gg-add`/`gg-del`; hides ≤900px — the row's auto
columns already overflowed there, and the column must not add to it. Empty or
binary-only commits draw no value. Docs: `architecture.md` /git payload,
CHANGELOG [Unreleased].
Verified: table test on the stat parser + repo-fixture test (merge, empty
commit, pure adds/deletes); `make ci-scoped`; scratch instance QA — listing,
detail conciliation (+758 −125 = sum of 28 file cards), empty-repo state,
800px (column hidden, overflow back to pre-change 25px) and 1440px.
visual-review: PASS (listing + detail + empty + narrow; overlayAudit ok)
Not done / debts: mobile app does not render the new fields (additive payload,
out of scope); the stat walk adds ~0.8 s to a manual graph refresh on this
repository's 250-commit window (documented in architecture.md).
Merge: fast-forward ready.
