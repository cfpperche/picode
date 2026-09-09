# 2026-09-09 — feat/process-review-fixes: adversarial review of ADR-0105's landing

Shipped: hook accepts the release cut (`## [x.y.z]` heading) and validates
staged fragments via `changelog-assemble.mjs --check`; selftest rows for both.
`make deploy` capture commit is pathspec-limited. `make adr` numbers across
`git worktree list`, probed from a worktree. Nine relative links repaired in
`docs/architecture/`. Capture tolerance: absolute 128 px budget, count
printed. Fragments ordered by commit date. Workflow: `apt-get update` before
the cached tmux install. AGENTS.md hook list, close comment and deploy-log
path corrected. `docs/handoff.md` no longer enumerates branches; the close
summary prints `git worktree list`.
Verified: `make hooks-check` green (31 rows); assembler tests; `--check`
exit codes; pathspec commit proven on a throwaway repo; `make ci-scoped`
(full matrix) green.
visual-review: n/a (no UI change)
Not done / debts: GitHub workflow still unexercised; the 128 px budget is a
guess until a capture log shows real counts; shards unproven against
cross-process port/file collisions (GitHub stays serial with -race).
Merge: fast-forward ready.
