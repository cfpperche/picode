# ADR-0024: Terminal settings — global defaults, per-terminal overrides, user presets

- **Status**: accepted
- **Date**: 2026-08-30

## Context

A terminal is a surface; what runs inside it is not PiCode's. Pi's TUI and
Claude Code's TUI want different things from the terminal underneath them, and
PiCode has no way to say so — the tmux options are constants in the code.

That is not theoretical. `internal/term/bridge.go` forced `mouse on` on every
session, which is what let the wheel scroll a TUI that does not take the mouse
itself. Removing it (2026-08-30, to get native text selection back) broke
scrolling in Pi's TUI while leaving Claude Code's untouched, because Claude
Code enables mouse tracking itself and Pi does not. One constant cannot serve
both, and the owner should not need an agent to change a constant.

**Two layers, and they do not belong in the same place.**

| Layer | Lives in | Examples | Real scope |
|---|---|---|---|
| Appearance (xterm.js) | `localStorage`, today's `#/preferences/terminal` | font, size, cursor, theme, padding | **per browser** — a laptop and a phone *should* differ |
| Behaviour (tmux) | nothing — constants in `bridge.go` / `terminals.go` | `mouse`, `history-limit` | **per session** — shared by every device that attaches |

A tmux option stored per browser would make one terminal behave differently on
two devices while the option itself is single. Behaviour must be stored
server-side.

**Not every flag applies the same way**, and a panel that hides this lies:

| Flag | When it takes effect |
|---|---|
| `mouse` | immediately, on the live session |
| `history-limit` | new panes only; existing scrollback keeps its size |
| `escape-time` | server-wide, not per session |
| `extended-keys-format` | server-wide (already flagged in `docs/handoff.md` as a landmine for multi-runtime TUIs) |

**Benchmark.** [Windows Terminal](https://learn.microsoft.com/en-us/windows/terminal/customize-settings/profile-general)
is the established shape: a `profiles.defaults` block plus profiles that
declare only what they change, inheriting the rest. We take that and add
presets, which it does not have.

## Decision

1. **Two homes.** Appearance stays in `localStorage`, per browser. Behaviour
   moves to the store: one global row, plus a row per terminal holding **only
   the fields that differ**.
2. **Live inheritance.** Changing a global default changes every terminal that
   has not overridden that field. Each field is therefore tri-state —
   *inherit / on / off* — and the panel shows the inherited value, so what a
   change will do is visible before making it.
3. **Presets are stamps that remember where they came from.** The user creates,
   edits and deletes them; PiCode ships none. Applying one writes its values
   into that terminal's overrides and records the preset's name. Editing a
   preset changes nobody silently; **Reapply** propagates on request. Deleting
   one removes a label and breaks nothing.
4. **Presets carry behaviour only.** A preset is meant to travel between
   devices; appearance is deliberately per device, so putting font in a preset
   would fight its own purpose.
5. **Only per-session flags are offered per terminal.** Server-wide options, if
   they are ever exposed, appear in the global panel and are labelled as such.
   Every field states when it takes effect.
6. **`mouse` returns to on by default**, with off available per terminal. The
   default that broke Pi's wheel becomes the opt-out rather than the given.

Entry points, both in the Terminals list: a gear beside **+ New terminal**
opens the global panel; a gear on a terminal's row opens that terminal's.

## Consequences

- **Easier**: a parity gap becomes a setting instead of a commit. The owner
  finds that TUI X needs a flag, flips it globally, and every terminal that
  has not overridden it follows.
- **Easier**: two TUIs with opposite needs can coexist — "TUI Pi" and
  "TUI Claude Code" as presets, applied per terminal.
- **Harder**: three states per field is more to render than two, and the panel
  must show provenance (inherited, overridden, from a preset) or it becomes
  guesswork.
- **Harder**: settings now live in two places by design. The panel has to make
  that legible — "this browser" versus "this session" — or it will read as an
  inconsistency rather than a decision.
- **Cost accepted**: applying a preset copies values. A preset improved later
  does not reach terminals already stamped until Reapply is pressed. That is
  the price of delete being safe.
- **If wrong**: the store rows are additive and the constants still exist as
  defaults. Removing the panel leaves the current behaviour.

## Weighed and not taken

Recorded as choices, not closed doors.

| Option | Why it lost, today |
|---|---|
| Presets as a live third layer (global → preset → terminal) | consistent with rule 2, but deleting a preset then damages every terminal bound to it, and the owner asked for delete. Provenance also becomes four-way per field |
| Copy global values at creation instead of live inheritance | fixing a parity gap would mean visiting every existing terminal one at a time — the opposite of the point |
| PiCode shipping preset definitions | we do not know what the TUIs of the future need; the owner does, when they hit it |
| Presets carrying appearance too | they are for sharing across devices, where appearance should differ |
| Exposing `allow-passthrough` or `status` | correctness and surface ownership (ADR-0017), not taste. Turning either off only breaks something |

## Open questions

- ~~Whether changing `mouse` on a session that already has an attached xterm.js
  reaches the browser.~~ **Measured 2026-08-30: it does — a PATCH is enough.**
  With a client attached and the option flipped from outside, the same wheel
  bytes that did nothing before the flip put the pane in copy-mode after it,
  with no reattach. The chain, end to end:

  | Step | Measured |
  |---|---|
  | `mouse on` | tmux sends `?1006h ?1000h ?1002h` to the outer terminal |
  | xterm.js | sees tracking on, sends SGR natively; `termWheel.js` stands down |
  | tmux | the app is not tracking, so tmux takes the wheel → scrolls |
  | drag-release | `OSC 52 ; ; <base64>` reaches the client (`set-clipboard external`) |
  | `termClipboard.js` | selection `''` is in `HONOURED`, so the copy lands |

  The cost is the known one: while tracking is on, a drag belongs to the
  application, so native browser selection needs Shift. That is the trade the
  per-terminal override exists to let a user make.
- Which flags beyond `mouse` earn a place. `history-limit` and `escape-time`
  are plausible; none has been asked for. The list should grow from real parity
  gaps rather than from what tmux happens to offer.
