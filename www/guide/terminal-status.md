# Terminal status for coding CLIs

A coding CLI running inside a PiCode terminal can identify itself and
report its lifecycle, so the sidebar shows **which CLI is present**, when it
is **working**, and when it **needs you** (ADRs 0056 and 0062).

PiCode does not read the terminal's pixels, and it does **not** write
your `~/.claude`, `~/.codex`, `~/.grok`, or `~/.pi`. You type `claude`,
`codex`, `grok`, or `pi` as usual. Inside a PiCode terminal only, a
wrapper on that session's PATH launches the real binary with the flags,
extension, or overlay that CLI accepts.

Turn it on in **Preferences → Terminal status**. New terminals after
that pick up the intercept; terminals that already existed need to be
recreated.

## States

| Presence / state | Meaning | Where you see it |
|---|---|---|
| `Terminal open` | tmux is open, but no supported CLI was confirmed | quiet terminal row |
| `<CLI> · Open` | Claude Code, Codex, Grok, or Pi owns the pane; activity is not known yet | CLI mark + open label |
| `<CLI> · Working` | the CLI started a turn and has not finished | spinner and CLI mark |
| `Needs you` | the CLI is waiting on you (permission prompt, question) | accent chip |
| `<CLI> · Ready` | the CLI finished its turn and remains open | quiet CLI label |
| `Stopped` | the terminal's tmux session is gone | terminal row |

The CLI identity comes from a session-only wrapper lease. Older terminals may
be reconciled from an exact pane command and PID, but that fallback reports
presence only. A `working` report expires after 30 minutes of silence. Escape
or Ctrl+C also clears an active state immediately; arrow and Alt-key escape
sequences do not. No identity or activity means "no signal" — never a guess.

## What each CLI gets

| CLI | How PiCode injects (session only) | Coverage |
|---|---|---|
| Claude Code | `claude --settings <picode json>` | working / needs-you / idle |
| Codex | invocation-only lifecycle hooks, trusted by their exact command hashes | prompt, needs-you, idle, interrupt |
| Grok | `GROK_HOME` overlay in PiCode's data dir; your `auth.json` is symlinked | session start, prompt, needs-you, idle |
| Pi | `pi -e <picode extension>` | session start, prompt, needs-you, settled |

Outside PiCode, `which claude` and `which pi` still resolve to your real
binaries. Codex hooks are passed with `-c` for that invocation and trusted
one by one; PiCode never enables Codex's blanket
`--dangerously-bypass-hook-trust` switch.

## CLI identity and lifecycle lease

The wrapper sends `start` and `end` to
`POST /api/terminals/{id}/runtime` with a canonical CLI, process id, and a
run id. Hook reports include that run id, so an old process cannot clear the
state of a newer process in the same terminal. The lease is memory-only and
is revalidated against the live pane; a daemon restart therefore waits for a
new wrapper report or an exact tmux command/PID fallback.

## Pi TUI isolation

The Pi extension listens to native lifecycle events. It reports only when
the process has a PiCode terminal id **and** is running in TUI mode. A Pi
TUI you launch manually gets terminal status; managed Pi agents (RPC),
`pi -p`, JSON output, and non-TUI child agents do not emit a duplicate.
Completion uses `agent_settled`, after retries, compaction, and queued
follow-ups are finished.

For agent runs, the wrapper adds one `-e` argument before yours. Your own
`-e` extensions and all other agent arguments still reach Pi in their
original order. Commands such as `pi auth`, `pi install`, and `pi --version`
bypass injection so Pi still sees the command or flag first.

## If a CLI has no launch injection

PiCode does not fall back to editing your home files. The row stays a plain
terminal (honest). A later fallback may use the terminal bell; pixels are not
scraped.
