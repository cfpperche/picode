### Added

- Messages can save private setup for recorded Pi, Claude Code, Codex and OpenCode conversations and attach it when they resume. Desktop and mobile show the next action, including installing the Pi adapter when needed.
- Automatic local HTTPS setup supplies verified public CA material to the launched client while preserving existing configured CA bundles.

### Fixed

- Pi terminal resumes use the recorded conversation file, including older session pins without resume arguments.
- Communication setup preserves unrelated OpenCode inline JSON settings and MCP servers; malformed inline configuration blocks launch with an actionable error.
