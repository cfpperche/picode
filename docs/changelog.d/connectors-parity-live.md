### Changed
- **Connectors are now verified against the real vendor CLIs.** Every agent CLI's Connectors pane — Claude Code, Codex, Omp, Antigravity, OpenCode, Grok, Muse Code, Hermes Agent — was checked live against the installed binaries, and the differences found are fixed: what you write in PiCode is what the CLI loads.

### Fixed
- **Claude Code connectors now save correctly from the pane.** The add command PiCode runs now matches what `claude mcp` accepts, and rows read from `claude mcp list` keep their address and kind.
- **Codex and Antigravity save to the one config file each CLI actually reads.** The "This folder" target refused instead of writing a file the CLI never loads.
- **Grok connectors can be turned on and off again.** The switch mirrors Grok's own enable/disable, and removing a connector no longer leaves stale entries behind.
- **OpenCode connectors land in the file OpenCode itself uses** (`opencode.jsonc` when present), so edits are never shadowed.
- **Muse Code connectors no longer produce a settings file Muse rejects** (the required `schema_version` stays present), and Grok rows report their enabled/disabled state.
