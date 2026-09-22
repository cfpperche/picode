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
| Browser | `browser` | The browser beside the agent's session: open, navigate, click, type. Not a headless browser. Same verbs as [Browser tools for pi](/guide/browser-tool). |
| Inbox | `notify_human`, `ask_human` | A note into your [Inbox](/guide/inbox-tools), or a question. Over MCP `ask_human` waits for your answer in the Inbox — hours if needed, reporting progress to the CLI so the call stays alive — and returns it to the agent. |
| Checklist | `checklist` | The agent's plan for the task; the current step shows on the terminal's card, as it does for pi ([Checklist](/guide/checklist)). The gate pi enforces before a change does not exist over MCP; the plan is the agent's discipline. |

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
| Global | Connectors → same cards → save globally | The CLI's user config, through the CLI's own command where it has one. |

The server has the same name in every place (`picode-computer`,
`picode-browser`), so a CLI that already has it from a file does not get it
twice from the launch.

Launch settings offer the group only for the CLIs that take servers on the
command line or environment: Claude Code, Codex and OpenCode. Grok, Hermes
Agent, Muse Code and Antigravity have no such mechanism (verified against
their binaries and docs, 2026-09-19): only their config files, so use the
Connectors pane at workspace or machine scope. Omp is a pi fork and can
take pi's own packages with `-e` per launch; that path is not built.

## By hand

Any MCP client can run the server; the CLI still needs to be inside a
PiCode terminal to have an identity.

```bash
claude mcp add picode-computer -- picode mcp computer
claude mcp add picode-browser -- picode mcp browser
```

`picode mcp computer browser inbox checklist` serves every family from one
process; the cards add one each.

## Without MCP: the `picode inbox` command

Not every agent gets MCP servers. A CLI you run once in a while, a script,
a sandbox whose config you cannot touch — none of them will read a
`.mcp.json`. For those, the same two moves the Inbox tools give pi are one
plain command away. The agent works in its terminal, reaches you through
the [Inbox](/guide/inbox-tools), and never needs a config file.

`picode inbox notify` files a non-blocking FYI — finished work, a state
change, anything you must know but need not answer:

```bash
picode inbox notify --title "Migration finished" --body "All 42 rows moved; report at docs/migration.md" --reason "the agent you left migrating"
```

`picode inbox ask` files a blocking question. With `--wait` the command
stays open, polls every 2 s, and prints your answer on stdout when you
give it — hours if needed, so a CLI harness's own tool timeout is the only
clock that matters:

```bash
picode inbox ask --question "postgres or sqlite for this service?" \
  --context "one service, two developers, no ops team" --wait
# waiting for your answer in the Inbox (item in-…)…        ← stderr
# The human answered: sqlite is fine                       ← stdout
```

Answer in the Inbox app — the item shows the question, the source (the
agent's name in a PiCode terminal, otherwise an honest `user@host`), and a
reply box. The reply is recorded on the item, which is exactly what the
waiting command picks up. Without `--wait`, the command prints the item id
and returns; `--timeout 30m` caps the wait when you want it capped.

Discovery is the same as every PiCode client: `--url`, else `PICODE_URL`,
else `<data dir>/server.json`, re-read on every poll so a restarted daemon
is found on its new port.

## What it will not do

- Give a CLI anything a pi agent could not get: the grant, the tier and the
  audit are the daemon's, and they are the same.
- Work from another machine: the server talks to the daemon on this one.
- Replace pi's packages: pi keeps its native tools and the capture in the
  chat.

## Where it goes next

Launch injection for a CLI the day its vendor adds a flag or environment
variable for it; the Omp `-e` path if there is demand.
