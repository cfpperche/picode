# ADR-0097: The dashboard measures every agent CLI, and says what it cannot

- **Status**: proposed (direction approved by the owner, 2026-09-07)
- **Date**: 2026-09-07
- **Amends**: [ADR-0041](0041-session-observability-dashboard.md), [ADR-0042](0042-dashboard-v2-breakdowns.md)
- **References**: [ADR-0069](0069-agent-cli-terminals.md), [ADR-0079](0079-sessions-under-agent-clis.md), [ADR-0088](0088-cross-cli-session-handoff.md)
- **Study**: [../benchmarks/2026-09-07-cross-cli-agent-telemetry.md](../benchmarks/2026-09-07-cross-cli-agent-telemetry.md)
- **Plan**: [../plans/dashboard-cross-cli.md](../plans/dashboard-cross-cli.md)
- **Number**: 0096 is the highest on `main` and in `.worktrees/*/docs/decisions`
  at the moment of writing; this file takes 0097 and may be renumbered at
  merge (precedent: `4502305f`).

## Context

The dashboard measures pi, and only pi. `internal/session/stats.go` scans
`~/.pi/agent/sessions` and nothing else, and `GET /api/sessions/stats` takes
no CLI parameter. PiCode has run six agent CLIs since ADR-0079/0088, and five
of them are invisible on the surface that claims to answer "what did this
week cost".

Measured on this machine, 2026-09-07, for the same 7-day window:

| | |
|---|---|
| What the dashboard reported (pi only) | **$517.04** |
| What the machine actually spent (pi + Claude Code) | **$1,916.35** |

The surface was showing 27% of the truth with no indication that a number
was missing. Worse, `SPEND BY MODEL` already listed `openai-codex`, `xai`,
`zai` and `opencode` — those are pi *provider* ids, not other CLIs, so the
omission read as coverage.

The gap was policy, not capability. `internal/clisession` already opens all
six stores read-only and says so in a comment: *"Cost is intentionally
pi-only on this surface … Hermes and OpenCode store spend in SQLite, but the
guest session row does not render cost."* The data was being opened and
dropped.

Two facts constrain any honest fix, both measured rather than documented:

1. **The CLIs record wildly different things.** Claude Code counts lines
   changed and request durations; Codex reports quota windows but never a
   price; Grok records prompt history and nothing else. Any single schema
   will have holes.
2. **Claude Code's cost is a floor.** Only 46 of 340 transcripts carry a
   `cost-state` record and none of the six newest do — a live session has no
   snapshot. The record is also cumulative for the session's whole life, so
   reading it as a window's spend would charge a month-old session's entire
   bill to the day its file was last touched: the fabricated-spike failure
   ADR-0041 refused, reached through a different door.

## Decision

The dashboard aggregates every agent CLI on the machine, and every metric it
shows carries a state saying whether that CLI could report it.

A new package `internal/climetrics` owns one `Meter` per CLI, reading the
same stores `internal/clisession` lists. Meters return *uncapped* windows;
ranking and the top-N cuts happen once over the union, so pi's ninth tool
meets Claude Code's first before anything is dropped. `GET
/api/sessions/stats` keeps its name and gains `?scope=machine|picode`,
defaulting to `machine` because the v1 surface counted every pi session
wherever it ran and a narrower default would silently drop rows.

Every metric resolves to `reported`, `partial`, `not-reported` or
`unavailable` — never to a `float64` that happens to be `0`. A dashboard
that prints `$0.00` for Codex is lying; one that prints `—` beside a reason
is not. A `coverage` payload carries the whole matrix so the surface can
explain its own blind spots, and a CLI whose store cannot be read degrades
to one coverage row instead of taking the dashboard down.

Spend is **one combined total** (owner's call), with a billing badge on each
breakdown row, because the alternative — two headline numbers — buries the
question the surface exists to answer. Claude Code's cost is derived by
spreading each session's own recorded cost across the messages that earned
it, so a window reports its share and a whole-session window reconciles to
the cent with what the vendor wrote. **Codex is never priced**: its cost
cell shows the quota window it does report (`used_percent`, `resets_at`,
`plan_type`), because PiCode will not ship a price list that ages silently.

Parsing is cached per file, keyed by `(path, size, mtime)`, and what is
cached is the *parse* rather than a windowed result — so one read answers
every range and survives every poll. This is not an optimisation detail: the
six stores are 3.1 GB on this machine, agents write to them constantly, and
the fingerprint therefore changes on almost every 60-second poll. Measured
here, a warm cross-CLI refresh went from **2.37 s to 46 ms** once the last
CLI joined the cache.

Guest CLIs contribute **no** `byProvider` rows. That breakdown feeds the
Providers view, which answers "what did the credential PiCode holds cost";
a guest signs in with its own account, so an `anthropic` row there would
bill a subscription's list-price equivalent to an API key that never paid
it. The cross-CLI spend story lives in `byCli` and `byModel`, where the CLI
is named.

## Consequences

- **Easier**: adding a seventh CLI is one `Meter` and one registry line —
  coverage, the by-CLI pivot and every breakdown follow from the `Window` it
  returns. The surface can finally answer "which CLI is the money", and
  three metric families ADR-0042 refused for lack of data become possible.
- **Harder**: the payload grew, and `WindowStats` was already an eleven-field
  contract three layers agree on. Nesting the new families behind `byCli`,
  `coverage`, `impact`, `timing` and `limits` keeps it navigable but does not
  make it small. Six vendors' formats are now six things that can drift.
- **Accepted cost**: the headline mixes metered API spend with
  subscription-covered list-price equivalents. The owner chose this over two
  columns; the badge and the coverage panel carry the honesty. Claude Code's
  per-day *shape* is approximate (cost follows tokens within a session) even
  though its totals are exact, and its per-model split is approximate for
  any model billed with no surviving message of its own.
- **If wrong**: the failure is a headline number that reads as money when it
  is partly a plan's list price. The mitigations are on the surface, not in
  a doc — the badge, the partial state and the coverage matrix — and the
  fallback is the two-column split the owner declined, which the `Billing`
  field already carries enough information to build.

## Measured

All on this machine, 2026-09-07, across all six CLIs.

| | cold (first parse) | warm (cached) |
|---|---|---|
| `today` | 1.78 s | **31 ms** |
| `7d` | 3.76 s | **46 ms** |
| `all` | 4.76 s | **112 ms** |

Fingerprint (the stat sweep every poll pays): **14 ms**.

The warm path is what the 60-second poll actually costs, and it is ~50x
better than the surface needs. The cold path misses the 3 s target this
work set itself for `range=all` — it is 4.76 s, the price of first-parsing
3.1 GB — and that is accepted rather than fixed: it is paid once per process
for a range the operator has to deliberately choose, behind the skeleton the
view already renders.

One correction worth recording: pi was the last CLI to join the cache and
turned out to be **2.32 s of a 2.37 s** warm refresh on its own, because it
alone still re-read its 437 MB tree every poll. The fix moved pi's parse
into `session.ParseFile` and drove it through the same cache as the guests,
which is also how the two parsers became one.

## Reopened refusals

ADR-0042 refused three things *for lack of data*, and AGENTS.md requires
naming what changed rather than quietly reversing them.

| Refused in ADR-0042 | What changed |
|---|---|
| Lines of code / commits ("v3 candidate") | Claude Code writes `totalLinesAdded` / `totalLinesRemoved`; OpenCode writes `summary_additions` / `_deletions` / `_files`. No diffing needed |
| p95 latency — "pi's JSONL carries no request duration" | Still true of pi. Claude Code writes `totalAPIDuration` / `totalToolDuration` / `totalDuration`; Hermes writes `started_at` / `ended_at` |
| Cost is pi-only (`clisession.go:21-24`) | A policy, not a limitation. It was hiding 73% of the window's spend |

## Corrections to prior behaviour

| Fixed | Was |
|---|---|
| Compactions bucket by the marker's own RFC3339 `timestamp` | Bucketed by file mtime, on a stale comment claiming pi writes no timestamp. Verified 62/62 compaction lines on disk carry one; the old path charged a long session's whole compaction history to the day it was last touched |
| `session.Fingerprint` returns `0:0:0` for a missing root | Returned `""`, which the new cache reads as "cannot tell". One uninstalled CLI would have disabled the dashboard's cache for every other one |
| Per-type cost (`usage.cost.{input,output,cacheRead,cacheWrite}`) is kept | Read and discarded; only the total survived |
| Three tests isolate `$HOME` before writing session fixtures | They wrote into the developer's real `~/.pi/agent/sessions`, leaving 535 of 593 session directories behind, named after the tests that made them |

## Refuse

| Temptation | Why not |
|---|---|
| An OpenTelemetry exporter (Claude Code and Codex both emit one) | Needs a collector, an endpoint and a retention story — infrastructure a single-operator tool should not require to answer "what did this week cost". The files are already on disk |
| A model price table, so Codex gets a dollar figure | It ages silently: the day a vendor changes a price every number stays plausible and wrong |
| Normalising `Bash` / `bash` / `shell` into one tool row | Three vendors' words; folding them asserts an equivalence we cannot back. The CLI mark disambiguates |
| Inferring billing mode from an auth file | A guess changes what the headline number means. The operator states it, or it stays `unknown` |
| Stacking the daily chart by CLI | Six series force a categorical palette, which `RankedBars.jsx` refuses on purpose |
| Budgets, alerts, projections, a charting library | Unchanged from ADR-0041/0042 |

## Alternatives considered

| Alternative | Why not |
|---|---|
| A second endpoint (`/api/fleet/stats`) beside the pi-only one | Two aggregations, two caches, and the mobile Now screen and Providers view would keep reading the wrong one. Extending the existing contract gives both the corrected numbers for free |
| Report only what every CLI has in common | The intersection is roughly "message counts". It would throw away lines changed, durations and quota windows — the whole reason guest CLIs are worth metering |
| Read the last `cost-state` as the window's cost | Simple, and wrong in the specific way ADR-0041 already refused: a month-old session's entire bill lands on one fabricated spike day |
| Persist turn/duration from the intercept hooks instead of parsing files | A cleaner source — the wrappers already fire `UserPromptSubmit`/`Stop` with a run id — but `TermState`/`TermRuntime` are in-memory only, so it needs a new durable table. Its own ADR, not this one |
| Omit CLIs that report nothing (Grok) | A CLI that is installed and silent is information. Hiding it recreates the invisibility this ADR exists to fix |
