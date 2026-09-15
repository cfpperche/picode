### Added

- **tmux guard.** PiCode terminals now run a `tmux` wrapper that refuses
  server-wide kills (`kill-server`, `kill-window`, `kill-pane`), pattern
  kills, and kills of sessions another terminal or the user created. It
  allows exact kills of sessions your terminal created and stamps sessions
  you launch with your terminal's mark, so cleanup stays possible. On by
  default; every refusal is explained on the pane and logged to
  `<dataDir>/tmux-guard.log`.

### Fixed

- A guard-style wrapper finding its real binary no longer depends on
  `dirname(1)`: under a minimal PATH it could resolve to itself and loop.
