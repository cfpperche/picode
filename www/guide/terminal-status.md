# Terminal status for coding CLIs

A coding CLI running inside a PiCode terminal can report its own
lifecycle, so the sidebar shows when it is **working** and when it
**needs you** — the same states Pi agents already show (ADR-0056).

PiCode does not read the terminal's pixels, and it does **not** write
your `~/.claude`, `~/.codex` or `~/.grok`. You type `claude` (or
`codex`, `grok`) as usual. Inside a PiCode terminal only, a wrapper on
that session's PATH launches the real binary with the flags or overlay
that CLI accepts.

Turn it on in **Preferences → Terminal status**. New terminals after
that pick up the intercept; terminals that already existed need to be
recreated.

## States

| State | Meaning | Where you see it |
|---|---|---|
| `working` | the CLI started a turn and has not finished | spinner on the terminal row and tab |
| `needs-you` | the CLI is waiting on you (permission prompt, question) | accent "Needs you" chip |
| `idle` | the CLI finished its turn | nothing — quiet means idle |

A `working` report expires after 30 minutes of silence. No chip means
"no signal" — never a guess.

## What each CLI gets

| CLI | How PiCode injects (session only) | Coverage |
|---|---|---|
| Claude Code | `claude --settings <picode json>` | working / needs-you / idle |
| Codex | `codex -c notify=[reporter, idle, …]` | end of turn only |
| Grok | `GROK_HOME` overlay in PiCode's data dir; your `auth.json` is symlinked | session start, prompt, needs-you, idle |

Outside PiCode, `which claude` is still your real binary.

## If a CLI has no launch injection

PiCode does not fall back to editing your home files. The chip stays
absent (honest). A later fallback may use the terminal bell; pixels
are not scraped.
