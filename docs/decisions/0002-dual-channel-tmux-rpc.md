# ADR-0002: Dual-channel agent control — tmux PTY + pi RPC

- **Status**: accepted
- **Date**: 2026-08-23

## Context

PiCode must offer both a rich GUI (status, tasks, diffs) and the *genuine*
Pi TUI experience. Two integration surfaces exist:

- `pi` interactive in a PTY — the full TUI, but no structured data out.
- `pi --mode rpc` — JSONL events (streaming, tool calls, state, steer/follow_up
  semantics), but no interactive TUI and dialogs arrive as RPC requests.

Hiding the TUI would violate "the browser is a door, not a cage" (users must
always reach the real Pi). GUI-only-on-RPC would die on interactive flows
like `/login` OAuth.

## Decision

Every agent is controlled through **two channels over the same underlying
session state**:

1. **Terminal channel**: interactive `pi` runs inside a **tmux session**;
   the browser attaches via xterm.js over WebSocket. Closing the tab
   detaches; the agent keeps running.
2. **Control channel**: `pi --mode rpc` subprocess managed by the Go server,
   feeding structured events to the rich UI and accepting task queue
   commands (`steer`, `follow_up`, `abort`, `get_state`).

Both read/write Pi's session files; tmux also gives us scrollback, resize
and detach semantics for free.

## Consequences

- **Easier**: interactive login flows work untouched (do them in the
  embedded terminal); task semantics come from RPC; closing the browser
  never kills an agent; the TUI is 1:1 Pi, zero reimplementation.
- **Harder**: two processes per agent (resource cost, must document);
  concurrent writers to the same session file must be understood (Pi's
  session manager owns the format; we treat files as read-only in the UI
  and never write them outside pi itself); RPC protocol discipline
  (strict `\n` framing — no U+2028/U+2029 splitting; extension dialogs
  must be answered by the UI or auto-approval policy).
- **If wrong**: the channels are independent; dropping one doesn't break
  the other.

## Alternatives considered

- **RPC only** with re-implemented UI: rejected — re-implements Pi's TUI
  forever (maintenance treadmill) and breaks interactive flows.
- **PTY only** with screen-scraping: rejected — fragile, no structured
  tasks/diffs.
- **tmux send-keys for control**: rejected — brittle; RPC is a real API.
