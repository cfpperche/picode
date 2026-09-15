# Broker

> Part of [PiCode's architecture](../architecture.md) (ADR-0105: one file per subsystem). Edit here; the index only links.

A Go MCP server (`picode-communication`, `internal/communication`, served at
`/mcp/communication`) exposes `list_contacts` / `send_message` /
`read_messages` / `ack_messages`. Sending persists a message to the peer
inbox — acceptance, not recipient execution; no tool starts an agent or
schedules work. Agents communicate through the MCP tool protocol — no
internals hacked.
