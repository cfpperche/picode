# Providers: custom endpoints (ADR-0129)

## Next

- Other CLIs (Claude Code, Codex, Grok) have their own custom-endpoint files; each needs its own plan.

## Debts

- Custom endpoints: levels/context/max per list (P2); `off` on pi default; tall dialogs scroll (sticky actions owed).
- Verify-for-built-ins (pi `auth check`) still answers from credential presence, so a bogus env-backed key reads green there — only custom endpoints spend a real request now.
- Hand-written `models.json` without the `providers` wrapper is invalid for pi and invisible to both tools; PiCode refuses to adopt it.
- The probe spends a real request on a hand-written `chat-template` gateway whose listing is disabled — nothing verifies those without a model id (P4's edge).
- `internal/modellist` speaks `openai-completions`, `openai-responses`, `anthropic-messages` and `google-generative-ai`; a provider using another compat shape (`string-thinking` on a non-OpenAI transport) is listed but its verify says so rather than guessing.
