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
