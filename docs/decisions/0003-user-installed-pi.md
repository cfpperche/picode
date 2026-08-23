# ADR-0003: Depend on user-installed Pi (no vendoring)

- **Status**: accepted
- **Date**: 2026-08-23

## Context

PiCode drives the `pi` executable (`pi` interactive and `pi --mode rpc`).
Options:

1. **Depend on the user's installed `pi`** (requirement: `pi` on `PATH`).
2. **Vendor a pinned `pi` version** inside the PiCode distribution.

Pi ships fast (model catalogs, skills, extensions, protocol updates all
move); PiCode is a manager/orchestrator, not a fork. The pi-acp adapter
(project by deepstereo) validated the "user-installed pi" model in
production for Zed.

## Decision

PiCode requires a user-installed `pi` (v0.80.4+ recommended) on `PATH`,
checks for it at startup, and offers a one-click setup path (runs the
official npm installer command, shows the terminal output inline). Version
mismatch warnings are advisory, not blocking.

## Consequences

- **Easier**: Pi updates (models, providers, features) arrive free; no
  fork maintenance; smaller PiCode binary; user's own skills/extensions
  work by default — their environment, untouched.
- **Harder**: startup dependency detection and helpful failure UX; protocol
  drift between old installed `pi` and new PiCode (mitigate: feature-detect
  via `pi --version` and RPC handshake, degrade gracefully).
- **If wrong**: vendoring can be added as a fallback path later without
  architectural damage.

## Alternatives considered

- **Vendored pi**: rejected — forks drift, updates become our problem,
  user customizations (their `~/.pi/`) diverge from the vendored copy.
