### Changed
- **`make desktop-restart` swaps the v2 shell too**: the target now builds
  `picode-shell.exe` (Tauri, ADR-0120) alongside the tray + native host, and
  `scripts/desktop-swap.sh` stops, copies and relaunches it when it was
  running. A stale shell refuses page commands with a Tauri ACL error — its
  capability list compiles into the exe (2026-09-14: `btab_cdp_call` refused
  for days after the CDP bridge landed). `DRY_RUN=1` prints the plan and
  touches nothing.
