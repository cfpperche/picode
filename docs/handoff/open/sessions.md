# Sessions, resume and handoff formats

## Next

- Sessions phase 2: codex scan cache; Hermes titles only, no `profiles/` scan.

## Debts

- [ ] Pi interactive lazy binding: a fresh successful reply did not expose Continue until explicit session listing; investigate automatic binding. Evidence: [live CLI report](../../reports/2026-09-20-continue-in-demo.md).
- [ ] Muse Code 1.3.0: READY completed but native session-index.db stayed empty and Continue remained absent after stop; investigate current session discovery. Evidence: [live CLI report](../../reports/2026-09-20-continue-in-demo.md).
- [ ] Omp Sessions listing (`GET /api/clis/omp/sessions`) reads only `~/.omp`, so workspace Omp agents' conversations under `<dataDir>/omp-sessions` (261 MB, 197 files on the owner's machine) never appear; including them scans every agent file per listing (`scanOmpFile` reads whole files) — decide scope (e.g. only with `?workspace`, like pi's ADR-0040 union) and measure the cost first.
- Codex resume: the sub-agent hook payload was never captured (names from the 0.154 roster); a sub-agent pin persists until that terminal's next session.
- Handoff (ADR-0088/0094): upstream formats undocumented (a bump is a refused write); Codex/Grok list a handed-off session only after a restart or by id; Hermes flattens tool calls.
- CLI lifecycle: npm data lags native Claude releases; grok uninstall is guided-only; Windows paths out of scope.

## Notes

- Hermes: `cli-v1-*` screenshots not regenerated; may write `shell-hooks-allowlist.json`.
