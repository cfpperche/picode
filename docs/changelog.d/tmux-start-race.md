### Fixed
- Creating two terminals at once on a machine with no tmux server running
  no longer fails one of them with "server exited unexpectedly": the daemon
  retries the tmux startup race for `has-session` and `new-session`.
