# ADR-0164: The tmux substrate is 3.7+, and reports name the server's version

- **Status**: accepted
- **Date**: 2026-09-20
- **Boundary**: process — which process owns the terminal canvas, and which
  tmux binary PiCode's reports describe.

## Context

ADR-0002 made tmux the PTY substrate; every agent and terminal lives in a
session on a per-instance socket (ADR-0139). Two things changed on
2026-09-20, measured on the owner's machine:

1. **The host runs tmux 3.7c** (`/usr/local/bin/tmux`, built from source)
   while Ubuntu 26.04's package stays at 3.6a in `/usr/bin/tmux`. tmux's
   `PROTOCOL_VERSION` is 8 in 3.6, 3.7c and 3.8-rc, so a 3.7c client talks
   to a 3.6 server: **replacing the binary does not restart the server**, and
   the running server keeps being the emulator until it exits. Both versions
   are live at once, and `tmux -V` (the client) says 3.7c while the server
   answering `#{version}` still says 3.6.
2. **The overlay surface PiCode does not own is now real.** 3.6 has
   `display-popup` (modal, ephemeral, rectangular). 3.7 added floating panes
   — "sit above the layout (tiled panes) like popups but unlike popups are
   not modal and behave like panes (so the same escape sequence support)",
   mouse move/resize only. 3.8 (rc, 2026-09-09) adds modal panes
   (`new-pane -O`, `-K` sends every key including the prefix, `-C` closes on
   an outside click) and `-x/-y/-X/-Y` positioning. An editor or a git graph
   can ride those panes without PiCode embedding a terminal emulator.

Re-measured before deciding, on isolated sockets (`-f /dev/null`, a private
`TMUX_TMPDIR` and `-L`, never the owner's server): the option defaults for
`mouse`, `status`, `allow-passthrough`, `extended-keys` and
`extended-keys-format` are identical in 3.6 and 3.7c; both answer a pane's
Kitty query (`CSI ? u`) with DA1 alone, so the forced
`extended-keys-format xterm` in `EnsureExtendedKeys` stays correct; DA1
parity (`CSI ? 1;2;4c`) holds once the new binary is built `--enable-sixel`,
which the Ubuntu package is. PiCode never calls `new-pane`, `split-window` or
`new-window` (its verbs are session- and pane-read only), so 3.7's change of
`new-pane` into a *floating* pane cannot bite it.

## Decision

PiCode's terminal substrate requires **tmux ≥ 3.7 for floating or modal
overlays**, and any overlay surface it gains later rides tmux's own panes
(`display-popup` in the 3.6 line, floating panes from 3.7, modal panes from
3.8) rather than an embedded terminal emulator. `/api/system` and the tmux
app report the **version in use** — the running server's `#{version}`,
falling back to the installed binary only when no server answers — never
`tmux -V` while an older server owns the panes.

## Consequences

Easier: overlays (an editor, a graph, a launcher) are panes with real TTYs,
mouse support, and the same escape-sequence fidelity as any other pane — no
emulator, no capability-query proxy, no cgo. Reports stop lying by
construction: the version the UI shows is the one that explains behavior.

Harder: the host requirement is now 3.7, and the version is a property of the
*running server*, which changes only when the server restarts (a reboot, or
the last session ending). Everything this repository measured on 3.6 stays
true until then, so the follow-up re-measure is a line in
`docs/handoff/open/terminal.md`, not an assumption here.

If we are wrong: a machine on 3.6 (fresh Ubuntu, a container, a CI image)
still runs PiCode — nothing gates on the version — and would silently show no
floating overlays if that surface ever ships. The mitigation is the honest
report plus this ADR; a hard gate is not worth breaking `#/clis`-era machines
for a surface that does not exist yet.

## Alternatives considered

- **Own the canvas: embed libghostty-vt** (or `charmbracelet/x/vt`), the way
  `coder/boo` does. Gives cell-level overlays over a child TUI's own grid, at
  the cost of answering DA1 / kitty-keyboard / OSC 52 / sixel ourselves and
  arbitrating mouse modes, plus cgo + Zig in CI for libghostty. That is a TUI
  product decision — a new surface with its own owner call — not a
  requirement to record here; the option stays open in this ADR's shadow.
- **zellij plugins**: pane-level overlays only, authored in Rust/WASM, and a
  second multiplexer beside tmux (ADR-0002).
- **Neovim as the canvas**, floating windows over terminal buffers: real and
  proven (`claudecode.nvim`, `agentic.nvim`), but the canvas then belongs to
  the user's editor, not to PiCode.
- **Stay on 3.6 + `display-popup`**: works today, but popups are modal and
  ephemeral, so an editor or a graph cannot stay on screen beside the agent.
- **Keep reporting `tmux -V`**: simplest, and misleading in exactly the window
  an upgrade creates — a client that answers 3.7c while 3.6 interprets the
  panes.
