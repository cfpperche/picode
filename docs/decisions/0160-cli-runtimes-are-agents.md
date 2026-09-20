# ADR-0160: CLI runtimes are agents

- **Status**: accepted (owner session, 2026-09-19)
- **Date**: 2026-09-19
- **Boundary**: persistence — an `agents` row may name any launchable catalog
  CLI, not only Pi; security model — a workspace instance of a CLI is
  `grant.KindAgent` (the agent id), not `term:<id>`. Process: `Runtime.Start`
  remains Pi managed-mode only (`pi --mode rpc`).
- **Amends**: ADR-0006 (one live *process* per agent; for Pi that is `pi`,
  for others the catalog command; managed RPC stays Pi-only until 0091),
  ADR-0011 (an agent is not only a Pi instance), ADR-0069 (catalog, launch,
  sensors and pins stay the interactive implementation; a *workspace
  instance* is an agent). ADR-0091 is untouched: no ACP, JSON-RPC or
  composer for guests.
- **Supersedes**: ADR-0159's rule that a guest is never an `agents` row.
  Unbound CLI terminals (`#/clis/new`, project shells) stay terminals.

## Context

Pi already has one identity and two modes (ADR-0006): `interactive` (`pi` in
tmux) and `managed` (`pi --mode rpc`). The owner asked to put Claude Code,
Codex and the other catalog CLIs in that same class: they are agents; for
now they only have interactive mode (the TUI); managed mode per CLI is
later. ADR-0159 instead invented a second identity (`managed_clis` +
`term:<id>`) so `Runtime.Start` would never see a guest. That leak showed
up as two names in New (**Agent** vs **Agent CLI**). The fence we needed
was "do not spawn `pi --mode rpc` unless the agent's CLI is Pi", not "this
is not an agent".

CLI management already built (catalog, install, `terminal_launches`,
sensors, pins, session resume, `picode mcp`, Inbox `needs-you`, prompt
door) is the interactive stack. It is not a second class of person. `#/clis`
remains the **runtime** surface (install, machine defaults, on-disk
sessions). A row in a workspace is an agent.

## Decision

An **agent** has a catalog `cli` (default `pi`). Pi keeps both run modes.
Every other launchable catalog CLI is interactive-only until that CLI gets
a managed adapter. `rpc.Runtime.Start` refuses a non-Pi agent: no process,
no queue drain. Unbound terminals are not agents. Shell terminals are not
agents. `managed_clis` is transitional: later slices migrate those rows
onto `agents` and drop the table. Grant keys for those instances become
the agent id; `term:` remains for terminals that are not agents.

## Consequences

Easier: one New → Agent, one fleet, Inbox and automations address agent
ids; Fatia 4's prompt door attaches to an agent whose interactive process
is already the CLI TUI. Harder: every `GetAgent` → `Runtime.Start` path
must honour `IsPi()` (the gate lives in `Runtime.Start` so a missed caller
cannot spawn `pi` on a Claude). Pi-only columns (provider, model,
packages, checklist, session JSONL) stay meaningful for `cli=pi` and are
ignored for guests until a later adapter. If we are wrong, the failure is
a Claude row that cannot start managed — not a second writer on a Pi
session.

## Decision table

| Conditions | Action |
|---|---|
| Create agent, `cli` empty or `pi` | `agents` row `cli=pi`. Interactive and managed as today. |
| Create agent, other launchable catalog CLI | `agents` row with that `cli`. Interactive only. |
| Unknown or detect-only CLI | refuse |
| `Runtime.Start` on non-Pi agent | error, no process |
| Unbound CLI terminal / shell | not an agent |
| Existing `managed_clis` row | still a principal until the migrate slice copies it to `agents` |

Coverage (this slice): `TestAddAgentCLIDefaultPi`,
`TestAddAgentWithCLI`, `TestStartRefusesNonPiAgent`,
`TestEveryMutationAppendsAnEvent` (`AddAgentWithCLI`).

## Alternatives considered

- **Keep `managed_clis` as the identity (0159).** Lost: two product
  classes for one gesture; every later surface special-cases `agentId` vs
  `termId`.
- **Guest rows in `agents` without gating `Runtime.Start`.** Lost: Restart
  would spawn `pi --mode rpc` on Claude.
- **Adopt ACP now (0091).** Lost: unconverged protocols; the owner uses
  the TUI.
