# Architecture

> Status: v0.1 — evolves with the project. Split into `docs/architecture/` on 2026-09-09 (ADR-0105). Last reviewed: 2026-09-07 (catalog CLI Activity defaults on; OpenCode activity plugin; OpenCode CLI catalog; Hermes Agent CLI catalog; ADRs 0071–0074; mobile IME accessory in 0044).
> Changing anything described here requires updating this file (see [AGENTS.md](/AGENTS.md)).

## The one-paragraph version

PiCode is a **single Go binary that serves a browser UI** and manages **real
`pi` processes** on the machine where it runs. Each agent can use one of two
exclusive channels: a tmux-backed PTY that renders the genuine Pi TUI inside a
browser terminal, or an RPC bridge (`pi --mode rpc`, JSONL over stdio) that
feeds the rich UI with structured events. An Inbox reply may borrow the RPC
channel for one correlated turn while a holder preserves the same tmux pane;
there is still only one session writer. A broker routes messages between
agents through a Go MCP server (`picode-communication`), so agents talk to
each other using the MCP tool protocol.

Agent CLIs (ADR-0069) is a separate terminal manager for installed Pi, Claude
Code, Codex, Grok, Hermes Agent, OpenCode, Muse Code and Antigravity
commands. It reuses terminal records, tmux, invocation wrappers and the event
feed. Muse Code and Antigravity are full catalog rows (sessions, editable
launch, activity where the vendor surface carries identity). Omp (oh-my-pi)
is the newest entry: terminal, on-disk sessions and resume, without
activity, launch settings or lifecycle jobs. These are not Agent records: structured chat,
JSON-RPC, packages, orchestration and session ownership remain Pi-only until
a future decision supplies those contracts for another CLI.

`picode install` (ADR-0018) enables a systemd **user** unit so it starts with
this Linux session (WSL included). Its `KillMode=process` leaves tmux-owned
terminals alive across daemon restarts; transient RPC children are separately
parent-bound and pane holders restore the TUI. `picode deploy` / `make deploy`
copies a repo build and restarts that unit — but first asks the daemon
`GET /api/deploy/readiness` and refuses while any agent or terminal other
than the calling pane is mid-turn (ADR-0086; the caller is always working
by the act of asking, so it never blocks itself; `--force` overrides).
`main` ships when the owner runs `make deploy` (ADR-0105), never from a
branch session.
The supervised daemon does not
re-exec when the binary on disk changes — that same-PID `Exec` used to
swallow systemd's SIGTERM and sit in `stop-sigterm` until `TimeoutStopSec=30`.
SIGTERM now cancels the binary watcher, drains HTTP for at most 8s, and
returns so the process exits below the stop timeout. Foreground `go build`
still hot-reloads. `picode update` checks GitHub for a newer release.

`picode provision` (ADR-0020) converges a machine on all of that at once:
`[boot] systemd=true` in `/etc/wsl.conf`, lingering so the unit starts
without a login, a valid certificate, the unit itself, `/api/health` as
proof, and — for a server (ADR-0050) — `pi` on PATH, Tailscale up, and
whether other machines can reach the daemon. `--dry-run` is the doctor.
`picode install --env KEY=VALUE` keeps the service environment in
`~/.config/systemd/user/picode.service.d/env.conf`, which deploys and
updates leave alone; `picode update` verifies the release's `SHA256SUMS`
before it restarts on the new binary. Every step is check → fix → verify, and a step that finds nothing to
do changes nothing at all — `/etc/wsl.conf` is merged line by line after a
backup, so a file that already satisfies the check comes back byte for byte
identical. `--dry-run` reports the plan; `--json` feeds it to a caller. Root
work (`wsl.conf`, linger) and user work (unit, cert) are separate scopes, so
the two can be applied by different runs. On WSL that caller is PiCode
Desktop, the Windows tool that owns the logon task.

`picode-desktop` (`cmd/picode-desktop`, `internal/desktop`) is that tool: a
single `.exe` cross-compiled from WSL with `CGO_ENABLED=0` (`make desktop`),
headless, one command at a time. `doctor` reports, `install` applies and
registers the logon task. It drives the distro through
**two** `picode provision --json` calls — `-u root` for `wsl.conf` and
lingering, then `-u <owner>` for the unit, certificate and data dir, because
installing those as root would put PiCode in `/root`. The merged view keeps
whichever pass resolved each step. Windows-side it owns only what cannot live
in the distro: trusting the mkcert CA (already elevated at install, so no UAC
dance) and an interactive user-logon task with limited privileges. The shell
(`picode-shell.exe`, Tauri) is the resident since ADR-0142: it holds the VM
open with a supervised `sleep infinity` child, learns the address from
`server.json`, and polls `/api/health` over HTTP rather than spawning
`wsl.exe` on a timer. `wsl.exe` answers in UTF-16LE **without a BOM**, so its
output is decoded by inspecting the bytes.

`picode-desktop disk` reports the one number a Windows user cannot get
anywhere else, and `picode disk` (Linux, `internal/hostfs`) is its other half.
The distro half measures with one `df` and **two** `du` calls — two because
`du` skips a path it has already visited, so a parent and its child asked in
the same command report the parent with the child missing — and names every
cache it knows with the command that gives it back, labelled `safe` (costs
time), `redownload` (comes back from the network) or `data` (the person's own,
shown and never offered as a command). The report also names what it could not
measure, root-owned paths such as docker's storage, instead of folding the
remainder into a category. The Windows half reads the VHDX path from WSL's own
registry (`HKCU\…\Lxss`, `VhdFileName` included), the volume's free space and
the file's length from one PowerShell call answered as JSON (no locale decides
what `.` means), and the sparse flag from `fsutil sparse queryrange`, whose
parse keeps only the hex numbers because every word around them is translated
on a non-English Windows. The difference between what the file holds and what
the distro uses is space Windows keeps for data the distro has already freed:
invisible in Explorer, and unavailable to WSL while the file is not sparse —
which is the state a permanently running distro stays in, since both the
compaction and the sparse conversion need the distro stopped. The shell's tray
icon shows that as one line refreshed every five minutes and warns in words
below 20 GB free; with no balloon API, the tooltip is the alert surface. Both
disk commands only read.

The one action is `picode-desktop disk-compact`, and the shell's tray menu
carries it as **Give back ≈N GB…**. It refuses to run blind: first the server's readiness
interlock (the same `GET /api/deploy/readiness` `picode deploy` asks), then an
explicit confirmation that names the cost — stopping the distro ends every
agent, terminal and tmux session in it. The flow stops the distro, converts
the file to sparse when the WSL build allows (`wsl --manage --set-sparse`; no
elevation) or compacts with Optimize-VHD from an elevated terminal otherwise,
restarts the distro, **re-arms the keepalive child the terminate killed**
(nothing else restarts it, and without it WSL idles out sixty seconds later),
and reports the before and after measured on the file. A failed conversion
still restarts the distro — that ordering is pinned by a test. `--dry-run`
prints the plan and stops nothing; `--force` overrides the interlock.

Desktop startup policy (ADR-0071) is explicit: no execution-time, battery,
idle or network gates; duplicate task starts are ignored; launch failures have
three retries one minute apart. `internal/desktop/task.ps1` is embedded and called
through Windows PowerShell's Task Scheduler COM API with literal, UTF-16
encoded arguments. Registration applies the full definition in one write and
verifies the result. No new runtime dependency is added to Go.

`startup-check` inspects the Windows task without starting WSL. Doctor shares
its configured/disabled/stopped/running/error distinctions and reports the
registered executable and last result without inventing an exit cause.
`startup-repair` backs up an existing owned definition and changes only its
policy, preserving action, principal, triggers and enabled/disabled choice.
It does not start/stop a process or touch the distro, certificates or agents;
only access denied may request UAC. Missing or unexpected task definitions are
refused. A normal shell Quit exits zero. The scheduler is not a general crash
supervisor: an already-started process that later exits nonzero may remain
stopped. Even a successful launch retry's one-minute gap does not guarantee
WSL/session survival. Upgrade the executables before repairing an old
installation.

On a machine without WSL, `install` first walks a stage machine derived from
what it observes — never from saved progress, so an interrupted run resumes
and a finished one is a no-op: install WSL (`--no-distribution`), restart,
install the distro (`--no-launch`), create the account, then provision. Both
flags exist to dodge the interactive account setup that plain `wsl --install`
opens. Exit code **3010** means the step succeeded and Windows wants a
restart, not that it failed; a `RunOnce` key resumes setup at the next logon
and deletes itself. The Linux account is named after the Windows one and its
password is left **locked** — provisioning reaches root through `wsl -u root`
and never needs sudo, so setting a password or granting passwordless sudo
would be a security decision the installer has no standing to make.

CI follows that runtime boundary. Linux and macOS execute the complete daemon
suite. Windows compiles every package and test with race instrumentation, then
executes the native `picode-desktop`/browser-host boundary; the daemon still
runs inside WSL, so pretending to run its POSIX path, permission, tmux and Pi
session scenarios directly on Windows would test an unsupported topology.
Live WSL tests additionally require a registered distro: `wsl.exe` alone is
not evidence that one exists on a hosted runner.

The hosted workflow classifies the complete Git diff before allocating its
expensive runners. Unknown paths fail closed to the full matrix. Frontend build
and package tests run once on Ubuntu. The complete Vite output crosses the
artifact boundary for an embedded-build check; dependencies stay on the
frontend runner.
The three-OS Go matrix never installs Node. Public docs build in parallel and
check committed screenshot/OpenAPI/llms.txt parity *before* their generators
can rewrite those files. That job also verifies tutorial composition, still,
render and shipped-MP4 hashes without launching capture or rendering. Each
screenshot and tutorial still has a named surface profile: the hash combines
shared shell/style/fixture inputs, selected API/store producers, and the local
imports reachable from that screen. Test files and unrelated handlers stay
outside the profile. Screenshot freshness remains blocking; strict tutorial
freshness is a manual audit that names only the tutorial surfaces that drifted.
A stable final gate accepts only successful or intentionally skipped jobs; a
newer run on the same ref cancels its predecessor.

| Changed paths | Frontend + artifact | Public docs | Three-OS Go | Embedded build |
|---|---:|---:|---:|---:|
| Unknown, empty, product/toolchain, or mixed | Run | Run | Run | Run |
| `docs-site/`, `docs-videos/`, Vale styles, or docs tooling only | Skip | Run | Skip | Skip |
| `docs/` and root Markdown only | Skip | Skip | Skip | Skip |

`scripts/ci-scope.test.mjs` is the executable decision table. The lightweight
classifier and final gate still run in every row, so path routing cannot turn a
change into a workflow with no status.

Every binary ships in one GitHub release, tag-triggered
(`.github/workflows/release.yml`), with the version stamped through
`-X internal/version.Version` — `Version` is a var for exactly that. Asset
names are a contract: `internal/install.assetName()` looks for
`picode-<goos>-<goarch>`, and `picode-desktop` for
`picode-desktop-windows-amd64.exe` plus the shell's
`picode-shell-windows-amd64.exe`; a test reads the workflow to keep them
from drifting. `picode update` and `picode-desktop update` both use
`install.LatestReleaseFor`, both verify `SHA256SUMS` before swapping, and the
desktop one refreshes the tool and the shell together or neither. Elevation is
decided at **runtime** (`ShellExecuteW` with `runas`), never by a manifest:
the tool does its everyday reads unelevated, and a `requireAdministrator`
manifest would prompt for those too.

The release workflow also validates matching `CHANGELOG.md` and
`whats-new.json` entries and publishes the changelog section as the GitHub
release body. A stamped binary exposes
`release: true` on `GET /api/version`; the two shells use that signal to offer
the bundled `web/shared/data/whats-new.json` highlights through their owned
`WhatsNew` surface. A browser acknowledges a semver in
`localStorage` (`picode-whats-new-seen`), so the release opens once per
browser, once the shell has booted and never over an active Inbox, create,
share, reconnect, or other modal flow. A **fresh install** is included: the
product-state gate ADR-0063 shipped with was removed by its 2026-09-11
amendment. Source builds stay manual-only. The surface is bounded to the newest three releases and nine
highlights and links to the complete release body for detail (ADR-0063).

## Application routes

Two independent React apps, the launcher, the route table and the native CLI pages live in [architecture/routes.md](architecture/routes.md) and the files listed under **Where the rest lives**.

| Page or capability | File |
|---|---|
| [Application routes](architecture/routes.md) | `docs/architecture/routes.md` |
| [File preview: HTML (ADR-0136)](architecture/file-preview.md) | `docs/architecture/file-preview.md` |
| [Native CLI settings (ADR-0101)](architecture/cli-settings.md) | `docs/architecture/cli-settings.md` |
| [Native CLI packages (ADR-0102)](architecture/cli-packages.md) | `docs/architecture/cli-packages.md` |
| [Native CLI providers (ADR-0103)](architecture/cli-providers.md) | `docs/architecture/cli-providers.md` |
| [CLI terminal launch settings (ADR-0069)](architecture/cli-terminal-launch.md) | `docs/architecture/cli-terminal-launch.md` |
| [Cross-CLI session handoff (ADR-0088)](architecture/cli-session-handoff.md) | `docs/architecture/cli-session-handoff.md` |
| [llama.cpp manager (ADR-0080)](architecture/llama-manager.md) | `docs/architecture/llama-manager.md` |
| [Data & persistence (ADR-0005)](architecture/data-persistence.md) | `docs/architecture/data-persistence.md` |
| [AgentManager](architecture/agent-manager.md) | `docs/architecture/agent-manager.md` |
| [TerminalBridge](architecture/terminal-bridge.md) | `docs/architecture/terminal-bridge.md` |
| [RPCBridge](architecture/rpc-bridge.md) | `docs/architecture/rpc-bridge.md` |
| [Broker](architecture/broker.md) | `docs/architecture/broker.md` |
| [SessionReader](architecture/session-reader.md) | `docs/architecture/session-reader.md` |
| [Compaction policy (ADR-0061)](architecture/compaction-policy.md) | `docs/architecture/compaction-policy.md` |
| [Diff panel for the pi TUI (ADR-0077)](architecture/tui-diff-panel.md) | `docs/architecture/tui-diff-panel.md` |
| [Model roles (ADR-0028, ADR-0033)](architecture/model-roles.md) | `docs/architecture/model-roles.md` |
| [MCP (Model Context Protocol) support](architecture/mcp.md) | `docs/architecture/mcp.md` |
| [Integrations (ADR-0075)](architecture/integrations.md) | `docs/architecture/integrations.md` |
| [Pins](architecture/pins.md) | `docs/architecture/pins.md` |
| [Snippets (ADR-0130)](architecture/snippets.md) | `docs/architecture/snippets.md` |
| [Canvas (ADR-0108, ADR-0118)](architecture/canvas.md) | `docs/architecture/canvas.md` |
| [Automations (ADR-0045)](architecture/automations.md) | `docs/architecture/automations.md` |
| [Change feed (ADR-0048)](architecture/change-feed.md) | `docs/architecture/change-feed.md` |
| [Notices — the in-app announcement layer](architecture/notices.md) | `docs/architecture/notices.md` |
| [Security model (ADR-0007)](architecture/security-model.md) | `docs/architecture/security-model.md` |
| [Shared box: the gateway (ADR-0051)](architecture/gateway.md) | `docs/architecture/gateway.md` |
| [Chrome extension (ADR-0043)](architecture/chrome-extension.md) | `docs/architecture/chrome-extension.md` |
| [Docker App and sysadmin tools (ADR-0065)](architecture/docker-app.md) | `docs/architecture/docker-app.md` |
| [Docker maintenance and health (ADRs 0067/0068)](architecture/docker-maintenance.md) | `docs/architecture/docker-maintenance.md` |
| [tmux app (ADR-0133)](architecture/tmux-app.md) | `docs/architecture/tmux-app.md` |

## Component diagram

```
┌────────────────────────────────────────────────────────────┐
│ Browser                                                     │
│  ├─ Rich UI (React + Vite + Tailwind — ADR-0008)           │
│  │   sidebar tabs (agents·workspaces·terminals·pins)      │
│  │   tasks · diffs · sessions tree · auth                 │
│  └─ xterm.js terminals (the real Pi TUI, 1:1)              │
└───────────────┬────────────────────────────────────────────┘
                │ HTTP /api/*  +  WebSocket /ws/*
┌───────────────▼────────────────────────────────────────────┐
│ picode (single Go binary, UI embedded — `-tags embedui`)   │
│                                                            │
│  AgentManager ─── spawn/stop/restart ──► pi processes      │
│  TerminalBridge ─ tmux sessions ───────► pi (interactive)  │
│  RPCBridge ────── JSONL stdio ─────────► pi --mode rpc     │
│  TaskQueue ────── steer / follow_up ───► RPCBridge         │
│  Broker ───────── inbox routing ───────► PiCode extension  │
│  SessionReader ── parse ~/.pi/agent/sessions/*.jsonl       │
│  ChangeFeed ───── events table → SSE /api/events (replay)  │
│  Automations ──── cron tick / webhook ──► new session      │
└────────────────────────────────────────────────────────────┘
```

## Why dual channel per agent

Revised by ADR-0006: **one live pi process per agent, two exclusive run
modes** — interactive (tmux TUI) or managed (rpc + panel). The original
simultaneous design risked concurrent writers on pi's session files.

| Channel | Carries | Fails to | Cost |
|---|---|---|---|
| tmux + PTY (`/ws/term/:agent`) | Full Pi TUI: interactive login, any command, escape hatch | provide structured data | ~zero — tmux does the work |
| RPC (`pi --mode rpc`) | Structured events: streaming, tool calls, state | interactive dialogs (needs UI mapping) | protocol discipline |

Both channels read/write **the same session files** (`~/.pi/agent/sessions/`),
so the user can alternate between the rich view and the terminal without
losing state. They never write one file concurrently — ADR-0060 keeps it that
way by never starting a second writer: an Inbox reply is submitted inside the
TUI process by the receiver extension, or typed into the pane over tmux. A
per-agent mutation guard serializes replies with pane/session changes. The
persistent run mode stays `interactive`. Closing the browser tab does not kill
anything: the interactive agent lives inside tmux; an ordinary RPC agent is
detached from the browser entirely.

Stopping a managed agent kills the **whole spawned process tree**. The rpc
client starts `pi` in its own process group and `Close` SIGKILLs the group
before closing stdin/stdout, so a child that inherits the stdout pipe (the
real `pi` behind the intercept wrapper) can never outlive its parent and
wedge `Runtime.Stop` — the 2026-09-05 pi-diff hang, where `Stop agent` and
`Open terminal` stopped answering entirely. Defense in depth: the pi
intercept wrapper `exec`s the real pi for managed `--mode rpc|json` runs,
so managed runs have no shell middleman to orphan in the first place.

## Key subsystems

One file per subsystem under `docs/architecture/` (ADR-0105). A handler, store or RPC change reads the one file it touches, not this index.

| Subsystem | File |
|---|---|

## Explicit non-goals

- Re-implementing the Pi TUI (we embed it instead).
- Sandboxing agents (that's Pi's trust model, not ours).
- Hiding Pi: every GUI action maps to something inspectable in the terminal.

## Direct session communication (ADR-0104)

See [architecture/direct-session-communication.md](architecture/direct-session-communication.md).
