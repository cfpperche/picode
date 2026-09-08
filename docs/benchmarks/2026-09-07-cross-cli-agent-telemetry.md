# Study: cross-CLI agent telemetry, for the cross-CLI dashboard (ADR-0097)

- **Date:** 2026-09-07
- **Sources:** two kinds, kept apart on purpose.
  *Published conventions* — Claude Code's OpenTelemetry metrics and events
  reference, Codex CLI's OTel configuration, ccusage's cost modes, and three
  local-first readers of the same files (ccusage, codex-trace,
  claude-code-otel) plus two multi-CLI orchestrator dashboards (CliDeck,
  awslabs/cli-agent-orchestrator).
  *Receipts from disk* — every claim about what a CLI actually records was
  read out of this machine's own stores on 2026-09-07, not inferred from the
  docs. Where a vendor's documentation and its files disagree, the files won.
- **Scope:** what an agent CLI records about its own sessions, and what a
  local aggregate may honestly say about it. Not a bar for the transcript
  view, the Providers roster, or quota fetching (`internal/usage`, ADR-0031),
  all of which keep their own sources.

## Why a new study

`docs/benchmarks/2026-09-01-llm-observability-dashboards.md` studied how to
*shape* a cost dashboard, and ADR-0041/0042 built it. It assumed one data
source, because PiCode had one: pi's JSONL. The question here is different
and newer — six agent CLIs now run under PiCode (ADR-0069/0079/0088), each
writing its own store in its own dialect. No existing benchmark covers "read
five other vendors' session files and say something true about all of them",
so this is a new dated study rather than a stretched citation.

## What each CLI actually records (measured, not documented)

| Signal | pi | claude-code | codex | opencode | hermes | grok |
|---|---|---|---|---|---|---|
| Cost | `usage.cost.total`, per message | only in a cumulative `cost-state` snapshot | never priced | `message.cost`, per message | `estimated_cost_usd` / `actual_cost_usd` columns | — |
| Tokens | per message | per message | `token_count.info.total_token_usage` | `message.tokens` | dedicated columns | — |
| Lines ± / files | — | `totalLinesAdded` / `totalLinesRemoved` | — | `summary_additions` / `_deletions` / `_files` | — | — |
| Durations | — | `totalAPIDuration` / `totalToolDuration` / `totalDuration` | line timestamps only | `time_*` | `started_at` / `ended_at` | — |
| Quota windows | — | — | `rate_limits{used_percent, window_minutes, resets_at, plan_type}` | — | — | — |
| Billing mode | — | — | — | — | `billing_mode`, `cost_status`, `pricing_version` | — |

Two findings changed the design, and neither is in any vendor's docs:

**Claude Code's cost is a floor, not a total.** Only **46 of 340**
transcripts carry a `cost-state` record, and none of the six most recently
written ones did — a *live* session has no snapshot. The snapshot is also
cumulative over the session's whole life, so reading it as a window's cost
would charge a month-old session's entire bill to the day its file was last
touched. That is the fabricated-spike failure ADR-0041 refused for cost,
reached through a different door.

**Model ids do not match across a vendor's own files.** The snapshot names
models by billing id (`claude-opus-5[1m]`) while messages name them plainly
(`claude-opus-5`). Matching the raw strings silently dropped **$540 of a
$1,001 session** — its largest model. Snapshot and message token totals also
disagree by ~1.6x on the same session, since compacted-away and sidechain
turns are billed but no longer on disk.

## What PiCode adapts

| Convention | Where it's from | PiCode's version |
|---|---|---|
| Read the CLI's own local files; no daemon, no API key | ccusage, codex-trace, claude-code-otel are all local-first readers of `~/.claude/projects` and `~/.codex/sessions` | `internal/climetrics`, one adapter per CLI, reading the same trees `internal/clisession` already lists |
| Cost from the vendor's own arithmetic, not a price list | ccusage's `display` mode shows Claude Code's pre-calculated cost rather than recomputing it | Per-model rate derived from the session's own snapshot (`costUSD` ÷ that model's surviving tokens), then applied per message |
| Separate "what it cost" from "what it was billed as" | Hermes' own schema splits `estimated_cost_usd` from `actual_cost_usd` and records `billing_mode` | A `Billing` field per CLI (`api` / `subscription` / `unknown`), never inferred, carried onto every breakdown row |
| Quota headroom is the real constraint on a plan | CodexBar and ccusage's 5-hour blocks; Codex records `used_percent` and `resets_at` itself | Codex's cost cell shows its quota window instead of a price — the owner's call, 2026-09-07 |
| One dashboard across several coordinated CLIs | CliDeck, awslabs/cli-agent-orchestrator | The by-CLI pivot and the coverage panel |

## What PiCode explicitly does not copy

| Not copied | Why |
|---|---|
| OpenTelemetry export (Claude Code and Codex both emit it; SigNoz/Grafana/CloudWatch guides assume it) | It needs a collector, an endpoint and a retention story — infrastructure a single-operator tool (ADR-0003) should not require to answer "what did this week cost". The files are already on disk and already complete enough |
| A model price table (ccusage's `calculate` mode) | It ages silently: the day a vendor changes a price, every number stays plausible and wrong. Codex therefore reports tokens and quota, never an invented dollar |
| Normalising tool names across CLIs (`Bash` / `bash` / `shell`) | Three vendors' words. Folding them together would assert an equivalence PiCode cannot back; the CLI mark disambiguates them instead |
| Burn rate, projections, budgets, alerts | Unchanged from ADR-0041/0042: no budget concept exists in PiCode |
| A charting library | Unchanged: hand-rolled SVG, the `lib/barchart.js` + `DailyChart.jsx` split |
| Stacking the daily chart by CLI | Six series would force a categorical palette, which `RankedBars.jsx` refuses ("the row label is already the identity"). The by-CLI card carries the split |

## Refusals this study reopens

ADR-0042 refused three things *for lack of data*. The data exists now, in
the guest CLIs rather than in pi, so each is reopened in ADR-0097 with what
changed:

| Refused in ADR-0042 | What changed |
|---|---|
| Lines of code / commits (named a "v3 candidate") | Claude Code writes `totalLinesAdded` / `totalLinesRemoved`; OpenCode writes `summary_additions` / `_deletions` / `_files`. No diffing required |
| p95 latency — "pi's JSONL carries no request duration" | Still true of pi. Claude Code writes `totalAPIDuration` and `totalToolDuration`; Hermes writes `started_at` / `ended_at` |
| Cost is pi-only (`clisession.go`'s "intentionally pi-only" comment) | The comment was a policy, not a limitation. Measured: the dashboard reported **$517.04** for a 7-day window in which the machine actually spent **$1,916.35** — 27% of the truth |

## Relation to ADR-0097

This study is the benchmark citation `.pi/skills/uiux-review/SKILL.md`'s
"Product benchmarks" checklist requires. The decision itself — the single
combined total, the coverage contract that refuses silent zeros, and the
scope filter — is in the ADR, not repeated here.

Sources: [Claude Code monitoring](https://code.claude.com/docs/en/monitoring-usage),
[Claude Code hooks](https://code.claude.com/docs/en/hooks),
[ccusage cost modes](https://ccusage.com/guide/cost-modes),
[Codex CLI OTel](https://signoz.io/docs/codex-monitoring/),
[codex-trace](https://github.com/PixelPaw-Labs/codex-trace),
[claude-code-otel](https://github.com/ColeMurray/claude-code-otel),
[CliDeck](https://yetanotherorchestrator.app/apps/clideck/),
[awslabs/cli-agent-orchestrator](https://github.com/awslabs/cli-agent-orchestrator).
