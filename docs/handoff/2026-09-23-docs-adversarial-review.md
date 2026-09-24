# 2026-09-23 — docs-adversarial-review: code-vs-docs sweep, six fixes

Shipped: one adversarial pass over docs vs code (worktree
`feat/docs-adversarial-review`, merge `26d991005` includes main 5b09e7bc1).
Fixed: ADR-0152's raw template header lines (Status/Boundary filled from its
own Boundary section and index row); two root-absolute `[AGENTS.md](/AGENTS.md)`
links (docs/README.md, docs/architecture.md) that 404 on GitHub; two links to
the generated, uncommitted `docs/handoff.md` (docs/README.md, docs/plans/docker-v3.md)
now describe the file; docs-site/api.md's stale "200+ operations" (spec has 387
paths). Verified clean and left alone: every `make <target>` in docs exists;
all code paths/package symbols named in architecture files exist (one
past-tense mention of the ADR-0176-deleted `internal/pkgs/guest_view.go`);
all ADR citations from docs and Go/JS comments resolve; ADR index/supersessions
complete; nine-CLI catalog matches README/AGENTS.md; change-feed event names
(incl. `agent.usage` wiring at cmd/picode/main.go) match publishers and
subscribers; mobile tabs, migrations (072 max), port range 8445-8455, data
paths, `--ctl-h`, gate-test names all check out.
Verified: `make ci-scoped` PASS before and after the main merge (fmt, vet,
hooks, docs, vale 0 alerts, docs-living ok).
visual-review: n/a (no UI change)
Not done / debts: none from this sweep.
Merge: fast-forward ready (main merged in, scoped gates green on the merge).

## Next up

- `close-summary`'s "docs/decisions/README.md: new ADR is not in the index"
  line is a false positive for edits to an already-indexed ADR — the check at
  scripts/close-summary.sh:41 keys on any `docs/decisions/NNNN-*` diff, not
  just new files; tighten it to `git diff --name-only --diff-filter=A` if it
  cries wolf again.
