---
description: Connectors. PiCode writes the files the MCP adapter reads. Pi does not speak MCP itself.
---

# MCP

Connectors. PiCode writes the files the **MCP adapter** package reads.

- **Where:** **Agent CLIs → Pi → Connectors**.
- **Not this:** Pi does not speak MCP itself. Without the adapter, Connectors is one line and **Open packages** — it does not write files.

Canonical: [pi-mcp-adapter](https://github.com/mariozechner/pi-mcp) (install as `npm:pi-mcp-adapter`).

1. Agent CLIs → Pi → **Packages** → This machine → install `npm:pi-mcp-adapter`.
2. Agent CLIs → Pi → **Connectors** → **Add connector**. The dialog lists the
   catalog; search it, set **Save to** (machine, workspace or agent), and press
   **Add**. **Custom server…** takes a command or URL by hand; **Import a
   file…** reads a connector definition JSON and asks you to review it first.
3. Servers another app already has (Claude Code, Cursor, Codex, …) show up
   under **Source** in the same dialog; adding one asks before it writes.
4. A row shows **Live** or **Failed** while an agent is running; with the agent
   stopped the pane says so once instead of guessing. **Sign in** appears on the
   row until the login exists; **•••** holds **Sign out** and **Remove** (the
   confirmation names the file it deletes from).
5. **Sign in** opens the server's login page as soon as you Add or turn it On (hidden once you are signed in). Approve there — the tab returns to PiCode on its own, same as Claude/Codex. You do not need to run an agent first. One login is reused by every agent on this machine. **Sign out** forgets that login on this machine.

Clicking an agent in the sidebar leaves this page and opens that agent.

| Target | File |
|---|---|
| This machine | `~/.pi/agent/mcp.json` |
| This workspace | `<folder>/.mcp.json` |
| This agent | only if the agent has its own work folder (`<work>/.pi/mcp.json`) |

| | pi TUI | PiCode |
|---|---|---|
| Adapter | `pi install npm:pi-mcp-adapter` | Packages |
| Servers | edit JSON | Agent CLIs → Connectors |
| Import Cursor/Claude/Codex | adapter CLI | **Use from…** (mirror, pick servers) |

No adapter → Connectors is one line and **Open packages**. It does not write files.
The page works with a terminal selected; it does not need an agent open.

## Cookbook

The connectors PiCode ships a path for. Each page is install, sign-in if any, and how to revoke. Anything else: **Custom server…** or **Import a file…** above.

| Connector | What it is |
|---|---|
| [Gmail](/guide/mcp-gmail) | Read, draft and send mail. Credentials stay in `~/.gmail-mcp/`, not in PiCode. |
| [DeepWiki](/guide/mcp-deepwiki) | Public GitHub documentation. No account. |
| [PiCode · Computer, PiCode · Browser](/guide/picode-mcp) | PiCode's own tools for Claude Code, Codex and the other CLIs. No account; the switch is in Settings. |

Webhooks are not MCP: [Integrations](/guide/integrations).
