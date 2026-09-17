# MCP (Model Context Protocol) support

> Part of [PiCode's architecture](../architecture.md) (ADR-0105: one file per subsystem). Edit here; the index only links.

Pi has **no native MCP** — a deliberate design choice (tool definitions burn
context; Pi prefers CLI tools/Skills). PiCode adopts MCP through the
community **`pi-mcp-adapter`** extension (`pi install npm:pi-mcp-adapter`):

- One proxy tool (~200 tokens) instead of hundreds of definitions;
  lazy server startup; on-demand discovery.
- Reads standard configs (`.mcp.json`, `~/.config/mcp/mcp.json`,
  `~/.agents/mcp.json`) and **imports host configs** (Cursor, Claude Code,
  Codex) — a migration path for users arriving from Cursor.
- PiCode's value-add (M3–M4): a visual MCP Server Manager per workspace and
  per agent (enable/disable, precedence layers) writing the same config
  files the adapter reads. We orchestrate the ecosystem; we don't fork it.

## Connectors pane (2026-09-12, `docs/plans/connectors-ux.md`)

Every agent CLI gets this pane at `#/clis/<cli>/connectors` (ADR-0150):
one per-CLI driver behind `/api/mcp?cli=<id>` manages that CLI's native MCP
config — Pi's driver is the adapter implementation below; guest CLIs gain
drivers (`internal/connectors`, one codec per config shape) in later phases.
`web/shared/domain/integrations.js` `CONNECTOR_DRIVERS` declares each CLI's
honest capability set (status `live` vs `configured`), and a request naming a
CLI without a driver fails loudly instead of writing Pi's files. Until a
guest driver lands, `#/clis/<cli>/connectors` for other CLIs shows the
"in development" placeholder.

`#/clis/pi/connectors` is one surface in three bands: the header (the intro
line and the single primary action, **Add connector**), the configured roster,
and the connector-packages section when a package declares `pi.mcp`. `#/mcps`,
`#/integrations*` and both apps' `ConnectorsPane` render this view embedded —
there is no separate "MCPs" page.

The Add flow is one `ResponsiveDialog` (desktop) / `MobileSheet` (mobile): a
searchable service list over the catalog and the host imports
(`connectorTabs(found)`), a labelled **Save to** control, and two secondary
entries — the custom-server form (`mcpAddSchema`) and the definition-file
import, which keeps its review-and-confirm step. **Save to** is the pane's
scope: it writes `scope=user|project|agent` onto the route so a reload keeps
the target, and the add request carries the same value. Adding from a host app
keeps its confirmation.

A configured row carries the status (only while an agent runs — `Live`,
`Failed`, or the `Sign in` next action; with the agent stopped the pane says so
once instead of showing `Idle` per row), the scope tag whose title is the file
it lives in, the target, a Radix switch for enable/disable, and an overflow
menu (**Sign out**, **Remove**). Removal confirms by naming that file. Status
words come from the adapter's live report (`mcp.updated`) and the signed-in
store; the pane never claims a tool list it did not fetch.
