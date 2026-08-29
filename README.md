<p align="center">
  <strong>PiCode</strong>
</p>
<p align="center">
  A browser-based <strong>Agent Development Environment (ADE)</strong> for <a href="https://pi.dev">Pi</a> coding agents.<br/>
  Create, configure and orchestrate agents across multiple workspaces — <em>no terminal fear required</em>.
</p>
<p align="center">
  <img alt="CI" src="https://github.com/cfpperche/picode/actions/workflows/ci.yml/badge.svg">
  <img alt="License" src="https://img.shields.io/badge/license-PolyForm--Noncommercial-blue">
  <img alt="Status" src="https://img.shields.io/badge/status-pre--alpha-orange">
</p>

---

## Why PiCode

Pi is a brilliant minimal coding harness — for people who live in the terminal.
But agents are becoming team members: they are **created**, **configured**,
**given tasks**, and need to **work in parallel** across projects. Managing that
with raw terminal sessions does not scale, and it actively excludes people who
never made peace with a shell.

**PiCode's moat: Pi users control their agents from the moment of creation.**

From the first "New Agent" click to orchestrating a fleet across workspaces,
PiCode owns the agent lifecycle — while remaining a thin, honest layer over
the real `pi` running underneath. Nothing is hidden: you can always drop into
the genuine Pi TUI, one tab away.

## For whom

- **Pi power users** juggling multiple agents, models and workspaces.
- **Humans with terminal aversion** who still want superhuman coding help.
- **Teams** that want agents on a server, accessible from any browser.

## Design philosophy

PiCode inherits Pi's own values — **simplicity and modularity** — and adds a
third: **the browser is a door, not a cage**.

| Principle | Meaning |
|---|---|
| Simplicity | One Go binary. Zero install ceremony. `picode` → browser → working. |
| Modularity | Every capability is a Pi extension/skill/config — the same primitives Pi users already know. |
| Door, not cage | The full Pi TUI is embedded 1:1 (xterm.js over tmux). The GUI never becomes a bottleneck. |
| Agents first | PiCode is developed *by* Pi agents as much as *for* them. The repo is a Pi-native workspace. |

## How it works

```
Browser (xterm.js + rich UI)
   ↕ HTTP + WebSocket
picode — single Go binary (UI embedded via go:embed)
   ├─ Agent manager     spawn / stop / restart pi per workspace
   ├─ Terminal bridge   tmux-backed PTY → the real Pi TUI in your tab
   ├─ RPC bridge        pi --mode rpc → structured events, tasks, diffs
   ├─ Task queue        steer / follow_up per agent
   └─ Broker            inter-agent messaging (via a Pi extension)
```

See [docs/architecture.md](docs/architecture.md) for details and
[docs/decisions/](docs/decisions/) for the reasoning behind each choice.

## Quick start

Requires: [Go 1.22+](https://go.dev), [Pi](https://www.npmjs.com/package/@earendil-works/pi-coding-agent), tmux 3.5+.

```bash
git clone https://github.com/cfpperche/picode.git
cd picode
make build
./bin/picode install    # systemd --user; starts with this Linux session
```

`make deploy` rebuilds from this repo and restarts the service. `picode update` is for a normal install (GitHub release). `make dev` runs without installing. `picode uninstall` removes the service (`--purge` also deletes `~/.picode`).

First run: add a workspace (your project folder), click **Run** — a real Pi
agent starts in the workspace and streams into the conversation panel. Close
the tab; the agent keeps running.

**Green padlock (recommended, once):** `make cert` issues a certificate from
a local mkcert CA trusted on Linux and Windows (WSL) — the browser stops
warning. Without it, a self-signed cert is generated automatically. Phone
access works over your tailnet with the same certificate.

## Documentation

Docs are a **living system** — they evolve with the code, by contract (see
[AGENTS.md](AGENTS.md)). Docs style follows the benchmark set in
[docs/benchmarks.md](docs/benchmarks.md).

- [Architecture](docs/architecture.md) — components, data flow, protocols
- [Philosophy](docs/philosophy.md) — why PiCode is a door, not a cage
- [Benchmarks](docs/benchmarks.md) — engineering + UI/UX bars we hold ourselves to
- [Handoff](docs/handoff.md) — current state of the project (start here)
- [Decision records (ADRs)](docs/decisions/) — every architectural choice, documented

## Roadmap

- [x] **M0 — Bootstrap**: repo, docs system, Pi harness, Go skeleton (v0.1.0)
- [x] **M1 — Terminal grid**: multi-tab tmux sessions running real `pi`, per workspace
- [x] **M2 — Agent panel**: RPC bridge, live status, tasks (steer/follow-up), diffs *(core shipped: managed mode, delivery engine, panel; diff view + palette in progress)*
- [ ] **M3 — Lifecycle**: agent creation wizard, provider auth (`/login` flows), profiles
- [ ] **M4 — Fleet**: inter-agent broker, plugin/skill manager, session tree browser
- [ ] **Localization**: UI localization (PT-BR first) — the project was born
      from Portuguese-speaking users, and that audience matters

## Contributing

Humans and Pi agents contribute under the same contract — read
[CONTRIBUTING.md](CONTRIBUTING.md) and [AGENTS.md](AGENTS.md) first.
Every session that changes state must end with an updated
[docs/handoff.md](docs/handoff.md) and [CHANGELOG.md](CHANGELOG.md) entry.

---

*The project's vision was born from Portuguese-speaking users with terminal
aversion — UI localization is on the roadmap, but the repository itself
speaks English (see [AGENTS.md](AGENTS.md)). 🇧🇷→🌐*

## License

Source-available. **Personal / noncommercial:** [PolyForm Noncommercial 1.0.0](LICENSE).
**Enterprise / commercial:** paid license — [LICENSE-COMMERCIAL.md](LICENSE-COMMERCIAL.md).
See [LICENSING.md](LICENSING.md). © 2026 cfpperche
