# Providers: custom endpoints (ADR-0129)

## Next

- Each other CLI (Claude Code, Codex, Grok) needs its own custom-endpoint plan.

## Debts

- Thinking levels are one selection for all listed models; `off` on pi default.
- Verify-for-built-ins answers from credential presence — a bogus env-backed key reads green.
- A `models.json` without the `providers` wrapper is invalid for pi; PiCode refuses to adopt it.
- Verify-from-dialog says "Add a model id first" instead of probing blind when the id is empty.
- Listing/probing speak the four API types the form offers; another `api` value lists but its Verify refuses and says so.
