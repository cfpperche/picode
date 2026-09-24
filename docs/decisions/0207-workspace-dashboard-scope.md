# ADR-0207: A workspace overview scopes recorded activity by folder

- **Status**: accepted (owner, 2026-09-23)
- **Date**: 2026-09-23
- **Boundary**: protocol — adds `GET /api/workspaces/{id}/stats` with a
  workspace-specific attribution rule; amends ADR-0127 for this new route.

## Context

The machine dashboard deliberately has no workspace filter (ADR-0127). Its
workspace buckets are labels for recorded folders, not independent window
totals: filtering that response cannot produce correct time series, prior
periods or coverage. A workspace page needs those figures without changing
what the home dashboard means. The owner approved a separate page and a
separate read contract.

## Decision

The home `GET /api/sessions/stats` remains machine-wide. The new workspace
route aggregates each CLI's recorded session entries inside the meter, before
totals or series are computed. A nonempty recorded cwd belongs to the
deepest registered workspace root containing its canonical path. An exact
tie, an unknown cwd and an unclaimed path belong to no workspace. Symlinks
are resolved when possible. The response contains the period, totals, daily
or hourly series and CLI coverage; it carries no transcript content. No
workspace identity is written into a vendor session.

| Recorded cwd and registry | Action |
|---|---|
| Empty, unclaimed or outside all roots | Exclude from workspace totals. |
| Inside one root, including a symlink to it | Attribute to that workspace. |
| Inside nested registered roots | Attribute only to the deepest root. |
| Two workspace IDs resolve to the same deepest root | Exclude as ambiguous. |

`TestWorkspaceForCwd` covers every row; `TestWorkspaceScopeFiltersCurrentPriorAndSeries`
checks that the accepted predicate changes the entire window, not only its
workspace bucket.

## Consequences

One workspace's activity has a coherent numerator across totals and chart,
while the machine dashboard and its cache remain unchanged. The scope is
based on the **current** registry, so registering, removing or moving a
workspace can change historical attribution. Sessions with no folder stay
unattributed. A workspace inside another takes its own sessions rather than
duplicating them in its parent. The endpoint reparses cached meter records
for each scoped request and must be measured on large installations. If the
current-registry interpretation is wrong for operators who move workspaces,
an ingestion-time identity would need persistence and a further ADR.

## Alternatives considered

| Alternative | Why it lost |
|---|---|
| Filter `byWorkspace` in the global response | Its series, prior, tools and coverage would still describe the machine. |
| Reintroduce `scope` on the home endpoint | Reverses ADR-0127's simpler machine-wide contract and makes every card ambiguous. |
| Count every nested workspace in its parent too | Double-counts one session across workspace pages. |
