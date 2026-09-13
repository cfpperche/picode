# ADR-0127: The dashboard measures the machine, and names nothing else

- **Status**: accepted (direction approved by the owner, 2026-09-12)
- **Date**: 2026-09-12
- **Boundary**: protocol — removes the `?scope=` query parameter and the
  payload's `scope` field from `GET /api/sessions/stats`, a contract
  ADR-0097 introduced; both shipped apps and the generated OpenAPI travel
  with this commit.
- **Amends**: [ADR-0097](0097-cross-cli-dashboard-metrics.md),
  [ADR-0042](0042-dashboard-v2-breakdowns.md)

## Context

ADR-0097 made the dashboard aggregate every agent CLI and added a
two-value scope: `?scope=machine|picode`, where `picode` counted only
sessions whose cwd sits under a folder a PiCode workspace claims. The
pair shipped as two filter chips — "This machine" and "PiCode" — beside
the date range.

Dogfooding the pair showed it was the maintainer's question wearing a
consumer's clothes:

- The label **collides with the product**. "PiCode" is the app, a sidebar
  group name and a workspace folder on the same screen; as a filter value
  it read as a third, unrelated thing.
- The **set it filters is dynamic, the label was static**. `claimedDirs()`
  re-reads the workspace store on every request, so adding or removing a
  workspace silently changed what the chip counted while its text never
  moved.
- It **rendered where it meant nothing**: with zero claimed workspaces
  both chips ran the identical query, and a fresh install got a control
  whose only possible effect was confusion.
- Attribution under the scope was **retroactive and lossy** (ADR-0097
  open item 2): a symlinked workspace dropped out silently, and removing
  a workspace rewrote history for that scope.

## Decision

The dashboard measures the whole machine and offers no folder scope.
`GET /api/sessions/stats` loses `?scope=` and its payload loses the
`scope` field; the browser drops the ScopePicker and the persisted
scope preference. A client that still sends `scope=picode` gets the
complete machine window — never a silently narrower answer (regression
test `TestHandleSessionStatsIgnoresRetiredScopeParam`).

Which folder belongs to which workspace remains a *label*, not a filter:
`Spend by workspace` keeps ranking every folder on the machine — claimed
folders by workspace name, the rest by their own folder name, with the
tooltip saying "outside your workspaces" rather than naming the product.
The question "what did building PiCode itself cost" stays answerable from
the same card, by reading, not by a mode.

## Consequences

- **Easier**: one aggregation, one cache identity (`root|range`), one
  answer to "what did this week cost". The retired symlink trap and the
  workspace-list cache fingerprint die with the scope. Adding a seventh
  CLI touches no scope logic.
- **Harder**: an operator with heavy out-of-workspace activity can no
  longer scope every card to managed folders in one click; they read the
  workspace card instead.
- **Accepted cost**: old clients sending `scope=picode` get a *wider*
  window than they asked for. Both shipped apps are updated in the same
  commit; nothing else reads the parameter.
- **If wrong**: the failure is an operator who genuinely needs the mode
  and finds the workspace card insufficient. The fallback is rebuilding
  the filter with trustworthy attribution first — canonicalised or
  ingested-time membership — which is the prerequisite ADR-0097's open
  item 2 never had.

## Alternatives considered

| Alternative | Why not |
|---|---|
| Keep the mode, rename the chips ("All folders" / "My workspaces") and hide them with zero workspaces | Fixes the copy, keeps the two real costs: retroactive lossy attribution promoted to a headline control, and a second cache identity to keep coherent |
| A per-workspace dropdown (Radix select) beside the range | Same attribution problem per row, plus nested-workspace ambiguity for a single-select, a third control shape in the head, and it still leaks nothing to fix for a fresh install |
| Aggregate unclaimed folders into one "Other folders" remainder row | Loses per-folder detail the card already has and was already shipped under the machine scope; no information is added |
