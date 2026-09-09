# ADR-0103: Native providers live under Agent CLIs

- **Status:** accepted (owner approved the migration, 2026-09-08)
- **Date:** 2026-09-08
- **Extends:** ADR-0101 and ADR-0102
- **Supersedes:** ADR-0058's top-level provider route for navigation only

## Context

Providers uses Pi's catalog, active auth slot and PiCode's extra-account vault.
Its generic top-level location hides the CLI whose credentials it manages.
Settings and Packages already expose that identity in Agent CLIs.

## Decision

Providers lives at `#/clis/providers/pi`, with a native-provider capability
registry initially containing Pi. Each app owns its editor and uses its
`AgentClisFrame`. Legacy list and new-provider URLs redirect by replacement;
`#/clis/providers/pi/new` opens Add provider. OAuth returns to the canonical
list in the same desktop/mobile application. Unsupported CLI identities and
explicit scope parameters block editing instead of choosing Pi implicitly.

The current machine-wide credential semantics, native Pi APIs, account vault,
quotas and verification remain authoritative (ADRs 0013/0031/0058). Launch
support does not imply native-provider support. The llama.cpp server manager
keeps its own routes and remains accessible from its provider connection.

## Consequences

Provider navigation consistently names its CLI. Future CLIs need explicit
adapters for their native credentials and supported actions. Existing links,
login return paths and both applications require regression coverage. No
credential migration, database change or new dependency is required.

## Alternatives considered

- Keep Providers top-level: leaves Pi-specific operations without CLI identity.
- Share credentials across CLIs now: changes persistence and authentication
  semantics beyond this navigation migration.

The Cursor/t3code benchmark adaptation is contextual, reload-safe navigation.
The [work plan](../plans/cli-native-providers.md) records acceptance conditions.
