# MCP

Pi does not speak MCP itself. PiCode writes the files the **MCP adapter** package reads.

Canonical: [pi-mcp-adapter](https://github.com/mariozechner/pi-mcp) (install as `npm:pi-mcp-adapter`).

1. `#/packages` → This machine → install `npm:pi-mcp-adapter`.
2. `#/mcps` → Add a server (command or URL). **More** adds environment variables, headers, or sign-in (OAuth / token).
3. **Use from…** is a tree of apps and their servers. Check the ones to use. It does not copy files.
4. Restart the agent. The list says **Idle** until that agent uses the server, then **Live** (or **Failed** / **Sign in**).

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
