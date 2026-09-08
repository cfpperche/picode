# Providers

PiCode reads and writes the same `~/.pi/agent/auth.json` as the pi TUI. Keys are never shown again after save.

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
