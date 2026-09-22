---
description: Connectors for every agent CLI. PiCode writes each CLI's own MCP config; Pi reads it through the MCP adapter.
---

# MCP

Connectors. Every agent CLI has the same Connectors pane, and PiCode writes
that CLI's own configuration file. Claude Code, Codex and the others read it
on their own; Pi reads it through the **MCP adapter** package.

- **Where:** **Agent CLIs → \<cli\> → Connectors**.
- **Not this:** Pi does not speak MCP itself. On Pi's pane, without the adapter, Connectors is one line and **Open packages** — it does not write files.

The steps below are Pi's; on another CLI skip the adapter and start at step 2.

Canonical: [pi-mcp-adapter](https://github.com/mariozechner/pi-mcp) (install as `npm:pi-mcp-adapter`).

1. Agent CLIs → Pi → **Packages** → Global → install `npm:pi-mcp-adapter`.
2. Agent CLIs → Pi → **Connectors** → **Marketplace**. Search the catalog, set
   **Save to** (global, workspace or agent), and press **Add**. **Docs** on a
   card opens that connector's own page (PiCode's cookbook for the seed;
   the vendor site or repository for the rest). PiCode's own connectors sit
   at the top; the rest is a curated slice of the official MCP Registry.
   **Custom server…** takes a command or URL by hand.
3. Every agent CLI has the same Connectors pane — **Agent CLIs → \<cli\> →
   Connectors** writes that CLI's own config file.
4. A row shows **Live** or **Failed** while an agent is running; with the agent
   stopped the pane says so once instead of guessing. **Sign in** appears on the
   row until the login exists; **•••** holds **Sign out** and **Remove** (the
   confirmation names the file it deletes from).
5. **Sign in** opens the server's login page as soon as you Add or turn it On (hidden once you are signed in). Approve there — the tab returns to PiCode on its own, same as Claude/Codex. You do not need to run an agent first. One login is reused by every agent on this machine. **Sign out** forgets that login on this machine.

Clicking an agent in the sidebar leaves this page and opens that agent.

| Target | File |
|---|---|
| Global | `~/.pi/agent/mcp.json` |
| This workspace | `<folder>/.mcp.json` |
| This agent | only if the agent has its own work folder (`<work>/.pi/mcp.json`) |

| | pi TUI | PiCode |
|---|---|---|
| Adapter | `pi install npm:pi-mcp-adapter` | Packages |
| Servers | edit JSON | Agent CLIs → Connectors |
| Import Cursor/Claude/Codex | adapter CLI | the CLI has its own Connectors pane — add the same service there |

Pi without the adapter → Connectors is one line and **Open packages**. It does not write files.
The page works with a terminal selected; it does not need an agent open.

## Cookbook

The connectors PiCode ships a path for. Each page is install, sign-in if any, and how to revoke. Anything else: **Custom server…** in Marketplace.

| Connector | What it is |
|---|---|
| [Gmail](/guide/mcp-gmail) | Read, draft and send mail. Credentials stay in `~/.gmail-mcp/`, not in PiCode. |
| [DeepWiki](/guide/mcp-deepwiki) | Public GitHub documentation. No account. |
| [PiCode · Computer, PiCode · Browser](/guide/picode-mcp) | PiCode's own tools for Claude Code, Codex and the other CLIs. No account; the switch is in Settings. |

Webhooks are not MCP: [Integrations](/guide/integrations).
