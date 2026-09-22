# 2026-09-22 — feat/omp-models-roles: omp's model roles live in the Models pane

Shipped: omp's model roles, quick-switch cycle, fallback chains, retry knobs and `modelRoleStorage` moved from `#/clis/omp/settings` to `#/clis/omp/models` (sections Model roles, Fallbacks, All models). The owner asked why they were split and made the call. Mechanism: `pane: "models"` on the fields in `internal/clisettings/specs.go`; same file, revision and writer, so ADR-0181 is unchanged. `Row`/`AddRow` are exported from `CliNativeSettings.jsx` and reused by `CliModels.jsx`; the settings-row CSS is shared. A role that points at a model the folder cannot reach, or that the allowed list excludes, says so on its row. Chains are titled "When default fails" / "When any openai model fails", and `@role` is explained.
Fixed: a list's Use inherited sat at the right edge because of the base `.set-row-stack .set-ctl > .btn { margin-left: auto }`. It now sits under the list, in Settings too.
Verified: Go and JS tests, including `TestOmpModelRowsLiveInTheModelsPane` and `TestChainLabelsReadAsSentences`; `make ci-scoped` PASS. On a qa-scratch (own port, daemon ownership checked before any write) a role was set through the picker, a chain entry was added, and the file was read back. Settings now shows only Thinking level, Memory and Interface. Blind spot: the "When openai/gpt-5 fails" title is covered by unit tests only and was never captured.
visual-review: PASS (two rounds; dark theme via `?theme=dark` with the background checked; desktop 1600×1000 and mobile 390×844; overlayAudit ok; screenshots read in subagents).
Not done / debts: the Interface rows in Settings look cramped. That predates this branch and was left alone.
Merge: fast-forward ready after `make close` (main moved: merge main first).

## Next up

- omp slice 4: resolved-config and doctor card from `omp config list --json`.
- Settings' Interface rows look cramped on omp's Settings pane (pre-existing, untouched here).
