### Added
- **A refusal that needs you comes with the command to run.** When a CLI turns
  a plugin action down because only a person in a terminal can answer it — Grok
  asks you to trust the source, a marketplace can ask for a command to be
  confirmed — the pane now shows the CLI's own words *and* the exact command,
  with **Copy command** and "Run it in a terminal." It works for the actions
  that answer immediately and for a background install or removal that failed.

### Fixed
- **Omp shows what its marketplaces offer.** Its plugin catalog was reported as
  unavailable; PiCode now reads `omp plugin discover` and lists the plugins a
  source offers, each marked installed when it already is. Omp's list does not
  say which source provides a plugin, so those rows are information and the
  list states the install form (`name@marketplace`) instead of offering a
  button that would run the wrong command.
- **OpenCode's plugin list says when a config is broken.** A `"plugin": "name"`
  string is refused by OpenCode itself ("Expected array | undefined"); PiCode
  used to list it as an installed plugin. The pane now names the file and the
  form OpenCode expects.
