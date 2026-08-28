# Packages

Extras for Pi (search, tools, themes). They run with **full access** — only install what you review.

Canonical: [pi packages](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/packages.md).

In PiCode (`#/packages`) pick a target, then Install.

| Target | Who gets it | How |
|---|---|---|
| This machine | Every agent on this computer | `pi install` |
| This workspace | Every agent in this folder | `pi install -l` |
| This agent | Only that agent, every session | PiCode remembers it and starts pi with `-e` |

This agent is PiCode-only. pi has no per-agent install. Restart the agent after adding or removing.

If a package on this machine or in this folder is behind, that row shows **Update**. The user menu marks Packages. Nothing updates until you click. Git and pinned versions stay as they are.

**Only this agent's packages** (checkbox): skip machine and folder packages. The agent starts with its own list, or none. Also skips skills/themes discovered from those places. Restart to apply.

| | pi TUI | PiCode |
|---|---|---|
| Machine / project | `pi install` / `-l` | same |
| One agent | — | **This agent** |

Search: install `npm:pi-web-search` (This machine). Chat expands `web_search` into source cards. Needs a model that supports native search (Gemini / OpenAI / Anthropic).

MCP servers: install `npm:pi-mcp-adapter`, then `#/mcps`. Guide: [MCP](/guide/mcp).
