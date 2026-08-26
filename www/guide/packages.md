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

| | pi TUI | PiCode |
|---|---|---|
| Machine / project | `pi install` / `-l` | same |
| One agent | — | **This agent** |
