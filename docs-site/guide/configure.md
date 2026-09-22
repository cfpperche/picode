---
description: Which screen edits which file, and the one workflow that does not brick a deploy.
---

# Configure

PiCode has two settings worlds. Do not mix them.

::: tip If you only remember one thing
**Agent CLIs → the CLI → Settings** edits that CLI's own settings (Pi's JSON one layer at a time). **Preferences** is PiCode chrome (theme, server). A workspace write needs the folder trusted. Restart the agent after Packages or Connectors.
:::

- **Where:** **Agent CLIs** (last icon in the desktop sidebar header, or **More** on a phone), then the pane named below. Preferences is the user menu.
- **Not this:** not [from source](/guide/from-source), and not a compose YAML you mount into Docker.

## What to open

Every CLI — Pi, Claude Code, Codex, Grok, Hermes Agent, OpenCode, Muse Code, Antigravity, Omp — has the same panes under **Agent CLIs → the CLI**.

**Providers / API keys** — [Providers](/guide/providers). One encrypted vault for every CLI; Pi's pane also writes `~/.pi/agent/auth.json`.

**Agent behavior** — [Settings](/guide/settings). Each CLI's own settings file; for Pi, one layer at a time (global, the workspace, or the agent).

**Keyboard map** — [Keyboard](/guide/keyboard). Editable for Pi, Codex, OpenCode, Antigravity and Omp; each writes that CLI's own key file.

**Packages / MCP** — [Packages](/guide/packages) / [Connectors](/guide/mcp).

**Theme, server port** — Preferences. PiCode chrome, not a CLI setting.

**Bind address, Docker socket** — `picode install --env KEY=VALUE`. systemd drop-in; later updates leave it alone.

## The change workflow

1. Pick the item above. That is the only screen you edit.
2. Save. Keys are never shown again after save.
3. Packages or Connectors: restart the agent.
4. Pi's workspace Settings: the folder must be in pi's trust list (`/trust` in the TUI). Untrusted writes return 409.

## Worked example: add a custom provider to Pi

1. Agent CLIs → Pi → **Providers** → **Add provider** → **Custom provider**.
2. Name, base URL, API key, and model ids exactly as the gateway spells them.
3. Save, then pick that provider on the agent.

Full fields: [Providers](/guide/providers). Automatic model routing is optional: [Model roles](/guide/roles).
