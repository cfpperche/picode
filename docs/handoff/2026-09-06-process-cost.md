# 2026-09-06 — feat/process-cost: the rite costs less than the change (ADR-0086)

Shipped: `GET /api/deploy/readiness` (loopback, no session) and a `picode
deploy` that refuses (exit 2) while any agent/terminal is mid-turn, `--force`
/ `PICODE_DEPLOY_FORCE=1` to override; `make deploy-batch` + systemd
`picode-deploy.timer` (12:00/18:00/23:00) that recaptures stale public images
first; `make ci-scoped` (scripts/ci-scope.mjs --local); `make close` and
`make close-summary`; `make worktree` (hardlinked node_modules, 1 s) and
`make worktree-gc`; `scripts/qa-scratch.sh`; docs-check fingerprint advisory
unless `--strict`; pre-commit cap of 100 lines on docs/handoff.md (selftest
rows); AGENTS.md reading table + rite rewrite; skills (quality-gate,
handoff-update, agent-browser traps, visual-review evidence in var/);
`.pi/roles.json` default/vision thinking `medium`; docs/screenshots frozen;
macOS CI fixes (EvalSymlinks in the worktree test; run/type routes validate
before the tmux 503). Old handoff archived verbatim; ADR index gained 0084–0086.
Verified: `make ci` green in the worktree (full, because the Makefile changed);
readiness route answered `{"ready":true}` on a scratch instance seeded via
the API; guard unit tests (refuse / force / silent daemon); hooks selftest
27/27; scoped CI classified a CSS-only diff as web, a handler as go.
visual-review: n/a (no UI change).
Not done: timers not yet installed at the time of the note (Next up 1); the
first batch deploy is unguarded because the running daemon predates the
route; `internal/server` suite still serial; history rewrite is the owner's.
Merge: fast-forward ready after merging main (`f93f22b3`).
