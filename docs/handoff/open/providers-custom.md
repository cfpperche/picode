# Providers: custom endpoints (ADR-0129)

## Next

- Other CLIs (Claude Code, Codex, Grok) have their own custom-endpoint files; each needs its own plan.

## Debts

- Custom endpoints: levels/context/max per list (P2); `off` on pi default.
- Verify-for-built-ins (pi `auth check`) still answers from credential presence, so a bogus env-backed key reads green there — only custom endpoints spend a real request now.
- Hand-written `models.json` without the `providers` wrapper is invalid for pi and invisible to both tools; PiCode refuses to adopt it.
- Verify-from-the-dialog needs a model id typed before it can ask (a saved definition always has one: the schema refuses a list with none), so the pre-save check says "Add a model id first" rather than probing blind.
- Listing and probing speak the four API types the form offers; a hand-edited `models.json` with another `api` value is listed in the catalog but its Verify refuses and names what it can speak (adding a shape means teaching `internal/modellist` that shape).
