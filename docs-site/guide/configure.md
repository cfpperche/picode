---
description: Which screen edits which file, and the one workflow that does not brick a deploy.
---

# Configure

PiCode has two settings worlds. Do not mix them.

::: tip If you only remember one thing
**Agent CLIs → Settings** edits pi JSON, one layer at a time. **Preferences** is PiCode chrome (theme, server). A workspace write needs the folder trusted. Restart the agent after Packages or Connectors.
:::

- **Where:** **Agent CLIs** (last icon in the desktop sidebar header, or **More** on a phone), then the pane named below. Preferences is the user menu.
- **Not this:** not [from source](/guide/from-source), and not a compose YAML you mount into Docker.

## What to open

**Providers / API keys** — Agent CLIs → Pi → [Providers](/guide/providers). Writes `~/.pi/agent/auth.json` (extra logins in `~/.picode/accounts.json`).

**Agent behavior** — Agent CLIs → Pi → [Settings](/guide/settings). One layer at a time: this machine, the workspace, or the agent.

**Keyboard map** — Agent CLIs → [Keyboard](/guide/keyboard). Writes `~/.pi/agent/keybindings.json` (one map per machine).

**Packages / MCP** — Agent CLIs → [Packages](/guide/packages) / [Connectors](/guide/mcp).

**Theme, server port** — Preferences. PiCode chrome, not pi.

**Bind address, Docker socket** — `picode install --env KEY=VALUE`. systemd drop-in; later updates leave it alone.

## The change workflow

1. Pick the item above. That is the only screen you edit.
2. Save. Keys are never shown again after save.
3. Packages or Connectors: restart the agent.
4. Workspace Settings: the folder must be in pi's trust list (`/trust` in the TUI). Untrusted writes return 409.

## Worked example: add a custom endpoint

1. Agent CLIs → Pi → **Providers** → **Add provider** → **Custom endpoint**.
2. Name, base URL, API key, and model ids exactly as the gateway spells them.
3. Save, then pick that provider on the agent.

Full fields: [Providers](/guide/providers). Automatic model routing is optional: [Model roles](/guide/roles).
