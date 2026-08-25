# ADR-0012: Preferences is PiCode; Settings is pi

- **Status**: proposed
- **Date**: 2026-08-25

## Context

`#/settings` today is PiCode-the-product (theme, server port). The TUI
`/settings` writes **global** `~/.pi/agent/settings.json` (29 knobs).
Composer `/settings` must open PiCode UI, not type into the dock
([slash-parity](../design/slash-parity.md)). Users also need project
(`.pi/settings.json`) and per-agent (SQLite) layers.

Architecture.md currently says Settings is product-only. That collides
with slash parity.

## Decision

1. `#/preferences` owns PiCode chrome (theme, port).
2. `#/settings` owns **pi** configuration, read and write, for the
   selected agent: global always; workspace overlay if the agent is
   bound to a folder; agent row always.
3. Composer `/settings` navigates to `#/settings`. It never sends
   `/settings` to tmux.
4. Auth, packages, and MCP stay on their own routes (ADR-0009, 0010).

Plan: [design/pi-settings.md](../design/pi-settings.md).

## Consequences

- **Easier**: one verb `/settings` means “configure pi” in both TUI and
  GUI; product chrome has a name that is not pi's.
- **Harder**: we write JSON pi owns (no settings CLI); must
  read-modify-write and respect project trust.
- **If wrong**: dumping TUI padding/cursor into the GUI cages users in
  a clone of the 29-row menu. v1 is a short allowlist.

## Alternatives considered

- **Keep `#/settings` as product, put pi under `#/pi`**: rejected —
  composer `/settings` would not match the hash, and users already look
  for Settings.
- **One page with product + pi**: rejected — architecture split
  (product vs pi) stays; only the hash names change.
- **Proxy `/settings` to the TUI**: rejected — slash-parity rule.
