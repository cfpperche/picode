# 2026-09-09 — feat/process-review: second process-cost review (ADR-0105)

Shipped: ADR-0105 (amends 0086). Deploy timer and `make deploy-batch` removed;
`make deploy` refreshes stale captures itself (pixel tolerance in
`docs-shots.mjs`). Changelog fragments in `docs/changelog.d/` + `make changelog`.
`docs/handoff.md` without "Current state", capped at 100 lines and 8 KB.
Pre-commit: whitespace check, fragment rule, no handoff/fragment commits on
`main` in the root. `scripts/go-test.sh` shards `internal/server` four ways and
unsets `PICODE_TERM_ID`; `make ci` runs gates with `-j4`; `make web` is a stamp;
`make close` reuses a green `ci-scoped`. `make adr` + mandatory Boundary line.
`docs/architecture.md` split into `docs/architecture/*.md` (28 files).
GitHub CI: Go matrix Ubuntu-only except tags/dispatch, tmux cached.
Verified: `make hooks-check` 29 rows green; `node --test` for the assembler;
sharded server suite green three times with `-count=1`; `make ci` (full, -j4)
green in the worktree. GitHub workflow unverified until the next push.
visual-review: n/a (no UI change)
Not done / debts: worktrees still start with a cold Go test cache (results are
keyed by directory); `.pi/compact.json` unchanged by owner decision; the split
architecture keeps original wording — subsystem files may repeat context that
the index used to carry.
Merge: fast-forward ready.
