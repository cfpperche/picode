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

## How the rule is enforced

"Every mutation announces itself" is checked two ways, because the first
one alone was a checklist. `TestEveryMutationAppendsAnEvent` exercises 118
hand-written cases and proves each one emits what it should — but a new
mutator that forgets its event is simply not on the list, and nothing
notices.

`TestEveryExportedMutationAnnouncesOrIsListed`
(`internal/store/mutation_coverage_test.go`) is the invariant itself: it
parses the package, finds every exported `*Store` method that writes to a
table, and requires each one to append an event — or to appear in
`silentMutators` **with a reason**. An entry without a reason fails, so "I
could not think of the event" cannot pass as a decision, and a companion
test removes a name that stopped being a mutator.

**Both signals follow calls on the receiver**, and the first version of
this test followed only one of them. Scanning a method's own body for SQL
missed every exported mutator that delegates the write — `AddAgent` through
`AddAgentWithCLI`, `CreateTerminal`, `EnablePeer`, `ReplaceFrom` and
thirteen more: seventeen writes waved through because the literal lived one
call away. Fourteen of them did announce, so the gate was right by luck
rather than by construction. Bodies are also read with line comments
stripped, since "we deliberately do not AppendEvent here" must not be the
thing that satisfies the check.

The sixteen deliberate exceptions are the event log's own machinery
(`AppendEvent` and `AppendEventTx` *are* announcing; pruning would refill
what it emptied), a restore's whole-database swap, auth last-seen and
expiry housekeeping, Web Push delivery marks, the extension's actuation
batches (ADR-0053/0054 — the panel polls for its own batch, so the feed has
no subscriber for it) and the ADR-0039 session-identity bookkeeping, whose
visible change is carried by `agent.updated`.
