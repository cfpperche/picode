# Change feed (ADR-0048)

> Part of [PiCode's architecture](../architecture.md) (ADR-0105: one file per subsystem). Edit here; the index only links.

Every store mutation appends a typed row to `events` in its own
transaction (`entity.action`, data = the entity's JSON view); the store
announces committed rows through `Store.OnEvent`, which `internal/feed`
fans out to `GET /api/events` subscribers (SSE: `hello` with the bootId,
`change` frames with `id:` for durable rows, `reset` when a cursor is
older than the seven-day retention) and to in-process listeners (the
push notifier). Ephemeral notices — `device.online` from presence,
`agent.state` on every streaming / dialog edge, and `agent.waiting` — ride
the same stream with id 0. Clients (`web/shared/client/feed.js`) keep one
`EventSource` per shell, resume from a `sessionStorage` cursor, patch
lists with `lib/feedReducers.js` and refetch when a reducer returns
`null`; the old timers only tick while the feed is down. The server
also publishes `agent.tui` (tmux watcher, `StartTuiWatch`),
`agent.usage` (per assistant message, `Runtime.OnUsage`) and
`device.offline` (`presence.Watch`); `agent.status` carries the run mode
at every start. Presence invokes its transition callback after releasing the
registry lock but before the heartbeat returns, so a sequential expiry cannot
overtake a detached `online` callback. Rule: a state
change that is not in `events` did not happen — write through the
store, never around it.
