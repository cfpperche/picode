# ADR-0021: Adopt a Pi session by copying it

- **Status**: accepted
- **Date**: 2026-08-29

## Context

Users already have Pi TUI sessions on disk (and sometimes live in tmux).
They want that history as a PiCode sidebar agent. Prompts can continue a
JSONL on an *existing* agent. Adopt creates a **new** agent. ADR-0006
forbids two live `pi` processes on one JSONL.

## Decision

Adopt **copies** the JSONL and points the new agent at the copy
(`--session`). The last `model_change` / `thinking_level_change` in that
file become the agent's provider, model, and thinking. The original file
and any live TUI are untouched. If the session cwd matches a workspace,
the agent is created there; otherwise it is a free agent with that cwd.
PiCode never kills the user's terminal.

## Consequences

- **Easier**: no stolen panes; TUI and PiCode can both exist.
- **Harder**: history forks at adopt time; later TUI turns do not appear
  on the copy.
- **If wrong**: hide **From a Pi session**; copies stay as ordinary files.

## Alternatives considered

| Alternative | Why not |
|---|---|
| Steal the live tmux session | User closes/removes the terminal themselves |
| Resume the original path | Two writers if the TUI is still open |
| Always create a workspace | Unknown folders are free agents (3C) |
