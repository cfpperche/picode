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

**Add account** keeps extra logins in PiCode (`~/.picode/accounts.json`). pi still sees **one** active slot in `auth.json` — **Use** copies that login into the slot. Two agents cannot use two Claudes at the same time (pi limitation).

## Usage

Claude, Codex, Copilot, Kimi and xAI rows signed in with an **account** show
**Usage**. ZAI and OpenCode Go show it for an **API key**. The dialog is that
plan's windows (5 hours, 7 days, week, extra) for the active login — the same
slot pi uses. A provider without a plan meter has no button.

If Codex or Grok has a banked reset, the dialog shows it. **Redeem** spends
that credit (you confirm first) and clears the current window.

Refresh reloads. The numbers are the provider's, not a guess from this session.
