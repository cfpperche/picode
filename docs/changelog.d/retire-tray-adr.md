### Added
- **ADR-0142: the shell becomes the only Windows resident.** The Go systray
  (`picode-desktop.exe --tray`) retires; `picode-shell.exe` takes autostart,
  the WSL keepalive, health/disk polling and the single tray icon, while the
  Go binary stays as the headless CLI tool the shell drives. Window close
  hides, tray Quit exits. Migration runbook in
  `docs/plans/retire-go-tray.md`.
