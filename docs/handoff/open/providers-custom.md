# Providers: custom endpoints (ADR-0129)

## Next

- P1: **Load models** — server fetches `GET {baseUrl}/models` with the key; 401/402/404 name the fix. P2: URL hints per API type (Anthropic-compatible wants no `/v1`).
- P4: Verify that spends a real cheap request (Cursor/Raycast bar) — owner call, changes server semantics.
- Other CLIs (Claude Code, Codex, Grok) have their own custom-endpoint files; each needs its own plan.

## Debts

- Thinking levels, context window and max output are one value for every model
  in the list (per-model fields are P2); the form writes `reasoning` and
  `thinkingLevelMap`, and `off` is left to pi's default because its provider
  value is not always the level name (`off` vs `none` in pi's catalogs).
- Tall create dialogs scroll as a whole (`.dlg-create` capped at `80dvh`), so
  Back/Save can sit below the fold on a short window; a sticky action row is
  the follow-up.
- Verify-with-pi answers from credential presence, so a bogus env-backed key reads green — pre-existing roster debt.
- Hand-written `models.json` without the `providers` wrapper is invalid for pi and invisible to both tools; PiCode refuses to adopt it.
