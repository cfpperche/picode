### Added
- **Packages works for the other eight agent CLIs.** Claude Code, Codex, Grok,
  Hermes Agent, OpenCode, Muse Code, Antigravity and Omp each get the Packages
  pane on their own CLI page, driven by the vendor's own plugin commands:
  installed list, install, remove, enable/disable, and the CLI's marketplace
  where it has one. Pi's pane is unchanged.
- **Each pane offers only what its CLI has.** Where a CLI exposes no verb the
  pane says so in one line instead of showing a control that does nothing —
  Codex has no plugin enable/disable, OpenCode has no plugin marketplace,
  Antigravity has no plugin update.
- **Install and remove run as jobs.** They keep running with the browser
  closed, refuse while that CLI has a live terminal (a plugin loads on the next
  start), and never replay after a PiCode restart. Antigravity, Muse Code and
  Omp got their plugin stores read here for the first time; OpenCode's plugin
  list is edited in its own config, with every other byte of the file kept.
