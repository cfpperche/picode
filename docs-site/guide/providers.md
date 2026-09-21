---
description: API keys and account logins for Pi on this machine.
---

# Providers

Accounts apply to this machine. Everything PiCode saves here lives in one
encrypted vault: `~/.picode/credentials.json` next to the key that opens it
(`credentials.key`). Keys are never shown again after save.

- **Where:** **Agent CLIs**, pick a CLI, then **Providers** — `#/clis/pi/providers`
  for Pi, `#/clis/codex/providers` for Codex, and so on for Claude Code,
  OpenCode, Grok, Hermes, Muse, Antigravity and Omp.
- **Not this:** not [Connectors](/guide/mcp) and not [Settings](/guide/settings). Older Providers links redirect here.

Pi keeps its own file (`~/.pi/agent/auth.json`) and PiCode reads and writes
exactly that, like the pi TUI. Canonical:
[pi Providers](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/providers.md).

## Sign in

**Add provider** (or `/login`) → pick a provider.

| Method | In PiCode |
|---|---|
| API key | All providers that accept a key |
| Account | Claude, Codex, Copilot, Kimi, xAI |
| Either | Claude, Kimi, xAI, OpenRouter, … |

Account login opens a browser tab (Claude, Codex) or shows a device code (Copilot, Kimi, xAI). Radius account login stays in the TUI (needs a gateway URL).

**Add account** keeps extra logins in the vault. pi still sees **one** active
slot in `auth.json` — **Use** copies that login into the slot. Two agents
cannot use two Claude logins at the same time (pi limitation).

**Pause** keeps a login but takes it out of play. The credential stays; the
account stops being offered. Pausing the one you are using promotes another;
the last live account cannot be paused — that is **Sign out**.

Sign out says what it breaks: the confirm names the agents and automations
configured on that provider.

## Other agent CLIs

Claude Code, Codex, OpenCode, Grok, Hermes, Muse, Antigravity and Omp have the
same pane for the providers they can use, with four actions:

| Action | What it does |
|---|---|
| **Import** | reads the login that CLI already has on this machine and stores a copy in the vault. The CLI's own file is never changed, and nothing is activated by importing |
| **Add API key** | stores a key for a provider this CLI can use |
| **Verify** | spends exactly one listing call to the provider with the stored key — the button says so — and remembers the answer with its age |
| **Use** | writes that account into the CLI's own login file, so the CLI runs on it — the same thing **Use** does for Pi |
| **Sign out** | removes that account from the vault |

**Use** never changes where the CLI keeps its state: no new folder, no
`HOME`-style variable, nothing about settings, sessions or memory moves. It
replaces one file — the CLI's own login file — and PiCode keeps a copy of what
was there the first time it writes (the path is shown when it happens).

Two rules come with that:

- **One account per CLI is live at a time.** Switching affects terminals you
  open afterwards, and only one login can sit in the file. Running two accounts
  of the same CLI side by side would need a separate home per account, which
  this deliberately does not do.
- **Use is refused while a terminal of that CLI is running** (the refusal says
  how many). Replacing a credential under a running agent is how sessions get
  corrupted.

Some limits are the vendors', not PiCode's, and the pane says so on the row:

- **Omp** keeps its accounts in its own database, which PiCode does not read.
  Its pane manages API keys; a subscription login stays with Omp.
- **Muse Code** and **Antigravity** keep their login somewhere PiCode does not
  write, so their panes list what exists here and name what they cannot do.
- When a login carries no account name (Claude, Muse, Antigravity), PiCode
  keeps **one** subscription login per provider — the pane says so on the row.
- A subscription login that a CLI renews by itself may only have one live
  copy on a machine. Importing it while the CLI keeps using it will end with
  one of the two asking for a fresh sign-in; the pane warns before that
  happens.

## Custom provider

**Add provider → Custom provider** opens a page that adds a gateway pi
does not know out of the box — OpenRouter-style aggregators, prepaid wallets,
self-hosted routers. The form asks for a name, the base URL, the API key and
the model ids exactly as the gateway spells them. Use lowercase letters,
digits and dashes for the name — it becomes the provider id.

| Field | Goes to |
|---|---|
| Name, Base URL, API type, models, Advanced | `~/.pi/agent/models.json` — pi's own provider definitions |
| API key | `~/.pi/agent/auth.json` — the same file as every other sign-in |

The key is never written into the definition. **Edit provider** reopens the
form (a blank key keeps the saved one); **Sign out** removes the key and
keeps the definition; **Remove provider** deletes both, naming what still
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

The other CLIs' panes verify differently — one listing call to the provider,
named on the button — because those CLIs have no equivalent command. See
[Other agent CLIs](#other-agent-clis).

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
