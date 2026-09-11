# ADR-0101: Native settings live under Agent CLIs

- **Status:** accepted (owner approved the plan, 2026-09-08)
- **Date:** 2026-09-08
- **Supersedes:** ADR-0012 points 2 and 3 for navigation only
- **Extends:** ADR-0069 and ADR-0079

## Context

The top-level Settings page edits Pi configuration. Agent CLIs now owns
multi-CLI launches and sessions. Native settings need the same explicit CLI
identity while retaining native files and the existing configuration scopes.

## Decision

Add Settings to Agent CLIs at `#/clis/settings/pi`. A small native-settings
capability registry initially exposes only Pi. Each application owns its
editor components; shared code owns route parsing and capability metadata.
Other CLI editors can add their own schemas and scopes later.

`#/settings` and mobile `#/more/settings` redirect by replacement, retaining
an explicit agentId or the available selected-agent context. The new address
without agentId edits global settings and keys. Composer shortcuts include
`?agentId=<id>`; `/scoped-models` also carries `focus=scoped-models`.
An invalid explicit agent or unsupported CLI never falls back to another.
The mobile conversation's quick agent settings sheet remains available.

Preferences remains PiCode configuration. Native Pi settings, keys, and agent
updates keep their existing API, persistence, trust and runtime semantics.
Settings loading is independent of terminal inventory and installation jobs.

## Consequences

Native settings have a durable multi-CLI home. Old links remain useful and
contextual links survive reload. The cost is a separate native-settings
editor registration for each future CLI; launch support alone cannot claim
native-settings support. No database migration or dependency is required.

## Alternatives considered

- Keep generic Settings top-level: hides its CLI identity.
- Force all CLIs into Pi's schema: would misrepresent their native controls.
- Move Providers, Packages and MCP too: separate work beyond this navigation.

## Amendment (2026-09-11)

Settings is a pane of the selected CLI at `#/clis/<cli>/settings`. The Agent
CLIs strip is CLIs | Messages; the catalog is the CLI identity (no CliCombo).
Old `#/clis/settings/pi` and `#/settings` rewrite. Persistence unchanged.

## References and acceptance

The [Cursor/t3code study](../benchmarks/2026-08-24-adopt-t3code-paseo-cursor.md)
informs context-preserving URLs and controls near the conversation.
The [work plan](../plans/cli-native-settings.md) records the decision table.
