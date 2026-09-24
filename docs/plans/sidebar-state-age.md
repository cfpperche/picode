# Sidebar state age — "Ready · 5m": how long the agent has been in its state

Owner request (2026-09-23): the sidebar pill should say how long an agent has
been in its current state — idle, ready, open — e.g. "Ready now", "5m ago".
Today only `working` shows an age. Status: **approved same day; shipped**
(see the addendum for what landed and the two deliberate deviations).

## Principle

An age is shown **only** when a truthful stamp for *entering the current
state* exists. No invented clocks: `createdAt` is never a state age
("created 3d ago" is not "ready 3d"). Where no stamp exists the pill renders
the label alone (ADR-0092: absence is not worth a line). This keeps
benchmarks.md's rule the terminal state machine already cites: status is
always truthful.

## Data audit — what already exists

The server stamps the instant the current state was entered; the gap is only
where the client looks.

| Status | Terminal-backed CLI agent (hook state, ADR-0056) | Managed agent (rpc) |
|---|---|---|
| needs-you | `term.stateAt` — hook reported needs-you | ask/inbox timestamp (verify field in the ask payload; follow-up if absent) |
| working | `term.stateAt` | `lastStartedAt` (approximate — today's behavior, unchanged) |
| ready (idle) | `term.stateAt` — hook reported idle | **missing**: `lastStatusAt` is only written on start/stop |
| open | `tui.startedAt` fallback (no `stateAt`) | interactive rows: no stamp → no age |
| stopped | `lastStatusAt` (stop write exists) | `lastStatusAt` (stop write exists) |

Facts that make the terminal column free:

- `TermStates.SetForRun` (`internal/server/term_state.go`) rewrites `At`
  only when the state tuple actually changes — `stateAt` **is** the
  transition instant, and it travels in `GET /api/terminals` and the
  `terminal.state` feed event.
- `Sweep` deletes an expired `working` entry (30 min TTL) → the row
  degrades to "Open" with the `tui.startedAt` stamp. Honest ("open since
  the TUI started"); no new server semantics, no invented "idle".

## Design

| Area | Design |
|---|---|
| Stamp resolution | New pure helper `agentStatusStamp(status, ag, term)` in `web/shared/domain/agentStatus.js`: per-status source (table above), never `createdAt`, returns `""` when nothing truthful exists. Replaces `AgentRow`'s single fallback chain, which today can show a stale pre-stop `stateAt` as a stopped age |
| Pill rendering | `AgentStatus` / `TerminalStatus` in `WorkspaceRows.jsx`: drop the `status === "working"` gate, render `age = relTime(stamp)` for every status. Copy stays compact — `Ready · now`, `Ready · 5m`, `Open · 2h`, `Stopped · 3d` — existing `.ws-status-age` style (tabular-nums, 0.75 opacity), `title={absTime(stamp)}` on hover. No "ago" suffix in the pill (width); the word exists in wide surfaces (Outcomes' `ago()`) and stays there |
| Ticker | `useNow(30_000)` in `web/browser/src/lib/useNow.js`: one `setInterval` + `setState`, paused on `document.hidden`, used by `Sidebar.jsx` so ages tick without feed traffic. Display-only — the ADR-0048 no-polling rule is about `/api/*`, untouched. Dashboard already ticks (`TICK_MS`); canvas chip host gets the same hook |
| Managed "went idle at" (the one server change) | At managed turn settle (`internal/server/rpc/runtime.go`, the turn-completion path that today only clears streaming), write a new `store.SetAgentTurnSettled(id)`: `last_status_at = now`, `last_status` stays `running`, announces `agent.status`. The event payload gains `lastStatusAt` |
| Feed patch | `web/shared/domain/feedReducers.js` `agent.status` case applies `lastStatusAt` (today it drops timestamps). Reducer tests extended |
| Surfaces (ADR-0062: same words, derived the same way) | `WorkspaceRows.jsx` (`AgentStatus`, `TerminalStatus` — sidebar + dashboard fleet), canvas `PanelBody.jsx` and `CanvasSurface.jsx` chips (same working-only gate today), mobile `AgentRow.jsx` (same gate, `" · " + relTime`) |
| Ordering | Untouched: `bucketAgentsByState` moves rows only when state changes, "never when a timestamp ticks" — ages render inside rows |

### Benchmark adaptation

Geist/Sentry short relative units (already relTime.js's documented
convention) applied to the status pill; benchmarks.md's "status is always
truthful" governs the no-stamp-no-age rule. Linear's deference: the age is
muted metadata at the row's end, never a second status.

## Decision table (implementation must cover every row)

| Conditions | Observable result |
|---|---|
| Terminal agent hook-reported idle 5m ago | `Ready · 5m`, ticks to `6m` within 30s of the minute |
| Terminal agent freshly idle (<1 min) | `Ready · now` |
| Managed agent settles a turn | pill flips `Working` → `Ready · now` without refetch (feed patch) |
| Managed agent never started (`lastStatusAt` nil) | `Stopped`, no age |
| Terminal agent stopped | `Stopped · <lastStatusAt age>`, not the stale pre-stop stateAt |
| Plain shell terminal | `Terminal open` (+ age since TUI start when a tui exists; bare label otherwise) |
| Working decays past TTL (sweep) | `Open · <tui age>` — degrades honestly, no stale spinner |
| No truthful stamp for the state | label only, no age, no placeholder |
| Tab hidden 1h, then visible | ages correct immediately (tick pauses/resumes; relTime at render) |
| Row reorder | driven by state change only; a ticking age never moves a row |

## Tests

- `web/shared/domain/agentStatus.test.js`: `agentStatusStamp` per status ×
  stamp availability, incl. "never createdAt", stopped prefers
  `lastStatusAt` over stale `stateAt`, empty when nothing truthful.
- `internal/store`: `SetAgentTurnSettled` moves `last_status_at`, keeps
  `last_status = running`, appends its event (events-invariant suite).
- `web/shared/domain/feedReducers.test.js`: `agent.status` with
  `lastStatusAt` patches the row in place; without it, unchanged behavior.

## Acceptance

- `make web` + scoped node tests green; `make ci-scoped` before close.
- Browser QA on a scratch instance (`scripts/qa-scratch.sh`): seed one
  terminal agent idle >1 min, one freshly settled managed agent, one plain
  shell, one stopped; screenshot the sidebar (light+dark), confirm ages,
  the 30s tick, the feed-driven `Working → Ready · now` flip without
  refetch, hover absTime. Screenshots read in a subagent; visual-review
  verdict in the handoff.

## Out of scope / follow-ups

- Managed `working` age stays approximate (`lastStartedAt`); exact per-turn
  start would ride `agent.state` events with an `at` field — separate change.
- Managed `needs-you` age if the ask payload lacks a timestamp.
- `interactive` legacy rows (no state machine → no stamp → label only).

## Addendum: shipped (2026-09-23, same day)

Landed as designed, with the owner's three recommendations taken: compact
ages (`Ready · 5m`, no "ago"), managed `working` still approximate via
`lastStartedAt`, and the sweep decay untouched (a TTL-expired `working`
degrades to "Open" on the TUI's own stamp — no invented idle).

Two deviations from the letter of the plan, both deliberate:

- **Canvas got no ticker.** The widened gate in `PanelBody` shows the age
  only in the transient loading placeholder, and canvas models are memoized
  through `sameModel` — a 30s tick would invalidate every model each beat
  for an age nobody stares at. The chips' stamps themselves are now
  per-status (`agentStatusStamp`), so a placeholder that does render shows a
  truthful age.
- **The `agent.settled` reducer patch also sets `streaming: false`.** The
  ephemeral state notice can be dropped for a slow consumer
  (`Broadcast`'s `default: drop`); the settle implies not streaming, so the
  pill flip stays atomic even when the edge notice was lost.

Verified: `make ci-scoped` green (fmt, vet, hooks, 22 Go packages, test-js,
build, living-docs); store unit test proves the settle stamp moves with
RFC3339Nano determinism and the invariant suites cover the new
`agent.settled` mutation; domain tests cover `agentStatusStamp` (never
`createdAt`, stopped never reads the pre-stop terminal state) and the
reducer patch. Live on a scratch daemon (`scripts/qa-scratch.sh`, seeded
workspace + three agents + three terminals): a managed Pi agent's pill ran
`Stopped → Ready now → ⠋ Working → Ready now` across a real turn with no
page reload (in-page marker survived), the flip riding `agent.settled`; a
hook-driven terminal ran `Ready 3m → Ready 4m → Ready 8m` on the 30s tick
with no feed traffic; `Needs you now` flipped live from a state report;
`Terminal open` and never-started rows show no age; mobile's Work list
shows `Idle · 5m` from the same shared reducer. `overlayAudit ok`.

The visual review caught two real gaps, both fixed in-session:

- **A fresh managed start showed no age** — `agent.status` did not carry
  `lastStatusAt`, so a start stamped the store but not the open rows; the
  pill stayed bare "Ready" until the first turn settled. The start/stop
  payload now carries `lastStatusAt` and the reducer applies it (an event
  without one keeps the row's own). Verified live: stop → start flips the
  pill to "Ready · now" with no turn and no reload.
- **Mobile dropped the age on non-working terminal chips** — mobile's
  `TermRow` kept its own working-only gate (now widened), and the `.m-state`
  cap of 92px clipped "Working · now" to "Working · n…" (now 118px, sized
  for the widest short-unit chip "Needs you · 59m").

Final verdict from the screenshot reader: `visual-review: PASS` — every age
token complete inside its pill, no truncation, card 5/5.
