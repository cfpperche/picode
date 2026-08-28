# ADR-0015: Browser file editor for the agent cwd

- **Status**: accepted
- **Date**: 2026-08-27

## Context

PiCode is named as a coding ADE. Until now diffs in chat were view-only
(`web/src/lib/diff.js`: "Accept/reject is out of scope"). The Cursor bar
said we are not building a code editor (no file tree, no tabs, no LSP).
The owner reversed the *file* part: a product called PiCode that cannot
open or edit a file in the browser is a mismatch. Terminal-averse users
have no hatch into the agent's files except the TUI.

This is not a request for an IDE. Worktrees (parallel isolated agents)
stay later. LSP, explorer-as-home, and replacing the Pi TUI stay refused.

`GET /api/agents/{id}/file` today serves **images** for the composer, not
text. List/browse already exist and stay under the agent cwd
(`relUnderCwd`).

## Decision

PiCode opens and edits **text files in the agent working directory** in
the browser. Agent diffs gain per-hunk Keep / Undo that write the same
files. The widget is CodeMirror 6 (popular primitive; not Monaco, not a
homemade textarea). Pins keep TipTap.

The file lives in the workspace shell next to the chat (`#/agent/<id>`).
It is not a new product route. Paths cannot escape the cwd.

## Consequences

- **Easier**: review and a small fix without the terminal; hunks are real.
- **Harder**: write API, dirty/conflict, binary/large-file empty states.
- **If wrong**: the TUI still edits the same files; we can hide the pane.
- Philosophy "door, not cage" still holds: the editor is another door
  into pi's cwd, not a cage around it.

## Alternatives considered

| Alternative | Why not |
|---|---|
| Stay view-only | Owner rejected; the name PiCode does not match |
| Monaco / VS Code web | IDE-shaped, huge; we refused IDE chrome |
| TipTap for code | Prose editor; pins already use it |
| External `$VISUAL` | TUI hatch; does not help the browser user |
| Replace tmux with Herdr | Different product; see the Herdr study. We keep ADR-0002 |
