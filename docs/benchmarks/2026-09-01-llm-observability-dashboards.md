# Study: LLM cost/observability dashboards, for the session dashboard (ADR-0041)

- **Date:** 2026-09-01
- **Sources:** public product documentation and UI conventions for
  Langfuse, LangSmith, Helicone (LLM tracing/cost observability — the
  closest real comparable to what PiCode's agents are), Grafana (stat and
  time-series panel conventions), and Stripe Dashboard (already named as
  an inspiration in `docs/benchmarks.md`'s own UI/UX list, for progressive
  disclosure and empty states that teach). No source clone, unlike the
  Cursor/t3code/paseo/herdr studies — these are hosted products; this is
  domain-convention knowledge, not receipts from a checked-out repo.
- **Scope:** what shape a *cost/activity dashboard* should take. Not a
  bar for anything else in the app — the composer statusbar and the
  session chip's own token/cost display (ADR-0031's context,
  `docs/design/session-surface-roadmap.md`'s D2) are untouched by this.

## Why these four, not Cursor/t3code/paseo

None of the three sanctioned benchmarks (or herdr) has a dashboard,
analytics, or cost-aggregation surface — they're composer/session-state
and PTY-runtime studies. `docs/benchmarks/README.md`'s own ritual is to
add a new dated study when that happens, rather than stretch an unrelated
citation to cover a surface it never addressed.

## What PiCode adapts

| Convention | Where it's from | PiCode's version |
|---|---|---|
| Date-range selector → headline stat row → breakdown | Langfuse/LangSmith/Helicone dashboards all lead with a period picker driving everything below it | One shared `DateRangePicker` (Today/7d/30d/All), native radio segmented control matching this app's existing `.termset-seg`/`.create-seg` pattern — not a new widget |
| Stat panel = big value + delta + sparkline | Grafana's Stat panel; Stripe's revenue/charges/payouts cards | `StatTile.jsx` — value, a neutral (never green/red) delta chip, a small SVG sparkline |
| Progressive disclosure; empty states that teach, not a blank well | Stripe Dashboard (per `docs/benchmarks.md`'s own list) | Genuinely empty period still shows honest `$0.00`/`0`, with one line ("No session activity in this period.") instead of an empty chart |
| Cost broken down by a small set of named dimensions (model/provider) | Every LLM cost tool leads with a provider/model breakdown, since that's usually the actionable lever | "Spend by provider" ranked bar list |

## What PiCode explicitly does not copy

| Not copied | Why |
|---|---|
| Trace-level waterfall / span drill-down | That's what the session chip and transcript view already are (D2, D3) — this dashboard is the aggregate layer above them, not a duplicate of either |
| Budgets, spend alerts, thresholds | No budget concept exists anywhere in PiCode; inventing one just to color a delta chip would fabricate a judgment the product doesn't otherwise make |
| Multi-tenant / team cost centers | Single-operator tool, no org hierarchy (consistent with ADR-0003's one-user-installed-runtime posture) |
| A charting library (all four of these tools ship one) | Hand-rolled SVG, mirroring this app's own `lib/gitgraph.js` + `GitGraph.jsx` split — a deliberate simplicity trade given the chart forms needed (sparkline, ranked bar) are simple enough not to need one, not an oversight |
| Live/streaming updates while the tab sits open | Out of scope for v1 — the dashboard fetches on mount and on range change, not on a poll; see ADR-0041's accepted-cost note on `range=all` having no cache |

## v2 addendum (same day, ADR-0042)

The owner dogfooded v1 and asked for a v2 built on a wider benchmark set.
Added sources, all public product documentation or open-source READMEs:

| Source | What it shows | What v2 takes |
|---|---|---|
| [ccusage](https://github.com/ryoppippi/ccusage) — CLI over Claude Code's local JSONL, the closest data-model twin to PiCode's scan | daily/weekly/monthly/session/5-hour-block reports, per-model breakdown, cache tokens split out, per-project grouping (`--instances`), burn rate | per-**model** rows, tokens by **type**, spend by **workspace**, top sessions |
| [Claude Code OpenTelemetry metrics](https://code.claude.com/docs/en/monitoring-usage) + [Grafana dashboard 25052](https://grafana.com/grafana/dashboards/25052-claude-code/) | cost/token usage by model and type, session count, active time, **top tools**, lines of code, commits, edit accept rate, **API errors** over time | Sessions tile, Tools ranking, Reliability facts (errors / aborted from `stopReason`), turns per day |
| [Claude Code Analytics API](https://platform.claude.com/docs/en/build-with-claude/claude-code-analytics-api) | per user/day: sessions, lines added/removed, commits, PRs, tool accept/reject, per-model tokens (input/output/cache_read/cache_creation) + cost | vocabulary check: same four token types, cost per model |
| [pi-agent-dashboard](https://github.com/BlackBeltTechnology/pi-agent-dashboard) — same pi backend | per session: state, live cost, tokens, model, thinking level, context bar, elapsed; grouped by folder/branch | grouping by folder; Fleet tile shows state per running agent |
| [nicknisi/fleet](https://github.com/nicknisi/fleet) | summary strip "N need you · N working · N ready · N idle", rows sorted by urgency | Fleet tile strip: working / waiting / idle, waiting alone takes the accent |
| [Marc Nuri's agent dashboard](https://blog.marcnuri.com/ai-coding-agent-dashboard) | card per session: repo/branch, model, context-window %, state; notifications on approval / PR / done | context % refused for now (one fetch per agent) — see ADR-0042 |
| [5of10 dashboard rules](https://5of10.com/articles/dashboard-design-best-practices/), Grafana, Vercel | 5–7 KPIs max, most critical top-left, comparison context, sparkline, "last updated", direct labels not legends, trend in the middle, breakdowns below, skeleton + empty states, date picker | four tiles, daily chart in the middle, 2-column breakdown grid, tail folded into "N more", tokens labelled in rows not a legend |

Gaps in v1 that this set exposed, against this machine's real data
(8 models, 70 error + 33 aborted turns, 5,262 `bash` calls, a `$6.15`
"unknown" provider bucket, an inert Fleet tile, no auto-refresh): all
closed in ADR-0042.

### Still not copied

| Not copied | Why |
|---|---|
| Burn rate / projection (ccusage blocks) | Same refusal as budgets: no target to project against |
| Lines of code, commits, PRs (OTel, Analytics API) | Needs tool-arg diffing or git log per session; v3 candidate |
| p95 latency (Langfuse / Helicone) | pi's JSONL carries no request duration |
| Trace waterfall | Still the transcript view |
| Context-window % per agent on the home | One `/status` fetch per running agent; wait for an aggregate endpoint |

## Relation to ADR-0041

This study is the benchmark citation ADR-0041 and
`.pi/skills/uiux-review/SKILL.md`'s "Product benchmarks" checklist item
require. The dashboard itself, its data source, and its reconciliation
with the two prior "cost as a new page" / "X is not the home" refusals
are in the ADR, not repeated here.
