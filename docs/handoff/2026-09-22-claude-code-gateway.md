# 2026-09-22 — feat/claude-code-gateway: Claude Code through an Anthropic-compatible gateway

Shipped: ADR-0190 (owner-approved, slice 3 of ADR-0187). "3rd-party platform" in Claude Code's sign-in dialog gains an "Anthropic-compatible gateway" option, with fields for the gateway URL (https, or http only for this machine), a token, and optional model and fast model. It uses ADR-0189's `env` block (`ANTHROPIC_BASE_URL`, `ANTHROPIC_AUTH_TOKEN`, `ANTHROPIC_MODEL`, `ANTHROPIC_DEFAULT_HAIKU_MODEL`). The pane shows "Claude Code uses a custom gateway · <host>".

A gateway counts as one only when a token sits next to the URL. A URL with no token is the person's own proxy, and no platform save, stop or Use touches it. The subtitle of the credential dialogs gets a 12px gap (`.dlg-body + .cred-form`), which also fixes the old key form (Codex/OpenCode).

Why the rule matters: measured on 2.1.280 against a local header-capturing server, Claude Code sends `ANTHROPIC_API_KEY` to the gateway as `x-api-key` alongside the gateway's own `Authorization: Bearer` token. With a gateway in use, the one-login rule injects no Console key.

Verified: `make ci-scoped` PASS. `TestClaudeCodeGateway` covers the proxy left untouched, the gateway env, no key riding with the gateway, Use on the subscription removing the gateway, and the refusals. Scratch QA ran with an isolated HOME on desktop and mobile, overlay audit ok, and a recheck confirmed the subtitle gap in three dialogs.
visual-review: PASS (ccg-2d-toast.png, ccg-5c-form.png, ccg2-a-desktop.png, ccg2-b-codex.png; card 5/5)

Not verified: a real gateway end to end. Presets for known gateways were left out until their URLs are verified.

## Next up

- Next CLI onto a GUI sign-in dialog of its own (per-CLI, shaped like its native login): Codex, Grok, Hermes, Muse, Antigravity; OpenCode last (owner's order)

Merge: fast-forward ready.
