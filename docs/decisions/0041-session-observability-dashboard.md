# ADR-0041: Session observability dashboard replaces the no-tabs-open home

- **Status**: accepted, supersedes the unmerged `feat/home-dashboard`
  attempt (its own `0039-home-dashboard.md` never reached `main`, so there
  is no live reader to redirect — recorded here for the trail, not as a
  formal supersession).
- **Date**: 2026-09-01

## Context

The main pane, with no tab open, used to show "No agents yet — add a
project folder" whenever `tabs.length === 0`, regardless of whether
workspaces/agents/terminals actually existed. A first fix (`feat/home-
dashboard`) replaced that with a `HomeView`: one card per workspace
listing its agents/terminals — essentially the sidebar, reformatted. The
owner rejected it: "ux fraca, ocupando parte da tela, sem sentido visto
que a sidebar já está disponível... não quero transformar essa view em um
acesso rápido apenas." The direction asked for instead: an analytics/
observability surface — spend, activity, fleet health — not another
navigation list.

Two existing decisions bear directly on this and needed reconciling
before building anything:

- `docs/design/session-surface-roadmap.md`'s Refuse table: **"Cost as a
  new page | it belongs on the session chip the user already clicks"**
  (shipped as D2, 2026-08-27). That refusal is about *one session's* cost
  duplicating the chip. This dashboard shows a fleet-wide aggregate —
  total spend, a trend, a breakdown by provider — that no chip or single
  session row can answer; nothing here duplicates D2.
- `docs/design/diff-editor-roadmap.md` / `conversation-control-roadmap.md`
  use **"X is not the home"** as a standing refusal against other surfaces
  annexing the product's identity ("Explorer as the product... it is not
  the home"; "IDE chrome (LSP, explorer-as-home)... still refused"). Both
  of those refused a surface that would sit *beside or over* the live
  agent conversation. This dashboard only ever renders when
  `noTabs && hasData` — by construction, nothing is open, so nothing is
  being displaced. The owner confirmed this reading explicitly before any
  code was written.

No existing benchmark study (`docs/benchmarks/`: Cursor, t3code, paseo,
herdr) covers a dashboard/analytics surface — see the companion study
`docs/benchmarks/2026-09-01-llm-observability-dashboards.md`, added per
`docs/benchmarks/README.md`'s own ritual for exactly this gap.

## Decision

The `showHome` gate (`hasData = workspaces+freeAgents+terminals > 0`,
`showHome = noTabs && hasData`) is unchanged from the rejected attempt —
it was already correct. What mounts there changes:

- Three stat tiles — **Spend** (period total, sparkline, delta vs. an
  equal-length prior period), **Activity** (messages, same treatment),
  **Fleet** (agents running now vs. total — a live count from data
  already in memory, no fetch, no sparkline).
- **Spend by provider** — one ranked, single-hue bar list (the provider
  name is already the identity label; no categorical palette needed).
- One shared date-range control (Today / 7 days / 30 days / All time),
  persisted per-viewer in `localStorage` (`lib/openTabs.js`), not the hash
  router — this app's `#` routes name *what object is open*, never a view
  filter (see `docs/architecture.md`).

New backend surface: `GET /api/sessions/stats?range=...`
(`internal/session/stats.go`, `internal/server/session_stats.go`). It
scans session JSONL at **message** granularity (reusing the already-
unexported `entryTS`/`costFrom` from `transcript.go`/`session.go`) rather
than bucketing by `Summary.UpdatedAt` (file mtime) — a session reopened
across several days would otherwise dump its whole cumulative cost onto
whichever day it was last touched, a fabricated spike. Day buckets use
the server's local timezone (this is a single-machine tool; every other
"what day was this" surface in the frontend already renders in browser-
local time). The response carries only aggregates — no `Preview`, no raw
session rows — both because the dashboard has no use for per-session
content and because shipping every session's message preview to the
browser just to sum a few numbers would be wasteful and needless
exposure. A `range=all` request has no cheap pre-filter (everything is in
the window by definition) and no caching in v1 — see Consequences.

No charting dependency was added. `web/src/lib/sparkline.js` +
`StatTile.jsx` mirror the existing `lib/gitgraph.js` + `GitGraph.jsx`
split (pure geometry module, thin SVG-rendering component); the bar list
reuses the CSS meter-bar pattern already shipped in `UsageDialog.jsx`.

## Consequences

- **Easier**: the home slot is honest about the environment instead of
  reciting a first-run pitch beside a full sidebar; the two prior
  refusals are reconciled in writing instead of quietly ignored.
- **Harder**: a second Go aggregation surface over the sessions tree to
  maintain, alongside `handleAllSessions`/`Summarize`.
- **Accepted cost**: `range=all` re-scans every session file on every
  request — no cache, matching `session.ListAll()`'s own existing
  precedent. If this becomes slow on a machine with years of history and
  cleanup left off (the default), an in-memory cache with a short TTL is
  the natural next step — not built now. No cross-mode "needs attention"
  signal ships in v1 (same reasoning the rejected attempt already gave:
  no signal today spans interactive + managed agents); a git-dirty-branch
  tile is the obvious v1.1 addition, using data already fetched.
- **If wrong**: reversible the same way the rejected attempt was —
  `showHome`'s mount point renders nothing in its place, and the true
  blank slate is untouched beside it.

## Alternatives considered

| Alternative | Why not |
|---|---|
| Keep the workspace/agent list (`feat/home-dashboard`) | Rejected outright — reformats data already visible in the sidebar, earns nothing. |
| Spend as its own dedicated route (e.g. `#/stats`) | The owner chose the home slot specifically, having weighed the "X is not the home" precedent; a separate route was the fallback if that precedent held, and it didn't. |
| Aggregate client-side over the existing `/api/sessions/all` | Ships every session's `Preview` and raw rows to the browser just to sum a handful of numbers, and re-scans the whole (uncached, unbounded) sessions tree on every dashboard load rather than once server-side. |
| Bucket by `Summary.UpdatedAt` (file mtime) instead of a new message-level scan | Wrong by construction for any multi-day session — see Context/Decision above. |
| Color-code the delta chip green/red by direction | No defined "good" direction for spend or activity in this product — there is no budget concept anywhere in PiCode. Coloring one here would fabricate a judgment the app doesn't otherwise make. |
