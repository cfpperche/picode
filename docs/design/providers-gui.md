# Providers GUI — complete login management

> Status: draft 2026-08-25. Source: pi `docs/providers.md` (canonical),
> TUI `/login` (account vs API key), ADR-0009 (amended). Not invented.

Today `#/providers` is a **roster of keys already in `auth.json`**. The TUI
`/login` is a **wizard**: pick provider → pick method → complete auth.
PiCode must become the wizard. Credentials stay in `~/.pi/agent/auth.json`.

## What the TUI actually does

1. Pick a **provider** (full loginable set, not only signed-in).
2. If the provider supports both: **Sign in with an account** vs **API key**.
3. Account = OAuth / subscription (browser or paste-callback).
4. API key = secret prompt → `{ "type": "api_key", "key" }` in `auth.json`.
5. `/logout` deletes that provider’s entry.

Subscriptions (account): Codex (ChatGPT Plus/Pro), Claude Pro/Max,
GitHub Copilot, xAI subscription, OpenRouter PKCE, Radius.

API key: Anthropic, OpenAI, Gemini, Groq, Mistral, DeepSeek, xAI, OpenRouter,
ZAI, OpenCode, Hugging Face, Fireworks, Together, Bedrock, … (full table in
[pi Providers](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/providers.md)).

Some ids support **both** (xAI, OpenRouter, Anthropic).

## Target IA (`#/providers`)

**Add provider** (primary, empty-state + header). Dialog, not TUI.

1. Searchable list of **all loginable providers** (not only `auth.json`).
   Source: catalog + a static method map we maintain from pi docs
   (`oauth | api_key | both`). Unknown catalog ids → API key only.
2. If `both` or `oauth`: choose **Account** or **API key** (same copy as TUI).
3. **API key** → password field, Zod, PUT `auth.json`. Never echo the key.
4. **Account** → until pi RPC `login` exists: honest dead-end
   (“Subscriptions need the pi TUI `/login`”) **or** a later OAuth worker.
   Do not fake a browser OAuth we do not own.
5. Roster: every saved entry with **method badge** (`api_key` | `oauth`),
   Sign out, Replace key. Empty: “No providers yet” + Add.

`/login` stays `#/providers` and should **open Add provider**.

## Phases

| P | Ship | Not |
|---|---|---|
| **P0** | Add provider; full list; method picker; API key save; roster + type badge; `/login` opens Add | OAuth |
| **P1** | llama.cpp row (path, optional key) on this page | — |
| **P2** | OAuth: only if pi adds RPC `login`, or we accept a PiCode PKCE helper (new ADR) | Reimplement 6 OAuth stacks silently |

## Constraints

- Same file as pi (`auth.json`). No second store until it hurts (your call).
- GET catalog never returns key material. Type (`api_key`/`oauth`) is OK.
- Zod + `noValidate`. `--ctl-h` on the dialog row.
- Related pi doc on the public `/commands#login` page.

## Open questions (need you only if we leave the default)

- Default for `both`: land on **API key** (GUI-complete) with Account as
  secondary. OK?
- Copilot Enterprise hostname: skip until P2.
