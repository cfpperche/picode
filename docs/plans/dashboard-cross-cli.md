# Dashboard — the cross-CLI generation

> **Status: direction approved by the owner (2026-09-07). Phases 1-4 built.**
> The owner chose one combined spend total (not two columns), Codex's quota
> window in place of a price, all four new metric families, and "ingest
> everything, filter in the view".
> **Naming.** "Dashboard v2" is already ADR-0042. Following the precedent in
> `docs/plans/git-graph-actions.md` (which renamed its own generation because
> "Git graph v2" was ADR-0038), this generation is **the cross-CLI generation**,
> not "v3".
> **Study:** [../benchmarks/2026-09-07-cross-cli-agent-telemetry.md](../benchmarks/2026-09-07-cross-cli-agent-telemetry.md)
> **Decision:** [../decisions/0097-cross-cli-dashboard-metrics.md](../decisions/0097-cross-cli-dashboard-metrics.md)

---

## Context

The dashboard measures Pi, and only Pi. PiCode now runs **six** agent CLIs
(`internal/clilaunch/config.go:23-32`: pi, claude-code, codex, grok, hermes,
opencode), and five of them are invisible to every number on the screen.

This is not a rounding error. Measured on this machine, 2026-09-07:

| | 7-day window |
|---|---|
| What the dashboard reports (Pi only) | **$517.04** |
| Claude Code, same window, priced from its own `cost-state` records | **$1,399.30** |
| Combined | **$1,916.35** |

The surface is reporting roughly **27% of actual agent spend** and says nothing
about the other 73%.

> **Correction (during Phase 1).** An earlier draft of this plan put the Claude
> Code figure at $2,326.65. That number charged each session's whole lifetime
> cost to the window its file was last touched in — the naive attribution that
> ADR-0041 refused for exactly this reason. Priced correctly (each session's
> cost spread across the messages that earned it, so only the in-window share
> counts), the figure is $1,399.30; both the Go implementation and an
> independent script agree to the cent. The naive whole-session figure for
> *all time* is $2,895.08.

Every card is silently captioned "Pi only" without the
caption. `SPEND BY MODEL` in the current screenshot shows `openai-codex`, `xai`,
`zai`, `opencode` — those are **Pi provider ids**, not other CLIs
(`web/shared/domain/providerIcon.js:6-34`), which makes the omission harder to
notice, not easier.

The gap is a policy, not a limitation. `internal/clisession` already indexes all
six CLIs read-only, and says so in a comment (`clisession.go:21-24`):

> `// Cost is intentionally pi-only on this surface: Claude Code, Codex, Grok,`
> `// Hermes and OpenCode listings do not populate it. Hermes and OpenCode store`
> `// spend in SQLite, but the guest session row does not render cost.`

The data is on disk, already being opened, and being dropped on the floor.

**Intended outcome:** one dashboard that answers "what did my agents cost, do,
and break this week" across every CLI — and that says out loud, per CLI and per
metric, what it cannot measure instead of printing a zero.

---

## 1. Owner decisions (2026-09-07)

| # | Decision | Consequence |
|---|---|---|
| 1 | **One combined SPEND total.** Not two columns. Breakdown rows carry a `[sub]` / `[api]` billing badge. | Total mixes real API spend with subscription list-price equivalents. Accepted knowingly; the badge and the coverage panel carry the honesty. |
| 2 | **No pricing table for Codex.** Its cost slot shows **quota-window consumption** instead (`used_percent`, window, `resets_at`, `plan_type`) — data Codex actually reports. | PiCode never ships a price list that ages silently. Codex contributes $0 to the total and must be named as excluded, never rendered as a zero row. |
| 3 | **All four new metric families** ship: code impact, time/latency, limit headroom, cost efficiency. | Three of these were refused in ADR-0042 *for lack of data*. The guest CLIs carry the data Pi never had. Each refusal must be re-opened in writing. |
| 4 | **Ingest everything, filter in the view.** A `PiCode / this machine` scope control. | Second piece of view state beside the range. Both stay in `localStorage`, never the hash router (ADR-0041's refusal: hash routes name *what object is open*, never a view filter). |

---

## 2. What each CLI actually reports (verified on disk, 2026-09-07; Grok re-verified 2026-09-11)

Every ✅ below was read out of a real file on this machine, not inferred from docs.

| Signal | pi | claude-code | codex | opencode | hermes | grok |
|---|---|---|---|---|---|---|
| Cost | ✅ `usage.cost.total` | ◐ `cost-state.totalCostUSD` + `modelUsage[].costUSD` — see below | ✖ never priced | ✅ `message.cost` | ✅ `estimated_cost_usd` / `actual_cost_usd` | ◐ `usage.json` `costUsdTicks` (10¹⁰/USD), per turn, since 1.0.x |
| Tokens (in/out/cache/reasoning) | ✅ | ✅ `message.usage` | ✅ `token_count.info.total_token_usage` | ✅ `message.tokens` | ✅ dedicated columns | ◐ `usage.json` per turn |
| Model / provider | ✅ | ✅ `message.model` | ✅ `turn_context` + `session_meta.model_provider` | ✅ `providerID`/`modelID` | ✅ `model` | ✅ `summary.json` |
| Messages | ✅ | ✅ | ✅ | ✅ | ✅ `message_count` | ✅ prompt rows |
| Turns / errors / aborted | ✅ `stopReason` | ✅ `stop_reason` + `tool_result.is_error` | ~ `turn_aborted` | ✅ `finish` / `error` | ✅ `end_reason` | ✅ `events.jsonl` `turn_ended.outcome` + `tool_completed.outcome` |
| Tool calls | ✅ `toolCall` | ✅ `tool_use` | ✅ `function_call` | ✅ `part` rows | ✅ `tool_call_count` | ✅ `events.jsonl` `tool_started` |
| **Lines ± / files** | ✖ | ✅ `totalLinesAdded` / `totalLinesRemoved` | ✖ | ✅ `summary_additions` / `_deletions` / `_files` | ✖ | ✖ |
| **Timing** | ✖ | ✅ `totalAPIDuration` / `totalToolDuration` / `totalDuration` | ~ line timestamps | ~ `time_*` | ✅ `started_at` / `ended_at` | ✅ `events.jsonl` `loop_started`→`tool_started`/`turn_ended` windows + `tool_completed.duration_ms` |
| **Rate limits** | via `internal/usage` | ✖ | ✅ `rate_limits{used_percent, window_minutes, resets_at, plan_type}` | ✖ | ✖ | via `usage/grok_cli.go` |
| Billing mode | api | sub *or* api | sub *or* api | api | ✅ `billing_mode` column | — |

Three consequences drive the design:

- **Claude Code's cost is a floor, not a total (◐).** Measured 2026-09-07:
  only **46 of 340** transcripts carry a `cost-state` record at all, and none
  of the six most recently written ones do — a *live* session has no snapshot.
  The snapshot is also cumulative, so it can never be read as a window. The
  adapter therefore spreads each session's recorded cost across the messages
  that earned it (in-window share only) and reports coverage as `partial`
  with both counts. Two traps found while building it, both now regression-
  tested: the snapshot names models by billing id (`claude-opus-5[1m]`) while
  messages name them plainly, and matching the raw strings silently dropped
  $540 of a $1,001 session; and a model can be billed with no surviving
  message of its own, so its cost is spread flat rather than lost.
- **Grok's turns, tools and durations are real; its tokens and cost are new and partial (◐).**
  Grok writes a turn/tool timeline in every session's `events.jsonl` —
  measured 2026-09-11: 188 of 231 sessions non-empty, 176 with completed
  turns, 139 with tool calls — and prices and counts tokens per turn in
  `usage.json`, which it only began writing in 1.0.x (3 of 231 sessions
  here). The adapter reads all four files per session (prompt history,
  summary, events, usage) and reports tokens and cost as `partial` with
  both counts — 6 of 785 turns in a 60-day window. Durations prorate by
  turn rather than by token, or every session without a usage record would
  lose its time along with its tokens.
- **Hermes already solved decision #1's honesty problem in its own schema** —
  `estimated_cost_usd`, `actual_cost_usd`, `cost_status`, `cost_source`,
  `pricing_version`, `billing_mode`, `billing_provider`. Copy that vocabulary
  rather than inventing one.

---

## 3. The one thing that must not be got wrong: silent zeros

`.pi/skills/uiux-review/SKILL.md` makes this an instant FAIL:

> *"a zero-item list/section without a placeholder, or a '0' badge, is FAIL"*
> *"Unavailable choices are hidden, or the reason is visible text — not a
> disabled pill hinted only by `title`."*

So every metric, per CLI, resolves to exactly one of three states, and the type
system carries it — not a `float64` that happens to be 0:

| State | Renders as | Means |
|---|---|---|
| `reported` | the value | the CLI wrote it and we read it |
| `not-reported` | `—` + reason on the coverage panel | the CLI does not emit this signal (Codex cost; Grok's tokens/cost for the turns before its first `usage.json`) |
| `off` | `—` + "integration off" | `internal/server/term_intercept.go` wiring disabled for that CLI |

A `COVERAGE` panel renders the whole matrix from §2 as data, so the dashboard
explains its own blind spots on the surface instead of in a doc.

---

## 4. Backend

### 4.1 New package `internal/climetrics`

Mirrors `internal/clisession`'s file layout (one file per CLI) because the
parsers are siblings of the ones already there.

```
internal/climetrics/
  climetrics.go   // contract, Sources(), merge, coverage
  cache.go        // incremental per-file parse cache
  pi.go           // delegates to internal/session.StatsRoot (existing, tested)
  claudecode.go   codex.go   opencode.go   hermes.go   grok.go
```

The capability is optional and explicit — a source that cannot meter a signal
says so, and the merge records it as `not-reported`:

```go
// Meter aggregates one CLI's own session files over a window. A Source that
// does not implement Meter contributes coverage rows and nothing else.
type Meter interface {
    CLI() string
    Meter(from, to time.Time, loc *time.Location, scope Scope) (Window, error)
    Coverage() []Signal   // what this CLI can and cannot report, always
}
```

**Reuse, don't rewrite:**
- `pi.go` calls `session.StatsRoot` (`internal/session/stats.go:161-183`) —
  untouched, keeps its tests.
- `claudecode.go` / `codex.go` extend the `raw` structs already scanning these
  files line-by-line in `internal/clisession/claudecode.go:85-97` and
  `codex.go:65-79`. The scan loop exists; it needs more fields, not a new loop.
- `opencode.go` / `hermes.go` widen the existing read-only column whitelists
  (`opencode.go:77`, `hermes.go:75`) — the DBs are already opened.
- `internal/transcript` stays out of this. Its package doc rules usage out on
  purpose (`transcript.go:1-9`); metrics are a different concern from replay.

### 4.2 The incremental cache — the load-bearing decision

Today's cache is all-or-nothing: `session.Fingerprint` over one tree, and any
change invalidates all four ranges (`internal/server/session_stats.go:23-59`).
That does not survive the new footprint:

| Tree | Size | Files |
|---|---|---|
| `~/.pi/agent/sessions` | 437 MB | 665 |
| `~/.claude/projects` | 453 MB | 339 |
| `~/.codex/sessions` | **2.24 GB** | 915 |
| opencode + hermes SQLite | 98 MB | 2 |

Measured on this machine: a full JSON parse of Claude Code's 453 MB takes
**1.14 s in Python** (Go will be faster, but the shape holds); a cheap
`cost-state`-only pass is 0.42 s. Any active session appends constantly, so an
all-or-nothing fingerprint would re-parse ~3.1 GB on most 60-second polls.

Design:

1. **Per-file parse cache** keyed `(path, size, mtime)` → per-day aggregate.
   A poll stats the file list and re-parses only what changed. Bounded by entry
   count with LRU eviction.
2. **Date-partition skipping for Codex.** `~/.codex/sessions/YYYY/MM/DD/…` means
   a `today` or `7d` window can skip whole directories without stat-ing their
   contents. This is what keeps the largest tree cheap.
3. **SQLite sources** key on `(db mtime, size)` plus a `MAX(time_updated)` probe
   — one indexed query, no table scan.
4. Keep the existing `statsCache` in front of the merged result.

**Budget to hold, asserted by a test:** warm poll = stat sweep only, no file
opened; cold `range=all` across all six sources under 3 s.

### 4.3 Contract and endpoint

**Extend `GET /api/sessions/stats` in place; do not fork a second endpoint.**
Two consumers already read it — `web/mobile/src/App.jsx:149` (`range=today`) and
`web/desktop/src/components/Providers.jsx:98-107` (`range=7d`, reads
`byProvider`). Both keep working on a superset, and get the corrected numbers for
free. New param: `?scope=picode|machine` (default `picode`), clamped the same way
`normalizeRange` clamps rather than 400-ing (`session_stats.go:121-128`).

ADR-0042 already flagged `WindowStats` as an eleven-field contract agreed by
three layers. Adding six flat fields makes that worse, so the new facts nest:

```go
type WindowStats struct {
    // unchanged, now cross-CLI
    From, To string
    Current  PeriodTotals
    Prior    *PeriodTotals
    ByProvider  []ProviderBucket   // + BillingMode (see below)
    ByModel     []ModelBucket      // + CLI
    ByWorkspace []WorkspaceBucket
    Tokens      TokenTotals
    Tools       []ToolBucket       // + CLI
    Turns       TurnStats
    TopSessions []SessionSpend     // + CLI
    Series      []DayBucket

    // new
    Scope    string          `json:"scope"`
    ByCLI    []CLIBucket     `json:"byCli"`     // the new pivot dimension
    Coverage []CoverageRow   `json:"coverage"`  // §3, always populated
    Impact   *ImpactTotals   `json:"impact,omitempty"`
    Timing   *TimingTotals   `json:"timing,omitempty"`
    Limits   []LimitWindow   `json:"limits,omitempty"`
}
```

**`ByProvider` must stay API-billed only.** `spendByProvider()`
(`web/shared/domain/providerRows.js:121-130`) joins it onto credential rows in
the Providers view. Attributing subscription-covered Claude Code spend to the
`anthropic` account row would overstate what that API key cost. `ProviderBucket`
gains `BillingMode`; Providers.jsx sums `api` rows only.

**Privacy invariant extends unchanged.** The sabotage regression test at
`internal/server/session_stats_test.go:114-141` asserts no message content
escapes. It must be extended to every new adapter — five more transcript formats
are five more chances to leak a preview string.

### 4.4 Billing mode

`cli_configs` already exists (`internal/store/migrations/026_cli_terminals.sql`).
Add `billingMode ∈ api | subscription | unknown`, default `unknown`, editable on
the existing Agent CLIs surface. Hermes overrides from its own `billing_mode`
column. **Never inferred** — a guess here silently changes what the headline
number means.

### 4.5 Fidelity fixes found during research (fold into Phase 1)

| Finding | Why it matters now |
|---|---|
| `entryTS()` (`internal/session/transcript.go:167-177`) only reads `message.timestamp`, so compactions fall back to **file mtime** (`stats.go:300-305`). Real compaction lines today *do* carry a top-level RFC3339 `timestamp`; the comment justifying the fallback is stale. | This is exactly the "fabricated spike day" failure ADR-0041 refused for cost. A long session touched today dumps its whole compaction history into today. |
| **523 of 580** pi session directories are leaked Go-test temp dirs (`--tmp-TestInboxApp…--/exact.jsonl`). | They inflate `Fingerprint`'s file count and the `all` scan, and will appear as `byWorkspace` rows the moment one ever gets a message line. |
| `costFrom()` (`internal/session/session.go:339-343`) reads only `usage.cost.total` and discards `cost.{input,output,cacheRead,cacheWrite}`. | The EFFICIENCY panel the owner asked for needs per-type cost. It is already on disk. |
| `current.messages` counts user+assistant; `byModel[].messages` and `tokens` count assistant only (`stats.go:430-461`). | Two different denominators under one word. Rename in the payload so the cards can't lie. |

---

## 5. Frontend

Everything lands in the existing shell: `.dashboard-view` scroller,
`.dash-inner` 1040px box, `.dash-grid-2` CSS-columns masonry
(`web/desktop/src/styles/app.css:1306-1477`). No new layout system.

**Refusals that stand** (ADR-0041/0042, re-affirmed):
- No charting library. New geometry follows the `lib/barchart.js` +
  `DailyChart.jsx` split, itself mirroring `lib/gitgraph.js` + `GitGraph.jsx`.
- No green/red delta colouring — PiCode still has no budget concept.
- No new hash route. Scope joins range in `localStorage`
  (`web/desktop/src/lib/openTabs.js:116-130`, key `picode-dash-scope`).
- **The DAILY chart stays single-series.** Stacking six CLIs would force a
  categorical palette, which `RankedBars.jsx` explicitly refuses ("the row label
  is already the identity"). The `BY CLI` card carries the split instead.

| Card | Change |
|---|---|
| Header | + scope segmented control beside the range picker. Same `--ctl-h: 36px` rhythm, native radio + label like `.dash-range`. |
| SPEND / ACTIVITY / SESSIONS | Source becomes cross-CLI. SPEND is one number (decision #1). |
| FLEET | Unchanged — still client-side from props (`dashboardStats.js:19-38`). |
| **BY CLI** *(new)* | `RankedBars` + `[sub]`/`[api]` badge + the existing `CLI_MARKS` glyphs (`Cl`, `Cx`, `G`, `H`, `Oc`, `π` — `web/shared/domain/terminalCli.js`). Top-left of the grid; this is the story of the release. |
| SPEND BY MODEL / TOOLS / TOP SESSIONS | Rows gain a CLI mark. **Tool names are not normalized** — `Bash` / `bash` / `shell` stay distinct. Inventing equivalences across vendors is the kind of quiet wrongness this repo refuses. |
| TOKENS | Cross-CLI + per-type cost from §4.5. |
| **IMPACT** *(new)* | Lines ±, files touched. claude-code + opencode report; others `—`. |
| **TIME** *(new)* | Active vs wall time, API vs tool duration. claude-code + hermes report. |
| **LIMITS** *(new)* | Quota windows: `used_percent`, window length, reset time. Codex from its own `rate_limits`; pi providers and grok from `internal/usage`. This is Codex's cost cell (decision #2). |
| **EFFICIENCY** *(new)* | Cache hit per CLI, cost/turn, cost/line-changed. Reuses the `cacheHit` formula at `stats.go:533-537`. |
| **COVERAGE** *(new)* | The §2 matrix as a panel. What makes the omissions legible instead of invisible. |
| TOP SESSIONS routing | `SessionsView.jsx:156` defaults `cli = "pi"`; a Claude Code row currently lands on Pi's tab. Pass the row's CLI. |

Mobile consumes the same endpoint (`web/mobile/src/App.jsx:149` →
`StatStrip.jsx`). Its "Spend today" will jump when the total goes cross-CLI —
intended, but it must be exercised, not assumed.

---

## 6. Phases

Each ends green and shippable on its own.

| Phase | Content | Ends when |
|---|---|---|
| **1** | `internal/climetrics` contract + coverage; pi and claude-code adapters; §4.5 fidelity fixes. New payload fields present, UI untouched. | `/api/sessions/stats` returns `byCli` + `coverage` for two CLIs; privacy test extended; `make ci-scoped` green. |
| **2** | codex, opencode, hermes, grok adapters + the incremental cache. | **Done.** Warm poll: 31 ms (today) / 46 ms (7d) / 112 ms (all), against 2.37 s before the cache. Cold: 1.78 / 3.76 / 4.76 s — the 3 s target for cold `all` was **missed and accepted**; it is a once-per-process cost behind a skeleton. Fingerprint 14 ms. |
| **3** | UI cross-CLI: BY CLI card, billing badges, scope control, COVERAGE panel, CLI marks. | **Done.** Desktop and mobile exercised on a scratch instance against the real stores. Seven rendering lies found and fixed (see ADR §"What the surface had to be taught"). TopSessions carries a CLI mark rather than routing per CLI: `#/clis/sessions/<wsId>` is ADR-0079's shape and widening it is that ADR's call. |
| **4** | IMPACT, TIME, LIMITS, EFFICIENCY panels. | **Done.** Code impact (+10,724/−998), Agent time (17h waiting, 9h tools, 144h sessions open — labelled agent time, not elapsed), Limits (Codex weekly at 81%, resets in 6d), Efficiency (91% cache hit, cost per turn/line). |
| **5** | Close: ADR + benchmark study + screenshots + CHANGELOG + handoff note. | `make close` green on the branch. |

---

## 7. Docs and process gates

- **Worktree.** `make worktree NAME=dashboard-cross-cli`. Never commit feature
  work on `main` (enforced by `.githooks/pre-commit`).
- **Benchmark study** — new dated file `docs/benchmarks/2026-09-07-cross-cli-agent-telemetry.md`,
  in the house format (adapts / does-not-copy tables). Sources gathered:
  [Claude Code OTel metrics](https://code.claude.com/docs/en/monitoring-usage),
  [ccusage cost modes](https://ccusage.com/guide/cost-modes),
  [Codex CLI OTel](https://signoz.io/docs/codex-monitoring/),
  [codex-trace](https://github.com/PixelPaw-Labs/codex-trace),
  [claude-code-otel](https://github.com/ColeMurray/claude-code-otel),
  [CliDeck](https://yetanotherorchestrator.app/apps/clideck/),
  [awslabs/cli-agent-orchestrator](https://github.com/awslabs/cli-agent-orchestrator).
  Must add a row to `docs/benchmarks/README.md`.
- **ADR** — next free number is **0097** (0096 is claimed but uncommitted in this
  checkout). Re-check `ls docs/decisions .worktrees/*/docs/decisions` at the
  moment of writing, per AGENTS.md, and carry the `**Number**` bullet noting the
  renumber risk. The ADR must **re-open in writing** the three ADR-0042 refusals
  this generation reverses (lines-of-code, latency, and the cost-is-pi-only
  policy in `clisession.go:21-24`), each with what changed — per AGENTS.md
  non-negotiable #6.
- `CHANGELOG.md` `[Unreleased]` entry; `docs/handoff.md` stays ≤100 lines;
  session note in `docs/handoff/`.

**Explicitly not adopted** (carry into the ADR's Refuse table): OpenTelemetry
export (PiCode has zero OTEL today, and a collector is infrastructure a
single-operator tool should not require); a Codex price list; budgets, alerts or
projections; a charting library; per-CLI categorical palette.

---

## 8. Verification

1. **Unit** — `internal/climetrics/*_test.go` with a small fixture transcript per
   CLI, following `session.TestRoot`'s precedent. Assert coverage states, not
   just totals: a CLI that cannot report a signal must produce `not-reported`,
   never `0`.
2. **Privacy** — extend the sabotage test (`session_stats_test.go:114-141`) to
   all six adapters: no message content in any payload.
3. **Performance** — a test that asserts the warm path opens no file, and a
   measured cold-scan bound.
4. **JS** — `node --test` for the new pure modules in `web/shared/domain/`.
5. **Dogfood on a populated DB** — `scripts/qa-scratch.sh` against a **copied**
   database, not an empty scratch (an empty instance does not exercise the
   sidebar/boot paths that have broken before). Override `server.port` in the
   copy; reach it over `https://localhost` so `crypto.randomUUID` exists.
6. **Visual** — `agent-browser --session dashboard-cross-cli
   --ignore-https-errors`, open with `?cb=$(date +%s)` to defeat the HTML cache,
   scope selectors to the visible view (hidden pages stay mounted). Verify the
   bundle served matches the bundle loaded before blaming code. Run
   `window.__picodeOverlayAudit()`; read the PNG, don't trust the eval JSON.
7. **Both shells** — desktop *and* mobile (`StatStrip`), since both mount from
   the same endpoint and mobile has caught mount bugs desktop masked.
8. **Gates** — `uiux-review` and `visual-review` verdict lines in the reply;
   `make ci-scoped` while iterating, `make ci` once on `main` at the merge.

---

## 9. Still open

| # | Question | Recommendation |
|---|---|---|
| 1 | Should the durable turn/duration signal come from the intercept hooks instead of file parsing? The wrappers already fire `UserPromptSubmit`→working and `Stop`→idle with a `runId`, but `TermState`/`TermRuntime` are **in-memory only** and lost on restart. | Not in this generation. Persisting hook transitions would give turn count and duration for all five guests without reading their files — a cleaner source than transcripts, but it is a new durable table and belongs in its own ADR. |
| 2 | Does `range=all` stay honest once it means "every CLI, all time"? Claude Code all-time on this machine alone is $2,895. | Ship it; the number is real. Revisit only if the cold scan bound in Phase 2 cannot be held. |
| 3 | Grok used to render as activity-only; since the 2026-09-11 re-measure it reports turns, tools and durations for every session with an `events.jsonl`, and tokens/cost where `usage.json` exists. Is the split worth the extra parsing? | Keep it. The timeline is what nearly every session has, and the `partial` tokens/cost row states its own gap instead of hiding it. |
