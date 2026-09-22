# ADR-0179: Agent CLIs are optional; tmux is the only runtime dependency

- **Status**: proposed (direction approved by the owner, 2026-09-22)
- **Date**: 2026-09-22
- **Boundary**: process — what `picode install`, `picode provision`, the
  System page and the Windows installer require before they call the machine
  ready; persistence — an `agents` row in the free workspace
  (`FreeWorkspaceID`) may name any launchable catalog CLI, as a workspace row
  already can (extends ADR-0160).
- **Supersedes**: the requirement in [ADR-0003](0003-user-installed-pi.md)
  ("PiCode requires a user-installed `pi` on PATH, checks for it at startup").
  Its other half — never vendor a CLI — stands.
- **Amends**: [ADR-0045](0045-automations.md) (the fire decision asks for Pi
  only when the run creates or wakes a Pi agent), [ADR-0050](0050-tailnet-server.md)
  (the provision doctor reports which agent CLIs are on PATH instead of
  blocking on `pi`), [ADR-0098](0098-windows-clean-install.md) (the desktop
  installer's runtime is tmux, git, curl, Node 22 and npm; CLIs are installed
  from Agent CLIs afterwards), [ADR-0160](0160-cli-runtimes-are-agents.md)
  (free agents too). The sentence "managed agents remain Pi" in
  [ADR-0069](0069-agent-cli-terminals.md) is history since 0160.
- **Plan**: [../plans/multi-cli-ade.md](../plans/multi-cli-ade.md)

## Context

PiCode started as a browser UI for one CLI, and ADR-0003 made that CLI a
startup dependency: `pi` on PATH, checked at boot, with an install hint when
absent. The product has since become a multi-CLI ADE — Agent CLIs manages
Pi, Claude Code, Codex, Grok, Hermes Agent, OpenCode, Muse Code, Antigravity
and Omp (ADR-0069), every launchable CLI is an `agents` row (ADR-0160), and
providers, settings, packages, connectors, memory, delivery and the
dashboard are declared per CLI (ADRs 0097, 0150, 0163, 0165, 0167, 0169,
0171, 0174). Measured on 2026-09-22: the server boots without `pi`,
`picode install` has no `pi` check, and the Providers roster reads Pi's
catalog only for Pi's own row.

Five places still treat `pi` the way tmux is treated, and each one breaks or
blocks something for a user who runs only Claude Code, Codex or Omp:

| Where | What happens without `pi` |
|---|---|
| `/api/system` (`internal/server/system.go`) | a fixed `pi` block beside tmux, mkcert and tailscale, plus the warning "pi is not installed — install it with npm …" in the same array as the tmux warning; the System page prints both |
| Automations (`internal/server/automations_run.go`) | `PiMissing` comes from `LookPath("pi")` for every automation, and the decision runs before the door delivery, so a scheduled message to a Claude Code agent fails with "no agent could start" |
| `picode provision` (`internal/provision/steps.go`) | `piStep` is a blocked step, `Converged()` is false, the command exits 1 on a healthy machine |
| Windows installer (`internal/desktop`, `cmd/picode-desktop`) | `pi` in `RuntimeTools` keeps the machine in the install-runtime stage, the post-install probe fails with "still missing after install", and the Node major is read from Pi's npm `engines` |
| Free agents (`POST /api/agents`, sidebar **New agent**) | no `cli` field: a free agent is always Pi, and the picker injects Pi as installed regardless of the catalog |

The front doors say the same thing in words: the docs landing page ("Agent
Development Environment for Pi"), Getting started ("Install Pi and tmux —
PiCode runs real Pi agents"), the README, the `--help` banner, the systemd
unit's `Description=`, `llms.txt` and the GitHub description. No page says
that PiCode works without Pi. `AGENTS.md` told every agent session that
"managed agents remain Pi" three days after ADR-0160 made that false, which
is how the bias kept reproducing.

## Decision

PiCode depends on tmux and on nothing else at runtime. Every agent CLI in
the catalog is optional, installed by the user (from Agent CLIs or with the
vendor's command) and never vendored; Pi is one of nine. Install, provision,
the System report and the desktop installer neither require nor warn about
any CLI: they report which ones are present. A feature asks for a specific
CLI only where that CLI is the process it runs, and says which CLI it needs.

| Surface | Needs Pi? | Why |
|---|---|---|
| Server boot, `picode install`, `picode provision`, `/api/system`, the Windows installer | no | this ADR |
| Agents of other CLIs (workspace or free), terminals, sessions, providers, native settings, guest packages, connectors, dashboard | no | ADR-0160, 0163, 0167, 0169 |
| Managed mode: `pi --mode rpc`, structured chat, composer, `/api/catalog`, Pi's Packages and Settings panes, `/trust` | yes | ADR-0091, ADR-0160 |
| Automation **start** (creates a fresh Pi agent) | yes | the run *is* a Pi process |
| Automation **message** | only when the target is `cli=pi` and not running | a guest agent is reached through its launch terminal (ADR-0089 door) |
| Free agent, `cli` empty or `pi` | yes, to run it | as today |
| Free agent, other launchable CLI | no | `AddAgentWithCLI(FreeWorkspaceID, …)` plus a free launch terminal |

## Consequences

Easier: a Claude-only or Omp-only user installs on Linux, WSL and Windows
and sees a System page that names what is present; automations aimed at
guest agents fire; the docs, the README and the operating contract describe
the product that ships; `agents.cli` means the same thing in a workspace and
in the free list. Harder: the System page and the automations banner have to
read the CLI catalog (`GET /api/clis`, `describeCLI`) instead of one boolean;
the desktop installer pins its own Node major (22, the CI version) instead
of asking Pi's package; `/api/system` changes shape (only the UI and one QA
fixture read `system.pi` today). If we are wrong — a subsystem quietly needed
Pi — the failure is a Pi-shaped error message on a machine without Pi, not a
broken install: the same class of failure users have today, now scoped.

The work lands in five branches, in order: this decision with the front
doors; System page and automations; provision and the installer; free agents;
the configuration docs, labels and jargon. Each branch carries its decision
table and tests; the installer's real acceptance is a run on the
`picode-test` VM, recorded in `docs/handoff/open/multi-cli-ade.md` until the
owner runs it.

## Alternatives considered

- **Keep Pi required and fix only the words.** Lost: a Claude-only user
  still cannot finish the Windows install, `provision` still exits 1, and
  scheduled messages to guest agents still fail. The words would then be
  false.
- **Make every CLI a dependency check.** Lost: nine warnings nobody asked
  for on a page that should name one hard requirement. Presence per CLI
  already lives in Agent CLIs.
- **Vendor the CLIs so nothing is ever missing.** Rejected in ADR-0003 and
  still rejected: forks drift, updates become ours, user customizations
  diverge.
- **Rename the free-agent button to "New Pi agent".** Honest and cheap, but
  it keeps two definitions of "agent" in one app, the leak ADR-0160 removed.
