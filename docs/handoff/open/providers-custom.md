# Providers: custom endpoints (ADR-0129)

## Next

- Other CLIs (Claude Code, Codex, Grok) have their own custom-endpoint files; each needs its own plan.

## Debts

- Custom endpoints: thinking levels are one selection for all listed models (window/max-output went per-model); `off` on pi default.
- Verify-for-built-ins (pi `auth check`) answers from credential presence — a bogus env-backed key reads green; only custom endpoints spend a real request.
- Hand-written `models.json` without the `providers` wrapper is invalid for pi and invisible to both tools; PiCode refuses to adopt it.
- Verify-from-dialog needs a typed model id first (a saved definition always has one), so the pre-save check says "Add a model id first" instead of probing blind.
- Listing/probing speak the four API types the form offers; a hand-edited `models.json` with another `api` value lists but its Verify refuses and says so (new shape = teach `internal/modellist`).
