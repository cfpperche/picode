# ADR-0102: Native packages live under Agent CLIs

- **Status:** accepted (owner approved the migration, 2026-09-08)
- **Date:** 2026-09-08
- **Supersedes:** ADR-0010 point 1 and ADR-0099 for navigation only
- **Extends:** ADR-0101

## Context

Packages manages Pi's native packages and agent overrides. Its top-level name
hides that identity, and the selected pane alone cannot preserve a target
through a reload or shared link. Settings already names its CLI in Agent CLIs.

## Decision

Packages is an independent Agent CLIs view at `#/clis/packages/pi`, with a
CLI selector backed by explicit package capabilities (Pi initially). Known
configuration editors live at `#/clis/packages/pi/config/<package>`.
`workspaceId`, `agentId` and the selected install `scope` travel in the query.
Canonical URLs without context mean machine packages. Legacy `#/packages*`
and mobile `#/more/packages*` replace themselves with canonical URLs, using
explicit context first and the available pane context only for legacy links.
Missing, mismatched or unsupported targets never silently choose another.

Each app owns its presentation. Context is validated independently of native
settings files, terminal inventory and installation jobs. Existing Pi APIs,
package installation, persistence and roles-layer semantics stay authoritative.
The desktop roles editor moves with Packages; mobile configuration links offer
the desktop layout until a mobile editor is implemented.

## Consequences

Package links preserve their intended target and fit the native CLI area.
Adapters remain specific to their CLI; launch support does not imply package
support. Redirects and both apps' navigation need regression coverage. There
is no database migration or new dependency.

## Alternatives considered

- Separate top-level pages with a CLI selector: viable for a future shared
  cross-CLI manager; today's package operations belong to Pi.
- One generic package backend: would invent common package semantics.
- Pane-only context: cannot identify the destination of a reloaded link.

## Amendment (2026-09-11)

Packages is a pane of the selected CLI at `#/clis/<cli>/packages`. Query
`workspaceId`, `agentId` and `scope` stay. The catalog is the CLI identity.
Old `#/clis/packages/pi*` rewrite. Persistence unchanged.

The [work plan](../plans/cli-native-packages.md) records acceptance conditions.
The Cursor/t3code benchmark adaptation is contextual, reload-safe navigation.

## Amendment 2026-09-25 — old addresses retired (owner)

The owner retired the compatibility addresses this ADR kept. `#/packages*`, mobile `#/more/packages*` and `#/clis/packages*` no longer rewrite. Canonical: `#/clis/<cli>/packages[/config/<pkg>]`. An old bookmark now lands where any unknown address does.
