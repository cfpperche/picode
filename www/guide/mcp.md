# MCP

Pi does not speak MCP itself. PiCode writes the files the **MCP adapter** package reads.

Canonical: [pi-mcp-adapter](https://github.com/mariozechner/pi-mcp) (install as `npm:pi-mcp-adapter`).

1. `#/packages` → This machine → install `npm:pi-mcp-adapter`.
2. `#/mcps` → Add a server (command or URL). **More** adds environment variables, headers, or sign-in (OAuth / token).
3. **Use from…** is a tree of apps and their servers. Check the ones to use. It does not copy files.
4. The list says **Idle** until an agent uses the server, then **Live** (or **Failed**). A login on this machine has **Sign out** instead of Sign in.
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
| Servers | edit JSON | `#/mcps` |
| Import Cursor/Claude/Codex | adapter CLI | **Use from…** (mirror, pick servers) |

No adapter → `#/mcps` is one line and **Open packages**. It does not write files.
