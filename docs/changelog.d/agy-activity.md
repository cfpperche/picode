### Added
- **Antigravity reports activity.** Working while it thinks, uses tools
  or starts up; Ready when idle — via a reporter PiCode installs as the
  CLI's title command (merged into your settings, removed on toggle-off;
  a foreign title command is refused, never replaced). No `hooks.json`
  decision hook ships, so nothing can gate your tools.

### Changed
- The Activity seed now runs as v2 so hookless-except-by-reporter CLIs
  are picked up on existing installs; owner-switched-off stays off.
