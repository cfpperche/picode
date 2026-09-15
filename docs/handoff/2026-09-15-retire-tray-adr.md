# 2026-09-15 — feat/retire-tray-adr (slice 0: Go-tray retirement)

- ADR-0142: shell becomes the only Windows resident; Go stays headless CLI.
- Contract: window close hides, tray Quit exits.
- Runbook `docs/plans/retire-go-tray.md` with handover decision table.
- Changelog fragment added.
- Commit c90e1077; `make ci-scoped` PASS.

## Next up

- Slice 1: resident core in shell (keepalive/health/tooltip).
- Owner live acceptance per runbook.

## Debts

- None new; ADR-0098 clean-machine chain still future, now targeting the shell.
