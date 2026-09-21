### Fixed
- **Check for updates stays when a CLI has one plugin, or none.** The control
  lived inside the filter bar, which only renders for two or more installed
  plugins, so Claude Code (one plugin) and an empty list had no way to run the
  check the pane itself says is how Update appears.
- **A group count is a number, not a toolbar pill.** The header reused the
  filter's bordered count chip, so "59" read as a stray button next to
  "Ships with the CLI".
