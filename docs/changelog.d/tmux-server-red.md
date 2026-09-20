### Fixed

- **A terminal no longer dies the moment it opens on a service without a
  `SHELL` setting.** The daemon fell back to `/bin/sh` — dash on
  Debian-family systems — and handed it a bash-only argument, so the pane
  exited before the first prompt and took its tmux server with it while the
  app still showed the terminal as running. The fallback is bash now, and
  the argument goes only to a shell that accepts it.

- Opening a terminal whose shell exits immediately reports the failure and
  keeps nothing behind, instead of leaving a terminal that reads as open
  with no session to attach to.
