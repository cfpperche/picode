# ADR-0201: OpenCode signs in through PiCode's GUI, by OpenCode's own server

- **Status**: accepted (owner direction 2026-09-23: GUI sign-in per CLI, OpenCode last, "pode seguir")
- **Date**: 2026-09-23
- **Boundary**: process. PiCode starts `opencode serve` on localhost while the dialog needs it, and stops it after 10 idle minutes. Protocol: `GET /api/opencode/catalog`, `POST`/`GET`/`DELETE /api/opencode/credential`, `POST /api/opencode/credential/code`, `add.kind: "opencode"`. OpenCode's catalog rows reach `clicreds` through `SetOpencodeCatalog`.

## Context

OpenCode 1.18.32 ships the HTTP API its own apps use (measured against a scratch HOME):
- `GET /provider` returns models.dev's 223 providers, with their variables and which are connected.
- `GET /provider/auth` returns the login methods its plugins add, each with prompts (select or text, with `when` rules).
- `PUT /auth/{id}` stores a key, with the prompts' answers as metadata.
- `POST /provider/{id}/oauth/authorize` returns `{url, method: auto|code, instructions}`, and `…/callback` finishes the flow: on its own for "auto", or with the pasted code for "code".

OpenCode writes its own `auth.json` in every case. Before this ADR, PiCode's OpenCode pane offered its 11 declared providers, a key form and a terminal login.

## Decision

The OpenCode dialog walks the flow `opencode auth login` walks, through OpenCode's own server:
1. Pick a provider from OpenCode's catalog.
2. Pick one of OpenCode's methods for it. A provider with no plugin gets the key door OpenCode defaults to.
3. Answer its prompts.
4. Then either:
   - paste a key (`PUT /auth`), or
   - open the page, and either wait for an "auto" flow to finish or paste the code a "code" flow asks for.

One sign-in runs at a time. What OpenCode stored is filed in the vault, with OpenCode's ids mapped to the vault's (`openai`→`openai-codex`, …). "Sign in from a terminal" stays as the fallback.

## Consequences

OpenCode's whole catalog, and its own login methods (ChatGPT browser/headless, Copilot, GitLab, Poe, xAI…), sign in without a terminal, and OpenCode keeps owning its store. The price is a dependency on OpenCode's server API. If a route changes, the dialog shows OpenCode's own error, and the terminal fallback still works. A browser OAuth whose callback is `localhost` on the PiCode machine needs the browser there, as with Codex.

## Alternatives considered

- **Drive `opencode auth login` through a pseudo terminal.** Rejected: the server API is the structured version of the same flow.
- **Write `auth.json` directly.** Rejected: that is the reverse engineering the API makes unnecessary.
