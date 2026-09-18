### Added

- Connectors (MCP) management for Claude Code and Codex (ADR-0150 phase 1): `#/clis/claude|codex/connectors` manages each CLI's native MCP config — Claude Code's workspace `.mcp.json` plus its vendor CLI for user scope, Codex's `config.toml` with comment-preserving surgical edits — behind the same pane and API shape as Pi, with honest "configured" status and per-CLI toggle capabilities.
