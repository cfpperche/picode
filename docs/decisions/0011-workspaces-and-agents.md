# ADR-0011: Workspaces contain agents; free agents exist

- **Status**: accepted
- **Date**: 2026-08-25

## Context

PiCode's sidebar labeled the list **Agents** but each row was a **folder**
with exactly one default agent (v1 invariant). Users want:

1. A group of **workspaces** PiCode knows (project folders).
2. **Many agents per workspace**, each with its own model/config — not a
   rigid profile for the folder.
3. **Free agents** above workspaces: unbound pi sessions (no project
   preconfig). These are ordinary pi processes with cwd = home, not
   `pi -e` packages.

The store already had `agents.workspace_id` and a comment that M3 would
add siblings. This ADR is that step.

## Decision

1. **Workspace** = a directory (`path` unique). Owns zero or more agents.
2. **Agent** = one live `pi` (tmux/managed), own provider/model/thinking/
   session JSONL. Config is per agent, never locked to the folder.
3. **Free agents** live in a reserved workspace `ws_free` (hidden from the
   workspace list). Cwd is the user home directory. They do not use
   `pi install -l`.
4. Adding a workspace still creates a first agent (so a new folder is
   immediately usable). Extra agents are `POST /api/workspaces/{id}/agents`.
5. The selected entity in the UI is the **agent id**, not the workspace id.

## Consequences

- **Easier**: several models on one repo; scratch pi without a project.
- **Harder**: sidebar density; session APIs stay workspace-scoped for cwd.
- **If wrong**: collapsing back to 1:1 is a data migration (keep the first
  agent per workspace).

## Alternatives considered

- Nullable `workspace_id`: extra SQLite rebuild under FK=ON. Sentinel
  workspace is simpler and still one `pi` process per agent.
- Treating free agents as `pi -e`: that is one process, dies on Stop, and
  is about packages — not the same as a named unbound agent.
