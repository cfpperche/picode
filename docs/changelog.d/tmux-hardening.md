### Added

- **tmux server loss is now recorded while it happens.** A daemon-side watch
  probes the tmux server every 15 seconds: a server that dies with sessions
  running gets a `terminal.server_lost` entry (last known session count,
  socket and when it was last seen) in the feed and a line in the daemon
  log; its recovery gets `terminal.server_back`, and a running server whose
  session count drops by five or more between probes leaves a log line.
  Until now a lost fleet was only reconstructed on the next daemon start.

### Changed

- **AGENTS.md**: the scratch-tmux rule now names the measured mechanism —
  `$TMUX` outranks `TMUX_TMPDIR` (so a "isolated" client still talks to the
  server it was started in), `-L` overrides `$TMUX`, and `TMUX_TMPDIR` only
  counts when that directory exists.
