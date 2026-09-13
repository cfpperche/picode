# Sessions, resume and handoff formats

## Next

- Sessions phase 2: codex scan cache; Hermes titles only, no `profiles/` scan.

## Debts

- Codex resume: the sub-agent hook payload was never captured (names from the 0.154 roster); a sub-agent pin persists until that terminal's next session.
- Handoff (ADR-0088/0094): upstream formats undocumented (a bump is a refused write); Codex/Grok list a handed-off session only after a restart or by id; Hermes flattens tool calls.
- Hermes: `cli-v1-*` screenshots not regenerated; may write `shell-hooks-allowlist.json`.
- CLI lifecycle: npm data lags native Claude releases; grok uninstall is guided-only; Windows paths out of scope.
