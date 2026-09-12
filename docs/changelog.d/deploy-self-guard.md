### Fixed

- **Deploying from a PiCode terminal no longer refuses because of that
  terminal.** The interlock that protects other people's turns counted the
  pane running `picode deploy` — which is working, by the act of asking —
  so a deploy started from inside PiCode could be told to wait for itself.
  It now ignores the caller's own pane; `--force` still means ending
  someone else's turn.
