# 2026-09-17 — cli-launch-quick-presets

Quick launch settings in the Agent CLIs launch editor (`feat/cli-launch-quick-presets`, **not merged — owner validation pending in the worktree**).

## Done
- `web/shared/domain/cliLaunchPresets.js`: verified per-CLI quick controls — pi (model, thinking), Claude Code (model, permission mode), Codex (model, `-c model_reasoning_effort`, sandbox, approvals), OpenCode (model, `--auto`). Pure in-place arg patching: generated flag replaced at its position, user args untouched, empty value removes. Table-driven tests (15) incl. alias, joined form, malformed flag, kv-vs-other `-c` pairs.
- `LaunchFields` (desktop + mobile) renders the controls above an **Advanced** reveal; same editor for defaults, terminal overrides and profiles. Dangerous picks (bypass/full-access/never) show a one-line amber warning. Grok, Hermes, Omp, Muse, Antigravity render the form unchanged (flags unverified).
- Launch profiles `<details>` opens by default once a row exists (both shells).
- End-to-end checked on a scratch instance: quick control → draft args → preview → Save/profile store; profile select in New terminal pre-fills controls.

## Gates
- `ci-scoped: PASS` (fmt, vet, hooks, test-js, build; green run recorded post-commit).
- visual-review: PASS (codex-danger2/claude-danger2/pi-stack/pi-advanced/profiles-row; overlayAudit ok; card 5/5). First capture round FAILED on framing, re-captured.

## Debts
- Omp inherits pi's flag shape but is left advanced-only until `omp --help` verifies `--model`/`--thinking` on the installed version; same for Grok/Hermes.

## Next up
- Owner validation in the worktree (`scripts/qa-scratch.sh start <name>` after `make web`); merge only after that.
