### Fixed

- Prevent OpenCode from hanging when communication reconnects its native conversation; validate resumed readiness without overriding newer activity or permission requests.
- Connect identified Codex conversations without restarting, and preserve their native identity when older launchers emit auxiliary completion notifications.
- Preserve terminal dimensions during communication setup and recognize Claude Code's compact footer without requiring the browser panel to resize.
