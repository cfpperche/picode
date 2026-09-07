# 2026-09-06 — feat/cli-lifecycle: CLI lifecycle — update checks, update, reinstall, uninstall (ADR-0087)

Shipped: `internal/clilifecycle` (method detection + per-CLI argv plans verified
against each vendor's 2026-09 `--help`), `internal/clijob` (durable jobs on
`cli_jobs`, one lane, request-key idempotency, interrupted-never-replayed,
Close cancels as interrupted), server routes `POST /api/clis/<cli>/update-check`
and `POST /api/clis/<cli>/lifecycle` + `GET /api/cli-jobs`, `cli.job` feed
events, migration 032. Agent CLIs surface (desktop + mobile) shows an update
badge and runs update/reinstall/uninstall with typed uninstall confirmation and
inline job progress; unknown install methods get docs only.
Verified: decision-table tests in clilifecycle/clijob/store/server; full go +
JS suites; `make ci-scoped` + `make close` green; visual-review on a scratch
instance with fake vendor CLIs (badge, dropdown, running job, done job, guided
card, typed dialog, mobile) — screenshots in `var/screenshots/clis-*.png`.
visual-review: PASS
Not done / debts: claude npm-registry lag vs native releases; grok uninstall
guided-only; Windows-native path handling out of scope (ADR-0087); opportunistic
update-check fires per surface visit, not on a schedule.
Merge: fast-forward ready (after merging main into the branch).
