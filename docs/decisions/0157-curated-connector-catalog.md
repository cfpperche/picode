# ADR-0157: Curated connector catalog (Connectors marketplace)

- **Status**: accepted (owner approved the direction and the three design
  choices — sync-filtered catalog, one catalog for all CLIs, host-import
  discovery removed — 2026-09-18)
- **Date**: 2026-09-18
- **Boundary**: security model — the catalog recommends servers; credentials
  still never pass through PiCode (ADR-0075's vault refusal stands). Process
  — it amends the record of ADR-0075/ADR-0150, which registered "marketplace
  refused": what was refused was a marketplace **with hosted OAuth and a
  credential vault**; a curated catalog that writes the vendor's own config
  through the ADR-0150 drivers is now accepted.

## Context

The Connectors pane adds servers through a modal dialog: a hand-curated list
of six presets in `internal/mcp/mcp.go`, plus source tabs fed by host-config
discovery (Found) and a definition-file import. The owner wants the Packages
tab's experience — **Installed | Marketplace** subtabs with cards and search
— instead of the dialog, for one consistent way to add things. Meanwhile the
definition-import feature (file import and host-config discovery) is
redundant: the Pi adapter already imports, and since ADR-0150's drivers
manage each CLI's native config directly, "import from Claude Code into
Codex" is the same as "add the same service from the marketplace".

Web survey (2026-09-18, APIs probed live): the **official MCP Registry**
(registry.modelcontextprotocol.io, ~8,100 servers, open v0.1 API, GitHub/DNS
namespace verification) is the only large source with a free open API and
vendor-neutral data. Smithery has a registry API but its connection model is
hosted OAuth (a vault by proxy). Cline runs an in-product marketplace (UX
benchmark). Docker's MCP Catalog is verified but container/gateway-bound.
PulseMCP/Glama/mcp.so are directories without open APIs. Model context
protocol servers repo is hand-curated and small.

## Decision

The Connectors pane becomes **Installed | Marketplace** subtabs, matching
the Packages tab. The Marketplace is a **curated catalog** served by PiCode:

1. **Source**: a sync of the official MCP Registry, fetched by the server,
   filtered by curation rules (active status, remote streamable-HTTP
   endpoint or installable package), cached in the data dir and refreshed in
   the background; the seed — the hand-picked presets and PiCode's own
   `picode-*` connectors — stays pinned first. No third-party proxy ever
   carries the connection.
2. **One catalog for every CLI.** Cards carry capability badges computed
   from the target driver (transport, sign-in hint); the driver's own
   verified add path writes the entry. What is per-CLI is only the
   Installed/Added state, already scoped by the pane route.
3. **Import is removed**: `POST /api/mcp/import`, the definition-file
   reader, host-config discovery (`Report.Found`/`Report.Imports`), the
   dialog's source tabs and "Import a file…", and the guests' "arrive in a
   later phase" refusals. Custom server (the manual form) stays.

Credentials remain in each CLI's own store; the catalog writes entries, never
tokens. No OAuth flow is proxied or hosted by PiCode or anyone else.

## Consequences

Users get one Packages-like surface for all nine CLIs, with breadth (the
registry's long tail behind search) without handing over the curated top
shelf. The catalog is an opinion about other people's software: a bad card is
PiCode's fault, and the curation filter becomes a maintained artifact (kept
in the repo, re-measurable). The registry can break or drift its schema —
the cache degrades to the seed, and the pane stays useful offline. Removing
import drops the migration path for users arriving from Cursor/Claude
Desktop; the replacement is "find the same service in the marketplace".
Guest panes gain the marketplace for free because it is one surface.
ADR-0075's marketplace refusal is hereby narrowed to its security core (no
vault, no hosted OAuth); ADR-0150's "marketplace rejected" alternative
record is superseded by this ADR.

## Alternatives considered

- **Hand-curated list only** (today's six, more of them): rejected — a
  static list is not a marketplace; search needs breadth only the registry
  provides.
- **Raw registry dump without curation**: rejected — 8k uneven cards into a
  UI for terminal-averse users is a directory, not a product; trust dies on
  the first broken card.
- **Smithery/PulseMCP as source**: rejected — external dependency for the
  primary flow, and Smithery's hosted-connection model contradicts ADR-0150
  decision 2 (the vendor binary stays the authority).
- **Keep the dialog, add breadth**: rejected — two ways to add things is
  the state we are escaping; the owner chose the Packages pattern.
