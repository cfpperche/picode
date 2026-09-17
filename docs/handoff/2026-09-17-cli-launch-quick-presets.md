# 2026-09-17 — cli-launch-quick-presets

Quick launch settings in the Agent CLIs launch editor (`feat/cli-launch-quick-presets`, **not merged — owner validation pending in the worktree**).

## Done
- `web/shared/domain/cliLaunchPresets.js`: verified per-CLI quick controls — pi (model, thinking), Claude Code (model, permission mode), Codex (model, `-c model_reasoning_effort`, sandbox, approvals, `--yolo`), Grok 1.0.34 (permission mode, `--always-approve`), Hermes Agent v0.21.3 (model, `--reasoning`, `--yolo`), OpenCode (model, `--auto`), Omp 18.2.4 (model). 34 table-driven tests: alias/joined forms, malformed flag, kv-vs-other `-c` pairs, exclusivity group.
- Codex yolo/sandbox/approvals form an exclusivity group (`applyQuickSetting`): the CLI refuses the combination, so picking one clears the others.
- `LaunchFields` (desktop + mobile) renders the controls above an **Advanced** reveal; same editor for defaults, terminal overrides and profiles. Dangerous picks show a one-line amber warning. Muse Code and Antigravity stay read-only (no adapter — structural).
- Launch profiles `<details>` opens by default once a row exists (both shells).
- End-to-end checked on the isolated devserver (:8446): quick control → draft args → preview → Save/profile store; exclusivity exercised in the UI (yolo ↔ sandbox).
- Summaries keyboard focus normalized to the accent ring (was UA dark ring, owner screenshot 20:17).

## Gates
- `ci-scoped: PASS` (fmt, vet, hooks, test-js, build; green run recorded post-commit).
- visual-review: PASS (codex-danger2/claude-danger2/pi-stack/pi-advanced/profiles-row/hermes/omp; overlayAudit ok; card 5/5). First capture round FAILED on framing, re-captured.

## Debts
- Grok `--sandbox <PROFILE>` and `--reasoning-effort` have no documented value list in 1.0.34 — left advanced-only rather than guessing options.

## Next up
- Owner validation in the worktree (devserver on :8446, isolated data dir); merge only after that.
