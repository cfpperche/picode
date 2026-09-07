# 2026-09-07 — feat/cli-install-missing: install when missing (ADR-0088) + honest update-check copy

Shipped: `install` lifecycle action offered only for missing CLIs — npm-backed
for pi/codex/claude-code (same argv as reinstall, same durable lane, guard,
post-success recheck), guided docs card for grok/hermes (PiCode never runs
vendor curl installers). `clilifecycle.ForMissing` + `InstallDocs`; store
accepts the action; server refuses install on installed CLIs. UI (desktop +
mobile): Install button, guided-install card, job action label, and the
update-check line now shows the real error text and stays silent for
unmanaged installs (no more misleading "Update check failed" for brew/manual
installs).
Verified: ForMissing/argv decision tests; server E2E test installs a missing
CLI through a fake npm and flips it to Installed (deterministic, count=2);
invariant test for the new action; ci-scoped green; visual-review on a
scratch instance — Install button, guided card, install job Done → Installed
flip (screenshots `var/screenshots/clis-codex-*.png`, `clis-grok-guided-install.png`).
visual-review: PASS (audit ok:true)
Not done / debts: ADR-0087 debts stand; claude npm install is offered though
native is Anthropic's recommended method (npm is official and documented).
Merge: fast-forward ready.
