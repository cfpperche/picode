# ADR-0066: Project groups inside the Docker App

- **Status**: accepted (owner requested the grouped view on 2026-09-04)
- **Date**: 2026-09-04
- **Extends**: ADR-0036, ADR-0065

## Context

The flat inventory repeats project metadata and makes a multi-project Docker
host difficult to scan. The owner's VS Code reference groups containers by
project; the owner explicitly wants that organization inside the Docker App.
The [Cursor/t3code/paseo benchmark](../benchmarks/2026-08-24-adopt-t3code-paseo-cursor.md)
informs the adaptation: compact disclosure rows, visible state, native controls.

## Decision

Group inventory by the exact `com.docker.compose.project` label already read
by the Docker client. Do not infer membership from container-name prefixes.
Sort project names and container names; unlabeled containers form a final
"Standalone containers" group. Summaries report actual container states;
created/exited count as stopped, while paused/restarting remain distinct.

Reuse list blocks with optional `id` and `collapsible` fields, retaining the
four block types and API version 1. Stable identities contain the complete
Docker endpoint/project pair. The host renders native `details`/`summary`,
with animated disclosure and saved per-app/per-group browser preferences.
New groups start closed. No new dependency is required.

Search matches a group's name or its child rows, hides empty matches and opens
matching groups. Users may fold search results; those temporary folds reset
when the query changes. Clearing search restores the saved normal view.
Refresh, navigation and row reordering preserve each group's preference.
Existing per-container operations and confirmation behavior remain in force.
On phones, long group/container names and image metadata wrap within the
available width so a narrow screen does not erase their identity.

## Decision table

| Conditions | Result | Coverage |
|---|---|---|
| Same project label, arbitrary names | One group | Go grouping tests |
| Similar name prefixes, different labels | Separate groups | Go grouping tests |
| No project label | Standalone group, last | Go grouping tests |
| Engine has no containers / is inaccessible | Existing empty / blocked state | App tests + visual QA |
| Mixed running, created, exited, paused, restarting | Counts reflect each state, without zero badges | Go summary test |
| Active job on the exact endpoint/container | Row and group announce pending work | Go busy matrix |
| Refreshed/reordered rows or another endpoint | Preserve identity / isolate preferences | Go identity + JS persistence tests |
| Project-name match / child-only match / no match | Whole group / matching rows / clear-search state | JS filter tests + visual QA |
| First visit / stored preference | Closed / restore saved fold | JS preference tests + visual QA |
| Search begins, folds, changes, clears | Temporary folds; normal preference preserved | JS search-state test |
| Storage malformed or denied | Folding still works in memory | JS storage test |
| Mouse or keyboard activates the heading | Native disclosure, usable focus | Browser QA |
| Long project/container/image names on a phone | Wrap without horizontal overflow | Visual QA |

## Consequences

The inventory becomes scannable without moving any Docker content into the
sidebar or mutating Docker resources. A detected group is an observed set of
containers, not proof that a usable Compose file is available. Registration,
project operations and deployment are specified separately in the
[Docker v2 proposal](../plans/docker-v2.md).

Source: [Docker Compose labels](https://docs.docker.com/reference/compose-file/services/#labels).
