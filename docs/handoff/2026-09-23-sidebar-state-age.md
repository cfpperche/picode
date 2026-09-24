# 2026-09-23 — sidebar-state-age: status pills say how long the agent has been in its state

Shipped: every sidebar/canvas/mobile status pill shows its state's age —
`Ready · now`, `Ready · 5m`, `Open · 2h`, `Stopped · 3d` — resolved per
status by `agentStatusStamp` (web/shared/domain/agentStatus.js): the
terminal hook's `stateAt` for CLI agents, the store's writes for managed
ones. New durable `agent.settled` event (`store.SetAgentTurnSettled`,
called from rpc pumpEvents at turn end) and `lastStatusAt` riding
`agent.status` patch rows live without a refetch; display-only 30s tick
(`web/browser/src/lib/useNow.js`) advances ages between events. Where no
truthful stamp exists the pill shows the label alone — `createdAt` is
never a state age. Plan + shipped addendum: docs/plans/sidebar-state-age.md.
Verified: `make ci-scoped` green twice (before/after review fixes); store
unit test proves the settle stamp moves; invariant suites cover the new
mutation; domain tests cover the stamp resolver and both reducer cases.
Live on a scratch daemon: a managed Pi agent ran Stopped → Ready now →
⠋ Working → Ready now across a real turn with no reload (in-page marker);
a hook terminal ticked Ready 3m → 4m → 8m with no feed traffic;
Needs-you flip live; mobile chips complete. Blind spot: not exercised
against a real multi-day-old fleet row (ages >1w render an absolute date
via relTime — unit-covered only).
visual-review: PASS (5 shots read by subagent; card 5/5 after fixes)
Not done / debts: none new; follow-ups live in the plan's Out of scope.
Merge: fast-forward ready.

## Debts

- Managed `working` age stays approximate (`lastStartedAt`) and managed
  needs-you has no timestamp on the fleet row — Plan: docs/plans/sidebar-state-age.md (Out of scope / follow-ups)
