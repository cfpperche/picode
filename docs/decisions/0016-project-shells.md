# ADR-0016: Project shells in tmux, as editor tabs

- **Status**: accepted
- **Date**: 2026-08-28

## Context

The GUI already embeds the **Pi TUI** (ADR-0002 / 0006): one tmux session
per interactive agent (`picode-<id>`), attached with xterm.js. That is
the escape hatch into `pi`, not a project shell.

The owner wants a real shell in the browser: each terminal is its own
tmux session, opened as a **tab in the file pane** (not a VS Code bottom
dock, not an agent tab). Track E already put files in that pane
(ADR-0015). LSP, explorer-as-home, and Monaco stay refused.

## Decision

A **project shell** is a PiCode-owned tmux session named `picode-sh-<agent>`
running `$SHELL` in the agent cwd. The browser attaches with the existing
`/ws/term` bridge. Closing the editor tab **detaches**; the session keeps
running. Kill is explicit (`DELETE`). This is not a pi process and does
not amend ADR-0006.

The shell opens as a tab in the file pane on `#/agent/<id>`. File tabs
and the shell share that pane. The Pi TUI dock stays the interactive
agent hatch.

## Consequences

- **Easier**: a terminal-averse user reaches a cwd shell without leaving
  the chat; `tmux attach -t picode-sh-…` still works if PiCode is gone.
- **Harder**: two tmux namespaces (`picode-` vs `picode-sh-`); deleting
  an agent must kill its shell; no tmux → one line + Open System.
- **If wrong**: hide the Terminal chip; the TUI dock and `!cmd` remain.

## Alternatives considered

| Alternative | Why not |
|---|---|
| Extra window on the agent's tmux session | That session *is* `pi`. Stop/kill the agent would take the shell. |
| Bottom dock like Cursor | Chat is the hero. The file pane is already the editor surface. |
| New PTY stack (Herdr, node-pty) | We already have `internal/tmux` + `internal/term`. |
| Auto-kill on tab close | Breaks detach; philosophy is door, not cage. |
