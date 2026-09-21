### Added
- **"Check for updates" for the other agent CLIs' plugins.** PiCode compares
  what each CLI has installed with what that CLI's own catalog offers and marks
  the plugins that are behind (`→ 0.3.0`), which is where **Update** appears.
  Before the check the pane offers the check itself instead of an Update button
  on every row, and a CLI whose catalog cannot be read says so rather than
  reporting that everything is current. Codex, OpenCode and Antigravity keep no
  Update action: those CLIs have no update command of their own.
