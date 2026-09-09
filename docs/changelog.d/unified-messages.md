### Added

- Agent CLIs → Messages: native Grok and Hermes communication through `picode messages contacts|send|read|ack`, sharing the existing MCP mailbox, authorization and history.
- Stored messages can notify an already open conversation through Pi's native receiver or a guarded TUI pointer. History distinguishes pending, notified, unconfirmed and acknowledged messages.

### Changed

- Grok/Hermes integration uses native hooks/plugins with ownership receipts and preserves their native executable, home and unrelated configuration.

### Fixed

- Native conversation identity no longer falls back to the most recent session for enrolled terminals; daemon restart recovery verifies the current pane/process.
- Prevent duplicate conversation addresses across `/new` and resume, stale activity reports, multiline draft submission, and attention starvation behind another recipient's backlog.
- An explicit successful Messages refresh clears stale action errors while preserving history.
- Codex lifecycle hooks remain active on resume/fork by placing their scoped overrides in the native subcommand.
