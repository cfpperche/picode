---
description: Give Claude Code, Codex and the other agent CLIs PiCode's own tools over MCP.
---

# PiCode tools for other agent CLIs

The tools a pi agent gets from PiCode's packages — the computer, the work
browser — reach Claude Code, Codex, OpenCode and the other agent CLIs over
MCP. The CLI runs `picode mcp <family>` as a small local server; the server
asks PiCode, and PiCode decides exactly what it decides for a pi agent: who
is asking, and whether you switched them on.

- **Where:** **Agent CLIs → the CLI → Launch settings → PiCode tools** for terminals you open from PiCode; **Agent CLIs → the CLI → Connectors** for the CLI's own config files.
- **Not this:** pi does not need it. Pi keeps its packages ([Computer use](/guide/computer-tool), [Browser tools](/guide/browser-tool)), which also draw the capture in the chat.

## What the CLI gets

| Family | Tool | What it does |
|---|---|---|
| Computer | `computer` | The Windows desktop through the desktop app: screenshot, windows, click, type, clipboard, open. Same twenty-three actions as [Computer use for pi](/guide/computer-tool). |
| Browser | `browser` | The page open in PiCode's work browser: snapshot, screenshot, events; acting needs a grant. Same verbs as [Browser tools for pi](/guide/browser-tool). |

Same names, same parameters, same answers. A prompt written for the pi tool
works unchanged in Claude Code.

## Who the CLI is, and what it may do

A CLI opened in a PiCode terminal is that **terminal**. PiCode's switches
apply to it as they apply to any agent: **Settings ▸ Computer** lists the
terminal, and the computer tool refuses until you turn it on; **Settings ▸
Browser** gives it the read tier until you grant more. A new terminal is a
new row. A CLI opened outside PiCode has no identity, and the tool says so
instead of guessing.

The server carries no credential of its own: it reads the install token
the way every PiCode script does, and every call is one request to the
running daemon.

## Three places to turn it on

| Scope | Where | What it writes |
|---|---|---|
| This terminal (and the ones launched with the same settings) | Launch settings → PiCode tools | Nothing in your files: PiCode adds the servers to the launch command (`--mcp-config` for Claude Code, `-c` for Codex, inline configuration for OpenCode). |
| This workspace | Connectors → Add connector → PiCode · Computer / Browser → save to the workspace | The CLI's project file (`.mcp.json`, `.codex/config.toml`, …). |
| This machine | Connectors → same cards → save to this machine | The CLI's user config, through the CLI's own command where it has one. |

The server has the same name in every place (`picode-computer`,
`picode-browser`), so a CLI that already has it from a file does not get it
twice from the launch.

Launch settings offer the group only for the CLIs PiCode can inject into
today: Claude Code, Codex and OpenCode. For Grok, Hermes Agent, Muse Code,
Antigravity and Omp use the Connectors pane.

## By hand

Any MCP client can run the server; the CLI still needs to be inside a
PiCode terminal to have an identity.

```bash
claude mcp add picode-computer -- picode mcp computer
claude mcp add picode-browser -- picode mcp browser
```

`picode mcp computer browser` serves both families from one process.

## What it will not do

- Give a CLI anything a pi agent could not get: the grant, the tier and the
  audit are the daemon's, and they are the same.
- Work from another machine: the server talks to the daemon on this one.
- Replace pi's packages: pi keeps its native tools and the capture in the
  chat.

## Where it goes next

Inbox (`notify_human`, `ask_human`) and Checklist as further families, and
launch injection for the CLIs that gain a mechanism for it.
