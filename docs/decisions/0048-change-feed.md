# ADR-0048: The change feed — every mutation is a durable event, served over SSE with replay

- **Status**: accepted (amends 0037 on delivery and 0047 on hooks)
- **Date**: 2026-09-01

## Context

Every surface learned about state by polling. An inventory found nine
desktop timers (apps and badges 15 s, tui-working 3 s, devices 4 s,
automations 15 s, status 15 s, dashboard 60 s, packages 30 min, health
2.5 s, presence 15 s), four mobile ones (`useFleet` at 5 s on the Now and
Work screens with three requests per tick, inbox and badges 15 s, stats
60 s, tui-working 3 s) and twenty-two hand-written `loadWorkspaces()`
calls after local mutations in the desktop shell. The desktop never
learned about a workspace, agent or terminal created from another device
or by an automation until a local action or a reload.

ADR-0037 chose the poll cycle for the Inbox ("the server has no SSE …
SSE remains a global v2"). ADR-0047 then needed to know about new inbox
items and waiting agents and added two single-purpose callbacks
(`Store.OnInboxCreated`, `Runtime.OnWaiting`) because there was nothing
else to consume. Meanwhile the `events` table (001) already had a
monotonic `id`, a `type`, ids and JSON data — but it was write-only,
covered a third of the mutations, and nothing read it.

The owner's direction was explicit: the correct design, not the smallest
patch. An invalidation-only stream ("something changed, go fetch") would
have been fewer lines and would have left the 22 refetch sites, the
callbacks and the write-only table in place; it is the weaker invariant
and the one that is hard to upgrade later.

## Decision

1. **Every mutation appends an event in the same transaction.**
   `AppendEvent` / `AppendEventTx` write a typed row
   (`entity.action`: `workspace`, `agent` incl. `agent.status`,
   `terminal`, `inbox`, `task`, `automation`, `run`, `pin`, `setting`,
   `push`, `terminal_settings`) whose `data` is the entity's JSON view or
   `{id}` on delete. Events appended inside a transaction are announced
   only when the store commits it (`store.commit` flushes, `rollback`
   drops), so a listener never sees an uncommitted change. Ids that are
   foreign keys (`agent_id`, `workspace_id`) are set only for real agents
   and workspaces; inbox, automation and terminal events carry their own
   ids in the payload. `TestEveryMutationAppendsAnEvent` enumerates the
   mutating methods and their events; a new mutator without a row there
   is the review signal.
2. **`internal/feed` fans out and replays.** `Store.OnEvent = feed.Publish`.
   `Subscribe(afterID)` reads `ListEventsSince` under the same lock that
   registers the live channel, so nothing falls between replay and live;
   a cursor older than the retained log returns `ErrReset`. Events older
   than seven days are pruned by the existing sweep. Ephemeral notices
   (`device.online` from presence, `agent.state` on every streaming /
   dialog edge of the runtime, `agent.waiting`) ride the same stream with
   id 0 and are never replayed.
3. **`GET /api/events` is server-sent events**, stdlib only: `retry`,
   `hello {bootId, latest}`, `change {Event}` with `id:` for durable rows,
   `reset` when the cursor cannot be honoured, a 25 s heartbeat. The
   cursor is `Last-Event-ID` (browser reconnects) or `?after=` (a fresh
   page resuming from `sessionStorage`). 503 when no feed is wired.
4. **Push consumes the feed.** `Notifier.OnEvent` handles `inbox.created`,
   a superseded result (`inbox.updated` still unread) and `agent.waiting`.
   `Store.OnInboxCreated` is gone.
5. **Clients patch where a reducer is faithful and refetch where it is
   not.** `lib/feed.js` keeps one `EventSource` per shell, persists the
   cursor, kicks the health watch when `hello.bootId` changes (a new
   binary → reload) and emits `feed.open / feed.down / feed.reset`.
   `lib/feedReducers.js` applies fleet, inbox, automation and run events
   and returns `null` when the honest answer is a refetch (an agent
   starting — the store cannot say managed vs interactive; an entity the
   list has never seen). Timers stay in place as the fallback and skip
   their tick while the feed is connected; `feed.open` after a drop and
   `feed.reset` refresh every list once.

Decision table (each row tested):

| condition | outcome |
|---|---|
| mutation inside a tx, tx rolls back | event dropped, nothing announced |
| mutation inside a tx, tx commits | events announced after commit, in order |
| subscriber slow (channel full) | event dropped for it; its next reconnect replays from its cursor |
| reconnect with cursor ≤ latest, ≥ oldest−1 | replay `id > cursor`, then live |
| reconnect with cursor < oldest−1 | `reset` then live; client refetches all |
| `hello.bootId` ≠ stored | cursor cleared, health watch kicked → reload |
| feed connected | apps / automations / devices / mobile fleet+inbox timers skip |
| feed down | timers run at their old cadence |
| reducer cannot apply | that list refetches; others untouched |
| presence ping from a stale or new device | ephemeral `device.online` |

## Consequences

- **One truth, one history.** The store is the truth; the events table is
  its history and the only announcement channel. Push, the desktop, the
  phone and any future consumer (outbound webhooks) read the same feed.
- **The log grows.** Bounded by seven-day retention; a phone away longer
  than that gets a `reset`, which is one full refresh.
- **Ephemeral vs durable is explicit.** Presence and live run state are
  not facts the store holds, so they are not replayed; the fleet view
  reconciles them on `feed.open` through a refetch.
- **The phone stops polling.** Verified on a scratch instance: with the
  feed connected the only periodic requests are the health ping, the
  presence ping and `tui-working` (tmux agents still have no channel).
- **Kept:** the dashboard's 60 s cache-backed poll (an aggregate over
  session files, not a list) and the health watch (the stream's own
  watchdog, idling at 20 s while the stream is up).
- **Closed the same day (amendment):** a server-side tmux watcher
  publishes `agent.tui` (one scrape per tick for the fleet instead of one
  per browser); `agent.usage` per assistant message lets the status bar
  add tokens and cost up instead of rescanning the session file;
  presence turns silence into `device.offline` on a 5 s ticker;
  `agent.status` carries the run mode (managed / interactive) at every
  start site so a start patches without a refetch; fifteen post-mutation
  `loadWorkspaces()` calls became a refetch only while the feed is down.
  Still refetched on purpose: workspace create / clone and the agent
  config PATCH, because `git` decoration is view-only and not in any
  event.
- **`PICODE_INSECURE` (HTTP/1.1) budgets six connections per origin;** one
  SSE stream plus one agent socket fits. TLS is HTTP/2.
- **Any mutation that bypasses the store is now a bug**, not a style
  choice: the server layer writes through store methods so the event is
  in the transaction.

## Alternatives considered

- **Invalidation hints only** (server says "inbox changed", client
  refetches). Fewer lines, no replay, but it keeps the ad-hoc callbacks
  and refetch sites, makes the events table optional again, and cannot
  serve a backgrounded phone or a future outbound integration. Refused
  as the weaker invariant.
- **WebSocket global.** Bidirectional for no reason; SSE reconnects and
  resends `Last-Event-ID` natively and passes proxies.
- **Long polling.** A worse SSE.
- **A sync engine (Replicache / Electric-style).** Wrong scale for a
  single-operator local daemon with a dozen list shapes.
- **Rich events without a durable log.** Divergence on the first dropped
  frame; replay is what makes patching safe.
