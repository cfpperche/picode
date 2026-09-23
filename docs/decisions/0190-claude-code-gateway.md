# ADR-0190: Claude Code through an Anthropic-compatible gateway, from the GUI

- **Status**: accepted (owner approved 2026-09-22, in session; slice 3 of ADR-0187)
- **Date**: 2026-09-22
- **Boundary**: persistence. PiCode writes `ANTHROPIC_BASE_URL`, `ANTHROPIC_AUTH_TOKEN` and the model keys into the `env` block of Claude Code's user settings, the same block as ADR-0189. Security model: a Console key PiCode would inject must never reach a third-party gateway.

## Context

Z.ai, Kimi, DeepSeek, MiniMax and other providers publish an Anthropic-compatible URL for Claude Code. The documented setup is `ANTHROPIC_BASE_URL` plus `ANTHROPIC_AUTH_TOKEN`, usually in `settings.json`'s `env`. (`CLAUDE_CODE_USE_GATEWAY` is a different thing: Anthropic's enterprise "Cloud gateway", connected through `/login`.)

Measured on 2.1.280 against a local server that captured the headers:
- a gateway set only in the settings is honoured, and it wins over the subscription file;
- with `ANTHROPIC_API_KEY` also in the environment, **Claude Code sends both**: `Authorization: Bearer <gateway token>` *and* `x-api-key: <the Anthropic key>`. Injecting a Console key next to a gateway would leak it to a third party.

## Decision

A gateway is a fourth kind in ADR-0189's mechanism. `PUT /api/claude-code/platform {kind: "gateway"}` writes:
- the URL, which must be https (plain http only for this machine);
- the token;
- optionally `ANTHROPIC_MODEL` and a fast model (`ANTHROPIC_DEFAULT_HAIKU_MODEL`).

It removes every platform's keys. A gateway is detected as a URL **with** a token beside it.

| Case | Effect |
|---|---|
| gateway in use | no Console key is injected (a platform in use already stops it) |
| Use on the subscription or a key | the gateway's URL, token and model are removed |
| a URL with no token (the person's own proxy) | not a gateway; no platform save, stop or Use touches it |

## Consequences

Claude Code can run on any Anthropic-compatible provider without a terminal, and PiCode's one-login rule doubles as the guard against the key leak above. The token lives in Claude Code's settings, like a platform's. Presets for known providers are deliberately left out until their URLs are verified: the form asks for the URL the provider publishes.

## Alternatives considered

- **Inject the gateway at launch.** Rejected for the reasons ADR-0189 gives.
- **Treat any `ANTHROPIC_BASE_URL` as a gateway.** Rejected: it would erase a corporate proxy the person configured.
