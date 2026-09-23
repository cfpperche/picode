# ADR-0191: Codex signs in through PiCode's GUI, by Codex's own app-server

- **Status**: accepted (owner direction 2026-09-22: every CLI signs in from the GUI with a dialog shaped like its own login; "pode seguir pro codex")
- **Date**: 2026-09-22
- **Boundary**: process. PiCode starts `codex app-server` (JSON-RPC over stdio) for the duration of a sign-in and speaks its `account/login/*` API. Persistence: after a non-Bedrock login, PiCode removes a leftover `model_provider = "amazon-bedrock"` from `~/.codex/config.toml`. Protocol: `POST`/`GET`/`DELETE /api/codex/login`, `DELETE /api/codex/platform`, `add.kind: "codex"`, and the roster's `platform` for Codex.

## Context

Codex 0.156.0's `/login` offers four options: Sign in with ChatGPT, Sign in with Device Code, Provide your own API key, and Use Amazon Bedrock. Codex ships the API its own apps use for exactly these. Measured with fake values in a scratch `CODEX_HOME`:
- `codex app-server`'s `account/login/start` takes `apiKey`, `chatgpt` (returns `authUrl`; Codex serves the `localhost:1455` callback), `chatgptDeviceCode` (returns `verificationUrl` + `userCode`), `amazonBedrock` and `amazonBedrockAccessKeys`. The two Bedrock types require the `experimentalApi` capability.
- Codex writes its own `auth.json` (`auth_mode`: `apikey`, `chatgpt`, `bedrockApiKey`…) and, for Bedrock, `model_provider = "amazon-bedrock"` in `config.toml`.
- Signing in with an API key afterwards **leaves** `model_provider = "amazon-bedrock"` in place, so the next session would point at Bedrock with no Bedrock credentials.

## Decision

The Codex dialog mirrors Codex's `/login` and drives the app-server:

| Door | What PiCode does |
|---|---|
| ChatGPT (browser) | `login/start chatgpt` → open `authUrl` → wait for `account/login/completed` (10-minute window) → import into the vault |
| Device code | same, showing `verificationUrl` and `userCode`; the door for a phone or another machine |
| API key | `login/start apiKey` → Codex writes `auth.json` → import into the vault |
| Amazon Bedrock | `login/start amazonBedrock` / `…AccessKeys`; Codex writes its files; the pane shows "Codex uses Amazon Bedrock · region" |
| Stop using Bedrock | Codex's own `account/logout`, then the `model_provider` line is removed |
| Any login that is not Bedrock, including Use on a vault row | removes `model_provider` only when it equals `"amazon-bedrock"`, keeping the rest of `config.toml` (format-preserving edit) |

Only one sign-in runs at a time (409 otherwise). "Sign in from a terminal" stays as the fallback.

## Consequences

Codex writes every file in its own format, so PiCode never reverse-engineers `auth.json`. The price is a dependency on an API Codex marks partly experimental (the Bedrock types): if it changes, the door returns Codex's own error and the terminal fallback still works.

The browser door needs the browser to reach `localhost:1455` on the machine running PiCode. The dialog says so, and offers the device code for everything else.

## Alternatives considered

- **PiCode's own OAuth engine** (it knows Codex's client id). Rejected: Codex's own flow also records the `id_token`, plan and account claims Codex reads, and it covers the API key and Bedrock the same way.
- **Write `auth.json` directly.** Rejected: that is the reverse-engineering the app-server makes unnecessary.
