# ADR-0042: Dashboard v2 — model/workspace/token/tool/reliability breakdowns, live refresh, fingerprint cache

- **Status**: accepted, amends 0041
- **Date**: 2026-09-01

## Context

ADR-0041 shipped the no-tabs-open home as a spend/activity/fleet surface
and the owner dogfooded it the same day. The v1 scan read two things out
of every session JSONL — `usage.cost.total` and the last `model_change`
line — and threw the rest away. The rest is most of what the log holds
per assistant message: `provider`, `model`, `usage.{input,output,
cacheRead,cacheWrite,reasoning}`, `stopReason`, `content[].toolCall`,
plus `session.cwd` and `session_info.name` on the header lines. Against
this machine's real tree (84 files, 175 MB) that meant: 8 models in use
and no per-model row; 70 error and 33 aborted turns invisible; 5,262
`bash` calls with no tool view; spend by folder unavailable although
pi's sessions directory is already one folder per cwd; and a `$6.15`
**unknown** provider bucket caused purely by reading `model_change`
when the message itself names its provider.

The companion study (`docs/benchmarks/2026-09-01-llm-observability-
dashboards.md`, extended for v2) compared ccusage, the Claude Code
OpenTelemetry metrics and their Grafana dashboard, the Claude Code
Analytics API, pi-agent-dashboard (same pi backend), nicknisi/fleet and
the general dashboard-design rules. Every one of them leads with a
per-model cost breakdown and a token split with cache tokens separate;
the Claude Code ones add top tools, errors, sessions and lines of code.

## Decision

Same surface, same gate (`showHome`), same thesis as 0041 — metrics, not
navigation; no chart library; no budget or alert; no new route. What
changes:

- **One scan, more facts.** `internal/session/stats.go`'s single pass now
  also collects per-model cost (`byModel`), per-folder cost with session
  counts (`byWorkspace`), token totals by type with the composer status
  bar's own cache-hit formula (`tokens`), tool-call counts (`tools`, top
  8), turn/prompt/error/aborted/compaction counts (`turns`) and the five
  costliest sessions by name/cwd/last-activity (`topSessions`). The
  prior window still contributes only the headline totals. An assistant
  message's inline `provider`/`model` is the running truth for the
  user/tool lines after it; `model_change` is only the fallback.
- **Workspace labels stay a server concern.** The `session` package never
  sees the store; `handleSessionStats` resolves each cwd bucket to a
  PiCode workspace with the same `canonDir` match `handleAllSessions`
  uses, and unclaimed folders keep only their cwd.
- **Fingerprint cache, not TTL.** `session.Fingerprint(root)` is a stat
  sweep (file count, total size, newest mtime — no file opened; 0.4 ms on
  this tree versus ~0.9 s for a cold `range=all` scan). The server caches
  one `WindowStats` per range and serves it while the fingerprint and
  window boundaries match, so an appended message is visible on the very
  next request and a dashboard polling every 60 s costs nothing while
  nothing changed. This is the "natural next step" 0041 named, done
  without a staleness window.
- **UI.** Four tiles (Spend, Activity, Sessions — already in the payload,
  now shown — and Fleet as a working / waiting / idle strip fed by the
  same `workingIds`/`waitingId` the sidebar spinner uses), a daily bar
  chart with a Spend / Messages / Turns toggle and native `<title>`
  hover, then a two-column grid: spend by model, spend by workspace,
  tokens (one stacked bar in four tones of the accent + cache hit), top
  tools, reliability facts, top sessions (click → that workspace's
  sessions page). Rankings show six rows and fold the tail into one muted
  "N more" row. Auto-refresh every 60 s and a 30 s tick on "updated …",
  both paused while the tab is hidden, plus a manual refresh button.
- **Session identity, never content.** `topSessions` carries path, name,
  cwd and last-activity time. The no-preview regression test now guards
  the `"preview"` key and message text specifically, so the aggregate-
  only property of 0041 is narrowed to "no message content", on purpose.

## Consequences

- **Easier**: the levers that actually move spend (which model, which
  folder, how much of the prompt is cache) are on the home; a failing or
  aborting provider shows up as a number instead of a feeling; the
  `unknown` bucket is gone from real data.
- **Harder**: `WindowStats` is a wider contract (eleven fields) that the
  session scan, the server labelling pass and six components all agree
  on; a future field lands in three places.
- **Accepted cost**: `byWorkspace` keys on the raw cwd string, so the
  same folder reached by two spellings (symlink, trailing slash) is two
  rows until it matches a workspace — the server's `canonDir` only
  labels, it does not merge. A session file with no `session` header
  line falls back to pi's encoded folder name, shown with its `--` fence
  stripped rather than decoded (the encoding is ambiguous).
- **If wrong**: every new field is additive and the components degrade
  to their one-line empty state; dropping a section is deleting a
  `<div className="dashboard-section">`.

## Alternatives considered

| Alternative | Why not |
|---|---|
| Burn rate / monthly projection (ccusage blocks) | A projection is a judgment about a pace the product has no target for — the same reason 0041 refused to colour the delta chip. |
| Lines of code / commits / PRs (Claude Code OTel, Analytics API) | Would need diffing `edit`/`write` tool args or reading git log per session; weak signal for the cost. Named as a v3 candidate, not built. |
| p95 latency (Langfuse / Helicone) | The JSONL has no request duration; consecutive message timestamps include the user's own thinking time. |
| Per-agent context-window % on the Fleet tile (pi-agent-dashboard) | One `/status` fetch per running agent from the dashboard; refused until an aggregate endpoint exists. |
| A 30 s TTL cache (the 0041 note) | Fingerprint is cheaper to reason about and never serves stale numbers; the stat sweep is two orders of magnitude below the scan. |
| A charting library for the daily chart | Still one bar per day with a native hover; `lib/barchart.js` + `DailyChart.jsx` follow the sparkline/gitgraph split. |
