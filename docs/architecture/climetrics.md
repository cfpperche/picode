# Cross-CLI usage metering (`internal/climetrics`)

**ADRs**: [0042](../decisions/0042-dashboard-v2-breakdowns.md) (dashboard
breakdowns), [0097](../decisions/0097-cross-cli-dashboard-metrics.md)
(cross-CLI metrics), [0127](../decisions/0127-dashboard-measures-the-machine.md)
(the dashboard measures the machine).
**Serves**: `GET /api/sessions/stats` (`internal/server/session_stats.go`).
**Reads**: the session stores `internal/clisession` already lists — nothing
is written, ever.

The dashboard used to measure pi and only pi while PiCode ran nine CLIs, so
the headline number was a fraction of the real one. This package turns what
each CLI recorded about its own sessions into one window.

## The shape

```
GET /api/sessions/stats?range=…
        │
        │  statsForRange: cache by (root|range, Fingerprint(meters))
        ▼
   Aggregate(Request, Meters())
        │
        ├── PiMeter ──────┐
        ├── ClaudeCode…   │  one goroutine per meter, uncapped Windows
        ├── Codex…        │
        ├── OpenCode…     │
        ├── Hermes…       ├──►  merge()  ──►  FleetStats
        ├── Grok…         │     (ranking, top-N, per-CLI pivot,
        ├── Omp…          │      coverage matrix, series)
        ├── Muse…         │
        └── Agy… ─────────┘
                │
            parseCache: memoised by (path, size, mtime)
```

One file per CLI (`pi.go`, `claudecode.go`, `codex.go`, `opencode.go`,
`hermes.go`, `grok.go`, `omp.go`, `muse.go`, `agy.go`), each implementing
`Meter`:

```go
type Meter interface {
	CLI() string
	Label() string
	Fingerprint() string        // change detector; "" forces a cache miss
	Meter(req Request) (Window, error)
}
```

Adding a CLI is one file plus a line in `Meters()`. Coverage, the by-CLI
pivot and every breakdown pick it up from the `Window` it returns.

## Silence is a value, not a zero

The rule the package exists to enforce: a signal a CLI never writes comes
back as a `State`, never as `0`. A dashboard that prints `$0.00` for Codex
is lying; one that prints `—` beside a reason is not.

| `State` | Means | The surface must show |
|---|---|---|
| `reported` | the CLI writes it and the adapter read it | the number |
| `not-reported` | the CLI does not record this signal at all | `—` and the reason |
| `partial` | recorded, but not for every session in the window | the number **and** what it is missing |
| `unavailable` | recorded, but this machine could not be read | `—` and the fix |

Every meter answers all ten `Signals` (cost, tokens, model, messages,
turns, tools, errors, impact, timing, limits) in a fixed order, so the
coverage panel renders a matrix instead of a ragged list — and a CLI that
reported nothing this window still gets a row. **The live answer for which
CLI reports what is the coverage matrix itself, not this file**: it is
computed from the session stores on the machine in front of you, and a
table copied here would be stale the first time a vendor ships a field.

Three consequences worth knowing before reading a number:

- **`Billing` is never inferred.** A meter reports what the CLI itself
  states; the operator sets the rest in `cli_configs`. It changes what a
  cost means (metered key vs. list-price equivalent under a plan), so
  `CLIBucket.CostState` travels beside `Cost` and the row can say "not
  priced" instead of a zero that reads as "free".
- **`Timing.SessionMs` is agent time, not clock time.** Sessions run
  concurrently; a 7-day window measured 226 hours, 32 per day. Labelling it
  "elapsed" would be false.
- **`Impact` carries no file count.** OpenCode records `summary_files` and
  Claude Code does not, so a merged total would describe a sliver of the
  lines beside it. The capability stays visible in the coverage matrix and
  the field returns when a second CLI reports it.

## Why the cuts happen once

Meters return **uncapped** windows and `merge` does the ranking and the
top-N over the union. Capping per CLI would drop pi's ninth tool before it
was ever compared with Claude Code's first.

`Aggregate` runs the meters concurrently and merges single-threaded, so the
wall time is the slowest meter rather than the sum. A meter that fails does
not fail the window: its row comes back `unavailable` with the error as the
note.

## The cache is what makes the poll affordable

`parseCache` memoises the *parse* of a session file by `(path, size,
mtime)` — deliberately window-independent, because caching a windowed
result would be thrown away whenever the range changed, while the parse
survives every range and the windowing that remains is arithmetic over a
slice.

Measured before it existed: a cold 7-day window over all nine CLIs took
5.8s, `all` took 11.4s, and agents write constantly, so the fingerprint
changed on almost every poll and almost every poll paid it. The budget is
`maxCachedEntries` (400k entries, not files — session files differ by four
orders of magnitude in size) and eviction drops the least recently used
file whole, never half of one.

Above it, `internal/server/session_stats.go` caches the finished
`FleetStats` by `(root|range, Fingerprint(meters))`. An empty fingerprint
component forces a miss: a meter that cannot describe its own state must
not be served from a cache that assumes it can.

## What it never does

- **Never writes.** Every store belongs to another vendor's CLI.
- **Never filters by workspace.** The dashboard measures the whole machine
  (ADR-0127); labelling which folders a workspace claims is the server
  layer's job.
- **Never carries message content.** Meters read counts and figures;
  `TestClaudeCodeNeverCarriesMessageContent` holds that line.

## Open items

Debts and unresolved questions live in
[`docs/handoff/open/dashboard.md`](../handoff/open/dashboard.md).
