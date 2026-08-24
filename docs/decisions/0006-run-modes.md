# ADR-0006: Agent run modes — one live pi process per agent

- **Status**: accepted (amends ADR-0002's simultaneous dual-channel model)
- **Date**: 2026-08-24

## Context

ADR-0002 designed each agent with two simultaneous channels: an
interactive `pi` in tmux and a `pi --mode rpc` subprocess, "over the same
underlying session state". Building the M2 delivery engine surfaced the
flaw: **pi session JSONL files are append-only trees owned by the running
process**. Two live processes on one session file means two concurrent
appenders (and two compactors) — corruption risk, not a UI problem.

## Decision

An agent runs in exactly **one mode at a time**, with at most **one live
pi process**:

| Mode | Process | Panel (structured) | TUI (interactive) |
|---|---|---|---|
| `interactive` | `pi` in tmux (M1 path) | — | full |
| `managed` | `pi --mode rpc` (M2) | full (events, tasks, steering) | — |

Mode switching is explicit and safe: starting one mode stops the other
first. Sessions remain per-process as pi creates them (keyed by cwd);
both modes see the workspace's session history via the session reader
(read-only), and `/resume` semantics stay available inside the TUI.

## Consequences

- **Easier**: no concurrent writers, ever; task delivery is unambiguous
  (managed mode only); status truth is one process.
- **Harder**: no live TUI *and* live panel for the same agent at the same
  instant (accepted — they're different workflows: driving vs. inspecting);
  the UI must communicate the mode switch clearly.
- Task queue applies to managed mode; in interactive mode, queued tasks
  stay `queued` (delivered when the agent is next run in managed mode).

## Alternatives considered

- **True simultaneous dual channel (original ADR-0002)**: rejected —
  concurrent session writers; pi's session manager owns the format.
- **RPC-only (drop tmux)**: rejected — loses the escape hatch for
  interactive flows (`/login`, ad-hoc work) that define "door, not cage".
