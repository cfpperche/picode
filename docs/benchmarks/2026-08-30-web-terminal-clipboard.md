# Copying out of a browser terminal backed by tmux

- **Date:** 2026-08-30
- **Why:** In a PiCode terminal, copying does not reach the system clipboard.
  The owner asked whether there is a flag that takes tmux's *skin* off a TUI —
  not whether tmux could be removed. There is no single flag, but the skin is a
  short list of options, and one of them turned out to be obsolete.

## What was measured here first

| Fact | Value | How |
|---|---|---|
| Claude Code grabs the mouse itself | yes | `tmux display -p '#{?mouse_any_flag,...}'` on its pane |
| Pi's TUI grabs the mouse | no | same, on the `pi` pane |
| tmux version | 3.6 | `tmux -V` |
| `allow-passthrough` | **off** (global, no session override) | `tmux show -g allow-passthrough` |
| `set-clipboard` | `external` | `tmux show -g set-clipboard` |
| `Ms` capability reachable | yes — `terminal-features[0] xterm*:clipboard`, and the bridge attaches with `TERM=xterm-256color` | `tmux show -g terminal-features`, `internal/term/bridge.go:82` |
| xterm.js OSC 52 handler | none — only `addon-fit` is installed | `web/package.json` |

So the mouse complaint and the clipboard complaint are **two different
problems**, and only the second is ours:

1. An app that turns mouse tracking on (Claude Code) takes the drag before the
   terminal can select. Shift is the universal bypass, and it behaves the same
   in Windows Terminal or iTerm — nothing to do with tmux or PiCode. Anthropic
   ships `CLAUDE_CODE_DISABLE_MOUSE_CLICKS` for exactly this.
2. When something *does* copy — tmux's own copy-mode, or an app emitting
   OSC 52 — the text never reaches the browser. That one is PiCode's.

## Two clipboard paths, two different blockers

| Path | Who emits | What governs it | Blocked here by |
|---|---|---|---|
| tmux copy-mode copy | tmux, **raw** OSC 52 | `set-clipboard` + the `Ms` terminfo capability | xterm.js ignores OSC 52 |
| App inside the pane (Claude Code, vim + oscyank) | the app, OSC 52 **wrapped in a DCS envelope** (`\ePtmux;\e\e]52;…\a\e\\`) | `allow-passthrough` | `allow-passthrough off` drops it inside tmux, *and* xterm.js would ignore it |

The second row is the one that bites hardest and the one most guides get
wrong. `set-clipboard` and `terminal-features` govern raw OSC 52 only; the DCS
envelope is a separate code path that bypasses tmux's clipboard handling
entirely, and tmux 3.3+ drops it unless `allow-passthrough` is on.

## Who solved it, and how

| Project | What they do | Receipt |
|---|---|---|
| **VS Code** | Shipped OSC 52 in the integrated terminal (2024). It is the reference behaviour: an app in the terminal may *set* the system clipboard. Still silently ignored over Remote SSH and in Codespaces — the same "copy goes nowhere" failure PiCode has today | [microsoft/vscode#193508](https://github.com/microsoft/vscode/issues/193508), [vscode-remote-release#11475](https://github.com/microsoft/vscode-remote-release/issues/11475) |
| **ttyd** (already a PiCode bar — "terminal honesty") | xterm.js over websocket, same stack as ours. OSC 52 only works once a handler is registered; xterm.js does nothing by default | [ttyd](https://tsl0922.github.io/ttyd/), [xtermjs/xterm.js#3260](https://github.com/xtermjs/xterm.js/issues/3260) |
| **Coder** | Web terminal with the same gap, still open | [coder/coder#16577](https://github.com/coder/coder/issues/16577) |
| **tmux upstream** | Documents the model: `on` lets apps set tmux's clipboard *and* tmux set the outer one; `external` (default since 2.6) lets only tmux set the outer one; `Ms` must be reachable or nothing is emitted | [tmux wiki — Clipboard](https://github.com/tmux/tmux/wiki/Clipboard) |

The shape everyone converges on is the same: **the terminal emulator must
handle OSC 52**. Nobody solves this by removing the multiplexer.

## The mouse option was obsolete

The bridge forced `mouse on` with this reason (`internal/term/bridge.go`):

> *xterm.js#426: the outer tty must be in mouse mode or the wheel becomes
> Up/Down (#1310) and never reaches the TUI.*

That was true when it was written. `web/src/lib/termWheel.js` has since grown a
fallback that synthesises the SGR wheel bytes itself precisely when nothing is
tracking the mouse:

```js
const mouse = term.modes && term.modes.mouseTrackingMode;
if (mouse && mouse !== "none") return "skip";   // the app will get them
...
const bytes = sgrWheelBytes(next);              // otherwise we send them
```

**Owner tested it on the real machine: with `mouse off`, the wheel still
scrolls.** So the option's only remaining effect was putting tmux's copy-mode
over every drag — the skin. Both call sites (`bridge.go`,
`internal/server/terminals.go`) stop setting it.

An application that wants the mouse is unaffected: it enables tracking itself
and tmux forwards that through. Claude Code does exactly this, which is why
Shift is still needed *there* — see "what this does not fix".

## What was decided (2026-08-30, owner)

1. **Stop forcing `mouse on`.** Native browser selection returns for anything
   that does not grab the mouse itself.
2. **`allow-passthrough on`** per session, so OSC 52 can leave the pane.
3. **A write-only OSC 52 handler**, hand-written, no dependency.

`set-clipboard` stays `external` — the DCS path bypasses it, so widening it
buys nothing here.

### Why write-only, and why not the addon

`@xterm/addon-clipboard` implements the **read** form as well —
`if (pd === '?') { const text = this._provider.readText(pc); }`, backed by
`navigator.clipboard.readText()`. In a product whose terminals exist to run
third-party coding agents, that hands any process in the pane a way to read the
user's system clipboard. ~25 lines of our own support only the set form. This
is the dependency justification AGENTS.md #3 asks for.

### Weighed and not taken

Recorded as choices, not as closed doors — revisit any of them with the owner.

| Option | Why it lost, today |
|---|---|
| `@xterm/addon-clipboard` | brings OSC 52 read into a pane running someone else's agent |
| `set-clipboard on` | lets any app in the pane write tmux buffers; the DCS path does not need it |
| Removing tmux | ADR-0002/0017 rest on it — and it would not fix the mouse complaint anyway, since the app grabs the mouse in any terminal |
| Copy-on-select in the browser | a different feature (local selection), not the OSC 52 path. Worth its own decision if drag-copy should also hit the clipboard automatically |

## What this does not fix

- An app holding the mouse still needs Shift to select by drag. That is the
  app's choice and the terminal's convention, in every terminal.
- `navigator.clipboard.writeText` can be refused without a user gesture,
  browser depending. A copy-mode drag carries one (mouse release); a copy an
  agent performs on its own may not. The handler must fail quietly rather
  than throw, and this stays an open question until it is exercised in
  Firefox — see `docs/handoff.md`.
