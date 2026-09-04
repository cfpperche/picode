# Terminal status for coding CLIs

A coding CLI running inside a PiCode terminal can report its own
lifecycle, so the sidebar shows when it is **working** and when it
**needs you** — the same states Pi agents already show (ADR-0056).

PiCode does not read the terminal's pixels, and it does **not** write
your `~/.claude`, `~/.codex`, `~/.grok`, or `~/.pi`. You type `claude`,
`codex`, `grok`, or `pi` as usual. Inside a PiCode terminal only, a
wrapper on that session's PATH launches the real binary with the flags,
extension, or overlay that CLI accepts.

Turn it on in **Preferences → Terminal status**. New terminals after
that pick up the intercept; terminals that already existed need to be
recreated.

## States

| State | Meaning | Where you see it |
|---|---|---|
| `working` | the CLI started a turn and has not finished | spinner on the terminal row and tab |
| `needs-you` | the CLI is waiting on you (permission prompt, question) | accent "Needs you" chip |
| `idle` | the CLI finished its turn | nothing — quiet means idle |

A `working` report expires after 30 minutes of silence. Escape or
Ctrl+C also clears an active state immediately; arrow and Alt-key escape
sequences do not. No chip means "no signal" — never a guess.

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

PiCode does not fall back to editing your home files. The chip stays
absent (honest). A later fallback may use the terminal bell; pixels
are not scraped.
