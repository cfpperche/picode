# ADR-0029: Composer lists live extension commands

- **Status**: accepted
- **Date**: 2026-08-31

## Context

PiCode's composer `/` menu is a fixed list plus skills and templates from
disk. Extension commands (`pi.registerCommand`) only ran if the user typed
the name in full and sent it. Packages such as `pi-roles` are opt-in; a
Roles page in the product would show chrome to people who never installed
the package (ADR-0028).

The live agent already answers `get_commands` over RPC. The composer
already sends prompts to that agent.

## Decision

`GET /api/agents/{id}/slash` includes `commands` from the **running
managed** agent (`get_commands`, `source: "extension"` only). The
composer picker shows them. Picking one sends `/name` as a prompt on
the control channel — not into the terminal.

Stopped agents, interactive-only agents, and RPC failures omit the
list (skills and templates still appear). Names that collide with a
PiCode `/` command are dropped so the product action wins.

## Consequences

Easier: `/roles` appears only when `pi-roles` is loaded in that process.
No package-specific UI in the app.

Harder: the menu is empty of those items until the agent is running.
Interactive (TUI) mode has no RPC `get_commands` from this path.

If wrong: we can later also read installed package manifests when the
agent is stopped. That is a follow-up, not this ADR.

## Alternatives considered

- **Hardcode `/roles` in `slash.js`.** Rejected: dead chrome without the
  package; every new package would need a PiCode change.
- **A Roles settings page.** Rejected: ADR-0028 — config stays in the
  package dialogs.
- **Type the slash into the TUI dock.** Rejected: slash-parity rule
  (composer never proxies `/` into tmux).
