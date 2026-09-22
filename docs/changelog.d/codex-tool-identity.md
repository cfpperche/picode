### Fixed
- **PiCode's tools work for a Codex agent.** Codex starts an MCP server with an empty environment, so a Codex agent's tool calls answered `no identity` — Delivery, Browser, Computer, Inbox and Checklist were all unreachable from one. The launch now writes the identity into each server's own config (`mcp_servers.<name>.env.…`); Claude Code and OpenCode inherit the launch environment and needed nothing.
- **PiCode · Delivery appears in the connector catalog**, so an agent whose CLI takes tools from its own configuration can add the Delivery tool like the other four.
