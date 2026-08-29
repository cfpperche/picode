# ADR-0017: First-class terminals (sidebar + main tabs)

- **Status**: accepted (supersedes ADR-0016's editor-tab / agent-scoped UI)
- **Date**: 2026-08-28

## Context

ADR-0016 put a project shell on the **agent** file pane (composer chip,
editor tab). The owner rejected that placement: a terminal is not part of
an agent, at least not now. They want terminals listed like agents, each
a tmux session, opened on the **same tab strip** as agents.

tmux + `/ws/term` from ADR-0016 stay. The Pi TUI dock (interactive `pi`)
stays the agent hatch (ADR-0002 / 0006).

## Decision

A terminal is a first-class PiCode record (`terminals` in SQLite): id,
name, cwd. Runtime is a tmux session `picode-sh-<id>` running `$SHELL`.
The sidebar has a Terminals icon next to Agents and Pins. **+** creates
one. Clicking a row opens it on the main tab strip (`#/term/<id>`).
Closing the tab detaches; removing the row kills tmux and the record.

No composer chip. No file-pane terminal tab. Not tied to an agent.

## Consequences

- **Easier**: a shell without picking an agent; survives like agents.
- **Harder**: mixed tabs (agent + terminal); hash `#/term/…` beside `#/agent/…`.
- **If wrong**: hide the sidebar icon; leftover tmux sessions stay attachable
  from a real terminal.

## Alternatives considered

| Alternative | Why not |
|---|---|
| Keep ADR-0016 editor tab | Owner: not now; found it in the composer and did not want it there |
| Tie cwd to the open agent | Same coupling; `cd` in the shell is enough |
| Bottom dock | Chat is the hero; tabs already exist for agents |
