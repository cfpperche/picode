---
description: API keys and account logins for Pi on this machine.
---

# Providers

Accounts apply to this machine. PiCode reads and writes the same
`~/.pi/agent/auth.json` as the pi TUI. Keys are never shown again after save.

- **Where:** **Agent CLIs**, pick **Pi**, then **Providers** (`#/clis/pi/providers`).
- **Not this:** not [Connectors](/guide/mcp) and not [Settings](/guide/settings). Older Providers links redirect here; other CLIs appear when their provider integration is available.

Canonical: [pi Providers](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/providers.md).

## Sign in

**Add provider** (or `/login`) → pick a provider.

| Method | In PiCode |
|---|---|
| API key | All providers that accept a key |
| Account | Claude, Codex, Copilot, Kimi, xAI |
| Either | Claude, Kimi, xAI, OpenRouter, … |

Account login opens a browser tab (Claude, Codex) or shows a device code (Copilot, Kimi, xAI). Radius account login stays in the TUI (needs a gateway URL).

**Add account** keeps extra logins in PiCode (`~/.picode/accounts.json`). pi still sees **one** active slot in `auth.json` — **Use** copies that login into the slot. Two agents cannot use two Claude logins at the same time (pi limitation).

**Pause** keeps a login but takes it out of play. The credential stays; the
account stops being offered. Pausing the one you are using promotes another;
the last live account cannot be paused — that is **Sign out**.

Sign out says what it breaks: the confirm names the agents and automations
configured on that provider.

## Custom endpoint

**Add provider → Custom endpoint** adds a gateway pi does not know out of
the box — OpenRouter-style aggregators, prepaid wallets, self-hosted routers. The form
asks for a name, the base URL, the API key and the model ids exactly as the
gateway spells them.

| Field | Goes to |
|---|---|
| Name, Base URL, API type, models, Advanced | `~/.pi/agent/models.json` — pi's own provider definitions |
| API key | `~/.pi/agent/auth.json` — the same file as every other sign-in |

The key is never written into the definition. **Edit endpoint** reopens the
form (a blank key keeps the saved one); **Sign out** removes the key and
keeps the definition; **Remove endpoint** deletes both, naming what still
uses the provider. Hand-edited `models.json` entries are preserved: PiCode
merges by name and never touches providers it does not own. Overriding a
built-in provider's URL is not offered here.

Model ids must match the gateway exactly — copy them from its model list.
The Advanced section selects the API type (Anthropic-compatible gateways
want **Anthropic Messages** and a base URL without `/v1`) and the
compatibility flags for gateways that reject OpenAI-only request fields.

Canonical: [pi Custom Models](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/models.md).

## Where a login comes from

A provider you never signed into here can still be signed in: if its API-key
environment variable is set, pi reads it. Those rows say so, named by the
variable (`GROQ_API_KEY`), and have no Sign out — change the variable, not
this page.

## Verify

**Verify with pi** in a row's ⋯ menu asks pi whether it could use that
provider right now (`pi auth check`). It costs nothing: no model call, no
token, and an expired login is reported rather than refreshed.

## Usage

Claude, Codex, Copilot, Kimi and xAI rows signed in with an **account** show
plan windows. ZAI, OpenCode Go, OpenRouter and MiniMax show them for an
**API key**. A provider without a plan meter shows none.

The windows are on the row itself:

| What you see | What it means |
|---|---|
| A bar and **live** | fetched within the last few minutes |
| A bar and **12m old** | the same number, that old. Click it to re-check |
| **not checked** | never fetched on this machine. Click **Check** |
| **sign in again** | the login expired |
| **no plan windows** | this provider publishes no quota |
| An amount, such as `$4.10 left` | a balance, not a percentage. It has no ceiling to draw a bar against |

PiCode refreshes only the **active** account of each provider, every few
minutes. Everything else is fetched when you ask. The numbers are the
provider's own; nothing here is estimated from your session.

**Usage** opens the full dialog: every window, the reset times and banked
resets. If Codex or Grok has one, **Redeem** spends that credit (you confirm
first) and clears the current window.
