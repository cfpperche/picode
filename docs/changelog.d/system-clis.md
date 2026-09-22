### Changed
- **The System page lists tmux as the only requirement** (mkcert and tailscale stay optional) and gains an **Agent CLIs** section with one row per CLI linking to its page; the `pi` row and the "install pi with npm" warning are gone (ADR-0179).
- **Automations report "Pi is not installed" only when a run actually needs Pi** — a start, or a message to a Pi agent. A message to another CLI's agent delivers through that agent's terminal.

### Removed
- The Pi update card on the System page and `POST /api/system/pi-update`; Pi updates from Agent CLIs like every other CLI (ADR-0087).
