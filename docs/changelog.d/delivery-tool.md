### Added
- **Delivery in the PiCode tools switch.** A CLI launched by PiCode (Claude Code, Codex, OpenCode) can be given the `delivery` tool from its launch settings, so it registers a change and asks for review in the project's **Git ▸ Delivery** view without hand-written client configuration. The tool family existed since ADR-0171; the form offered only Computer, Browser, Inbox and Checklist.
- **`packages/pi-delivery`** for a Pi agent: the same `register`, `update`, `request-review`, `withdraw-review`, `show` and `list` as a tool instead of a command, with the retry key derived from the session, the action and the payload so a repeated call replays the same declaration.

### Changed
- `picode help` and `picode mcp` list PiCode's tool families from the catalog; the help line named only computer and browser.

### Fixed
- A QA scratch instance started from inside a PiCode terminal no longer passes that session's agent identity to the terminals it opens, and keeps its sessions in its own tmux server.
