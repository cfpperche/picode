<p align="center">
  <img src="www/public/favicon.svg" width="80" alt="PiCode logo">
</p>

<h1 align="center">PiCode</h1>

<p align="center">
  <strong>Run and supervise real Pi coding agents from your browser.</strong><br>
  Create agents, connect them to projects, watch their work and step in when
  they need you — from a desktop or a phone.
</p>

<p align="center">
  <a href="https://github.com/cfpperche/picode/actions/workflows/ci.yml"><img alt="CI status" src="https://github.com/cfpperche/picode/actions/workflows/ci.yml/badge.svg"></a>
  <a href="https://cfpperche.github.io/picode/"><img alt="Documentation" src="https://img.shields.io/badge/docs-online-777ff0"></a>
  <a href="LICENSE"><img alt="PolyForm Noncommercial license" src="https://img.shields.io/badge/license-PolyForm--Noncommercial-blue"></a>
  <img alt="Pre-alpha status" src="https://img.shields.io/badge/status-pre--alpha-orange">
</p>

<p align="center">
  <a href="https://cfpperche.github.io/picode/guide/getting-started">Get started</a>
  ·
  <a href="https://cfpperche.github.io/picode/">Documentation</a>
  ·
  <a href="docs/architecture.md">Architecture</a>
</p>

> [!IMPORTANT]
> PiCode is pre-alpha. The core workflow is usable, but interfaces, storage
> migrations and installation paths may still change between releases.

![PiCode desktop showing a multi-workspace agent fleet and its activity dashboard](www/img/app-fleet.png)

## Why PiCode

Pi is a focused coding harness with a great terminal experience. A terminal is
also a hard place to supervise several long-running agents across projects,
especially when the person directing them does not live in a shell.

PiCode adds the orchestration layer: one browser workspace for creating,
configuring and steering a fleet of Pi agents. It does not replace Pi. Every
agent is a real `pi` process, and the genuine Pi TUI remains one tab away.

**The browser is a door, not a cage.**

## What you can do today

| Capability | What it gives you |
|---|---|
| Agent fleet | Create free agents or attach several agents to a workspace, each with its own model, provider and working directory. |
| Chat and terminal | Use a structured conversation view or switch to the real Pi TUI running in a tmux-backed browser terminal. |
| Project tools | Browse and edit files, inspect diffs and Git history, manage sessions, and open persistent project terminals. |
| Human inbox | Collect questions, approvals and finished work in one place; reply without hunting for the right agent tab. |
| Automations | Start fresh agent runs on a schedule or webhook, with templates, limits and run history. |
| Desktop and phone | Supervise the same fleet through the desktop UI or the installable mobile PWA, with pairing and push notifications. |
| Pi ecosystem | Manage providers, packages, MCP adapters and settings while keeping Pi's native files authoritative. |

PiCode is designed for solo developers running a few agents, terminal-averse
users who still want direct control, and teams hosting agents on a machine they
manage.

## What stays yours

PiCode is deliberately a thin layer over the tools and files you already own.

| Concern | Source of truth |
|---|---|
| Agent runtime | Your installed `pi` binary |
| Conversations | Pi session JSONL files under `~/.pi/agent/sessions/` |
| Credentials and configuration | Pi's own auth, settings, package and extension files |
| Interactive processes | tmux sessions that survive browser and daemon restarts |
| Orchestration | PiCode's local SQLite database under `~/.picode/` |

If PiCode is not running, your Pi sessions and configuration are still regular
Pi data. See the [architecture](docs/architecture.md) for the full trust and
persistence model.

## Quick start

The supported service install runs on Linux or WSL with systemd user services.
Building from source requires:

- [Go 1.26+](https://go.dev)
- [Node.js 22](https://nodejs.org) (the version used by CI)
- [Pi](https://www.npmjs.com/package/@earendil-works/pi-coding-agent)
- tmux 3.5+

```bash
npm install -g @earendil-works/pi-coding-agent
git clone https://github.com/cfpperche/picode.git
cd picode
make build
./bin/picode install
```

Open `https://localhost:8445`, then:

1. Add a workspace for a project folder.
2. Create an agent inside it and choose its provider, model and run mode.
3. Select **Run agent** and follow the work in Chat or Terminal.

Closing the browser does not stop the agent. PiCode generates a self-signed
certificate on first run; `make cert` installs a locally trusted mkcert
certificate when you want the browser warning to disappear.

For release binaries and remote or shared servers, continue with the
[installation guides](https://cfpperche.github.io/picode/guide/remote-server).
The docs also cover [phone pairing](https://cfpperche.github.io/picode/guide/mobile)
and the [Chrome extension](https://cfpperche.github.io/picode/guide/browser-extension).

### Useful commands

| Command | Purpose |
|---|---|
| `make dev` | Run from the repository; run `make web` once on a fresh clone. |
| `make deploy` | Rebuild this checkout and restart the installed service. |
| `picode pair` | Print a one-time link for another browser or phone. |
| `picode update` | Download and verify a newer GitHub release. |
| `picode uninstall` | Remove the service; add `--purge` to delete `~/.picode`. |

## How it works

```text
Desktop browser / mobile PWA
            │ HTTPS + WebSocket + server-sent events
            ▼
┌─────────────────────────────────────────────────────┐
│ picode · one Go server                              │
│                                                     │
│ lifecycle · inbox · automations · files · event feed│
│             │                         │             │
│       tmux-backed PTY          JSONL RPC bridge     │
└─────────────┼─────────────────────────┼─────────────┘
              ▼                         ▼
        real Pi TUI                pi --mode rpc
              └──────── Pi sessions and config ───────┘
```

An agent uses one live channel at a time, so the TUI and RPC view never write
the same session concurrently. The browser can disconnect without owning the
agent process. Details and trade-offs live in
[docs/architecture.md](docs/architecture.md) and the
[architecture decision records](docs/decisions/).

## Project documentation

- [Public documentation](https://cfpperche.github.io/picode/) — tutorials,
  guides, command reference and HTTP API
- [Current project state](docs/handoff.md) — what is shipped, in flight and
  still owed
- [Architecture](docs/architecture.md) — components, protocols and security
- [Decision records](docs/decisions/) — the reasoning behind architectural
  choices
- [Release process](docs/release-process.md) — maintainer checklist for public
  releases (cadence proposal pending)
- [Engineering benchmarks](docs/benchmarks.md) — the product and quality bars
  used in review

## Contributing

Humans and Pi agents work under the same repository contract. Read
[CONTRIBUTING.md](CONTRIBUTING.md) and [AGENTS.md](AGENTS.md) before making a
change. Pull requests keep code, tests, documentation, changelog and handoff in
sync.

## License

PiCode is source-available. Personal and noncommercial use is covered by the
[PolyForm Noncommercial License](LICENSE). Enterprise and other commercial use
requires a paid license; see [LICENSE-COMMERCIAL.md](LICENSE-COMMERCIAL.md) and
[LICENSING.md](LICENSING.md).
