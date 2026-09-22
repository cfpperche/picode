### Added
- **Backups cover every agent CLI.** A snapshot now keeps each CLI's settings file; with **Include secrets**, its login file; and with **Include sessions**, the conversations of Claude Code, Codex, Grok, Omp and Muse Code as well as Pi's. Restore puts them back. OpenCode, Hermes Agent and Antigravity keep conversations in a live database, which is not copied.

### Fixed
- **Shared-server members can install CLIs from Agent CLIs.** Their container now has Node.js 22 and installs into each member's own `~/.local`; before, the distro's old Node and a root-owned npm made Install fail. Existing containers are updated on the next `picode provision`.
