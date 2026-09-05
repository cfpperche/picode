# ADR-0077: Side diff panel for the pi TUI — an extension, drawn as a non-capturing overlay

- **Status**: accepted
- **Date**: 2026-09-05

## Context

Claude Code shows a side panel in its terminal: every file changed with
`+/-` counts, then the hunks of one file, toggled with `/diff` and refreshed
as the agent edits. The owner supervises pi sessions in PiCode's terminal
tabs and wants the same view for pi. The question was where it belongs.

Three placements were on the table:

- **Write into the PTY from PiCode.** Pi owns the whole screen; bytes
  written by the daemon are either editor input or garbage in the next
  redraw. ADR-0060 already demoted tmux typing to a fallback for this
  reason: the TUI never stops being the writer.
- **A PiCode web feature.** The Files/Changes pane (ADR-0074) is that, for
  PiCode's own chat. It cannot appear inside a terminal tab, where the
  owner spends most of the session.
- **A pi extension.** Pi 0.85's extension contract has everything the
  panel needs: `registerCommand`, `registerShortcut`, the
  `tool_execution_*` events, and overlays positioned by anchor, sized in
  percentages, with a `visible(termWidth)` gate. PiCode already injects
  extensions into the pi it spawns (ADR-0056, ADR-0060) and ships four
  MIT packages under `packages/` (ADR-0028, 0037, 0055, 0061).

One trap in the extension route: `ctx.ui.custom()` — the documented way
to show an overlay — is wrapped in `ui_prompt_start`/`ui_prompt_end`
(`withUIPrompt("custom", …)` in the extension runner). A panel that stays
open for the whole session would report "waiting for user" to every host
that reads those events, PiCode's guest-TUI sensors included (ADR-0056).

## Decision

Ship `packages/pi-diff`, an MIT pi package in the mold of `pi-checklist`:
pure logic in `src/logic.ts` (git output parsing, focus rules, layout)
covered by `node:test`, and I/O glue in `extensions/diff.ts`. `/diff`
toggles a right-hand overlay shown directly with `tui.showOverlay(…,
{ nonCapturing: true })`, obtained through a zero-height widget factory
(`ctx.ui.setWidget`) so pi's own widget lifecycle owns the overlay: set
shows, clear disposes. The editor keeps focus throughout. The list is
`git diff --numstat HEAD` plus untracked files; focus follows the last
`edit`/`write`/`multiedit` and can be overridden (`/diff <path>`, `next`,
`prev`). Refresh runs after mutating tools and at `turn_end`/`agent_end`,
debounced. The footer carries the total at all times. Scrolling is
`alt+n`/`alt+u` and `alt+pageDown`/`alt+pageUp`, because `alt+↑/↓` and
`alt+j/k` are pi's own. PiCode core does not change: the package talks to
git, never to the daemon.

## Consequences

- Easier: the owner gets the panel in every pi terminal tab, and in a
  plain `pi` outside PiCode, with one `pi install`. Nothing to uninstall
  from PiCode; removing the package restores stock pi.
- Harder: pi's overlay API is marked experimental. A change there breaks
  the panel, not PiCode. The panel is invisible in `--mode rpc` (PiCode
  chat), where the Files/Changes pane already exists; only the footer
  total survives there.
- Accepted: the comparison base is always `HEAD`, not the session's
  first commit. Changes made outside the session appear at the next
  refresh, not immediately.
- If we are wrong about `showOverlay` staying public on the `TUI` type,
  the fallback is `ctx.ui.custom({ overlay: true, nonCapturing })` plus
  filtering our own `ui_prompt_*` events in PiCode's sensors.

## Alternatives considered

- **Bytes into the PTY**: refused, see Context; it corrupts the TUI.
- **`ctx.ui.custom()` with `{ overlay: true }`**: refused for the
  lifecycle side effect above; it also parks the command handler's
  promise for the whole session.
- **`ctx.ui.setWidget` above the editor instead of an overlay**: refused;
  a widget shrinks the chat with every line and cannot sit beside it.
- **Extending PiCode's Files/Changes pane to terminal tabs**: refused;
  a web pane next to an xterm does not follow the agent's focus or the
  TUI's theme, and it cannot exist for a `pi` run outside PiCode.

## Amendment (2026-09-05): a layout column in fullscreen mode

The first deploy drew the panel as an overlay in every mode. In pi's
default *regular* TUI mode the terminal owns the scrollback, so the overlay
scrolled away with the chat and sat wherever the TUI's working area
happened to start — the owner asked for a panel that fills the screen and
stays put. Pi's *fullscreen* mode (`--tui-mode fullscreen`, `tuiMode` in
settings) owns the viewport and lays the screen out from a layout root
(`ViewportTUI.setLayoutRoot`). The extension now wraps that root in an
`HStack` — chat and dock on the left, the panel on the right, weights
55/45, hidden under 100 columns — so the panel is a full-height column
that never moves and the editor wraps at the narrower width instead of
being covered. Regular mode keeps the overlay, now full height, and the
first `/diff` points at fullscreen mode once. Because a TUI mode switch
replaces pi's TUI instance, the widget slot is re-set on every refresh.
Cost accepted: reading `layoutRoot` is a private field of the alt-screen
TUI; if pi renames it the panel falls back to the overlay path. Whether
PiCode should spawn its terminal TUIs with `--tui-mode fullscreen` is a
separate owner decision, not taken here.

