### Fixed
- **A terminal whose shell is gone no longer survives as an empty row.** tmux
  reports a session as alive for the few milliseconds before it reaps a pane
  that died at once, so PiCode could keep a terminal that never lived — and
  every later action on it answered "no server running". The creation now asks
  a second time, 50 ms later, before it trusts the answer.
