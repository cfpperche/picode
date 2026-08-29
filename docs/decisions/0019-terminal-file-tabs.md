# ADR-0019: Open terminal paths as editor tabs

- **Status**: accepted
- **Date**: 2026-08-29

## Context

VS Code's integrated terminal turns a path into an editor tab (Ctrl/Cmd+click).
PiCode is the host for the same Pi TUI and for first-class shells (ADR-0017).
The chat FilePane (ADR-0015) already opens text in the agent cwd, beside the
chat. That pane is not on the terminal tab. Clicking a path in xterm did
nothing.

We are not building an IDE: no explorer, no LSP, no MIME zoo. Text files
only in V1. http(s) opens in the browser.

## Decision

A path under that terminal's cwd, activated with Ctrl/Cmd+click (OSC-8 or
detected), opens a **first-class tab** on the same strip as agents and
terminals (`#/file/t/<id>/<path>` or `#/file/a/<id>/<path>` for the Pi TUI
dock). The same path focuses the existing tab. Paths outside the cwd are
not links. The chat FilePane is unchanged.

## Consequences

- **Easier**: the terminal is a door into the same files the chat already
  edits.
- **Harder**: mixed tabs; file API on terminals (`GET/PUT /api/terminals/{id}/text`).
- **If wrong**: hide file tabs; Ctrl+click becomes a no-op again.

## Alternatives considered

| Alternative | Why not |
|---|---|
| Split/overlay on the terminal | Competes with tabs; user asked for a tab |
| Reuse the chat FilePane | Leaves the terminal |
| Preview PDF/3D/video | Not V1; browser-native later, 3D never |
| Click without modifier | Steals selection |
