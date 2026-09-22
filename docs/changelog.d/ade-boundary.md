### Changed
- **PiCode is an ADE for coding-agent CLIs, not a Pi UI.** The README, the docs landing page, `llms.txt` and the operating contract now describe an Agent Development Environment for Pi, Claude Code, Codex, Grok, Hermes Agent, OpenCode, Muse Code, Antigravity and Omp; no CLI is required to install PiCode, and tmux is the only runtime dependency (ADR-0179).
- **Getting started installs tmux and PiCode first.** The agent CLIs come afterwards, from Agent CLIs or with the vendor's command; the from-source, remote-server and shared-server guides no longer ask for `pi` on PATH before the install.
- The `picode --help` banner and the systemd unit's `Description=` now say "browser ADE for coding-agent CLIs".
