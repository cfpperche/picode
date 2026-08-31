# ADR-0026: Four sidebar tabs; workspaces own terminals

- **Status**: accepted (amends ADR-0011 §5's sidebar structure and ADR-0017's "not tied to a workspace"; both otherwise stand)
- **Date**: 2026-08-30

## Context

The sidebar had three tabs (Agents, Terminals, Pins) and the Agents tab
mixed two collapsible sections: free agents and workspace groups. Three
kinds of thing, two of them sharing one tab, one collapse map holding both
real workspace ids and synthetic section keys. The owner asked for one tab
per kind — and for workspaces to become real containers: a workspace holds
its agents *and its terminals*, and a terminal can be born inside one.

ADR-0011 gave workspaces agents and hid `ws_free`; ADR-0017 made terminals
first-class and flatly "not tied to an agent". Neither contemplated a
terminal belonging to a workspace. Benchmarks: Cursor's activity bar gives
each container kind its own icon with content below (container is a tab,
not a section); t3code routes environment → thread, a two-level URL that is
"workspaces as a level". Both support one-kind-per-tab.

## Decision

The sidebar has four flat tabs: **Agents** (free agents, name-sorted, flat
— the Terminals shape), **Workspaces**, **Terminals** (free terminals
only), **Pins**. No tab duplicates another: what lives in a workspace
appears only inside its card. The Workspaces tab has no section-level
collapse; each workspace card collapses individually.

`terminals` gains `workspace_id` (default `ws_free`, migration 013, no FK
— SQLite refuses ADD COLUMN with REFERENCES plus a non-NULL default, and
the cascade is app-driven regardless). The wire stays flat:
`GET /api/terminals` carries `workspaceId` per row; grouping is client-side.
`POST /api/terminals` accepts `workspaceId` and a workspace terminal with
no cwd starts in the workspace folder. Removing a workspace kills its
terminals' tmux sessions best-effort and deletes records plus settings
overrides in one transaction; the cleanup preview counts terminals so the
dialog warns. V1 has no "move terminal between workspaces" — a terminal is
born where it lives. ADR-0017's "not tied to an agent" stands.

## Consequences

- **Easier**: each kind has a home; a project's shells live with its agents.
- **Harder**: a stored `picode-side-tab: "agents"` now shows only free
  agents — the old mixed view is gone and the change is indetectable, so
  there is no migration; the empty state's one-line action is the bridge.
  Four 32px tabs eat the header: the brand version yields below 254px.
- **Accepted cost**: if a tmux kill fails after the workspace DELETE, the
  `picode-sh-*` session is orphaned — recoverable from the tmux catalog
  (ADR-0025). A failed cleanup preview still allows unregistering, without
  the terminal count.
- **If wrong**: the tab split is presentational and reversible; the
  `workspace_id` column collapses back by defaulting everything to
  `ws_free`.

## Alternatives considered

| Alternative | Why not |
|---|---|
| Keep sections, add terminals to groups | The mixed tab was the complaint |
| Embed `terminals[]` in `workspaceView` | N extra tmux calls per poll; the app already holds one flat terminals list for `t:<id>` tabs |
| FK with table recreation | First create/copy/drop/rename migration in the repo for integrity the app enforces anyway |
| Terminals return to free on workspace remove | Owner chose: they die with it, like agents, with a warning |
| Move terminal between workspaces in V1 | Owner deferred; create-inside covers the need |
