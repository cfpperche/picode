# Diff panel for the pi TUI (ADR-0077)

> Part of [PiCode's architecture](../architecture.md) (ADR-0105: one file per subsystem). Edit here; the index only links.

**An extension, not core.** Opt-in pi package at `packages/pi-diff/`
(MIT). `/diff` shows a right-hand overlay in the pi TUI: every file
changed against `HEAD` (tracked and untracked) with `+/-` counts, then
the numbered hunks of the file the agent touched last. In pi's fullscreen
TUI mode the panel is a layout column (an `HStack` around pi's layout
root: chat left, panel right, full height, fixed while the transcript
scrolls); in regular mode it is a full-height overlay drawn straight
through `tui.showOverlay(…, { nonCapturing: true })`. Both are owned by a
zero-height `ctx.ui.setWidget` slot, re-set on every refresh — never through
`ctx.ui.custom()`, whose `ui_prompt_*` lifecycle would read as "waiting
for user" to the guest-TUI sensors (ADR-0056). The package runs git
itself and never calls the daemon; PiCode has no code path for it. In
`--mode rpc` only the footer total survives; the Files and Changes pane
(ADR-0074) is the diff view there.
