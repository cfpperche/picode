# Handoff — living project state

> **This file is the heartbeat of PiCode.** Any session (human or agent)
> that changes state MUST update this file before ending.
> Ritual defined in `/skill:handoff-update`; contract in [AGENTS.md](/AGENTS.md).
>
> Format rules: newest entries at the top of "Recent activity"; keep it
> short — this is a handoff, not a diary. Archive old blocks to
> `docs/handoff-archive.md` when the file exceeds ~150 lines.

## Current state (read this first)

**Phase: M0 — Bootstrap (complete). M1 — Terminal grid (next).**

What exists right now:
- Repo public at `cfpperche/picode`, MIT, CI (gofmt/vet/test/build) green.
- Documentation system live: philosophy, architecture, benchmarks,
  ADRs 0001–0003, this handoff.
- Pi harness: `AGENTS.md` contract, skills `quality-gate`, `uiux-review`,
  `handoff-update` in `.pi/skills/`.
- Go skeleton compiling: `picode` binary serves embedded UI at `/`,
  `/api/health`, `/api/version`. Placeholder page with live health check.
- No frontend framework chosen yet (candidate decision: React vs Svelte —
  needs ADR-0004 when M1 starts).

## In flight

- Nothing. Tree is clean, CI green, no half-done work.

## Next up (M1 — Terminal grid, in order)

1. **tmux session manager** (`internal/tmux/`): create/kill/list sessions;
   require tmux ≥ 3.5 + `extended-keys-format csi-u`; write recommended
   config into the user's tmux setup flow (or document clearly).
2. **PTY WebSocket bridge** (`internal/term/`): attach xterm.js ↔ tmux
   session via WebSocket; survive disconnect; resize handling.
3. **Workspace registry** (`internal/workspace/`): data dir, list/add/remove
   workspaces (folders), spawn `pi` inside each as tmux session.
4. **Minimal UI**: terminal grid — one tab per agent/workspace, xterm.js,
   "New workspace" flow (empty state that teaches, per benchmarks).
5. ADR-0004: frontend framework decision (bring criteria to the table).

## Known debts / open questions

- Persistence format for the workspace registry (JSON file vs SQLite) —
  decide with ADR-0005 when workspace registry lands.
- Security token auth: current skeleton binds localhost only; when we add
  `--listen`, token auth is mandatory (contract in architecture.md).
- Frontend has zero build tooling — `internal/web/public/` is hand-written.
  Introduce tooling (vite) only when M1 UI work starts, keep `go:embed`
  pipeline intact.
- Branch protection + CODEOWNERS on GitHub — needs manual setup or rulesets
  API (owner action, listed here so it's not forgotten).

## Recent activity

- **2026-08-23** — Bootstrap (M0): repo created, docs system, ADRs 0001–0003,
  Pi harness (AGENTS.md + 3 skills), Go skeleton with embedded UI,
  health/version endpoints + tests, CI, Makefile. Released as 0.1.0.
