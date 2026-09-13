# Providers: custom endpoints (ADR-0129)

## Next

- P1: **Load models** — server fetches `GET {baseUrl}/models` with the pasted key; 401/402/404 name the fix (key / wallet / URL). P2: URL hints per API type (Anthropic-compatible wants no `/v1`).
- P4: Verify that spends a real cheap request (Cursor/Raycast bar) — owner call, changes server semantics.
- Other CLIs (Claude Code, Codex, Grok) have their own custom-endpoint files; each needs its own plan (ADR-0103 keeps Providers Pi-only).

## Debts

- Verify-with-pi answers from credential presence (`pi auth check`), so a bogus env-backed key still reads green — pre-existing roster debt, unchanged by the custom flow.
- Hand-written `models.json` without the `providers` wrapper is invalid for pi and invisible to both tools; PiCode refuses to adopt it rather than guessing.
