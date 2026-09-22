### Fixed
- **PiCode Desktop (Windows): installing a CLI from Agent CLIs works after setup.** When the distro's npm installs into a root-owned `/usr` prefix, setup now points your account's npm prefix at `~/.local`, so **Install** for Pi, Claude Code, Codex, OpenCode or Omp runs without root. An nvm setup is left alone.
- Setup upgrades a Node.js older than 22 through NodeSource again (Pi needs 22.19 or newer), and names the replacement in the confirmation on a distro PiCode did not create.
