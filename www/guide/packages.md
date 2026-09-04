# Packages

Extras for Pi (search, tools, themes). They run with **full access** — only install what you review.

Canonical: [pi packages](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/packages.md).

In PiCode (`#/packages`) pick a target, then Install. **Installed** lists what each target has (Update when a newer version exists, Remove); **Marketplace** is the pi.dev gallery.

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

Checklist: install `packages/pi-checklist` from the PiCode repository so an agent plans before it changes anything and the sidebar shows its current step. Guide: [Checklist](/guide/checklist).

Compact earlier: install `packages/pi-compact` so long sessions summarize at 100k tokens or half the window, with a cheap model, instead of waiting for the edge. Guide: [Compact earlier](/guide/compact).
