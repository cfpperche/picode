### Changed
- **Connectors foundation for every agent CLI (ADR-0150).** Connector capability is now a declared per-CLI driver registry, and `/api/mcp` rejects requests naming a CLI without a driver instead of silently writing Pi's files. Pi behavior is unchanged.
