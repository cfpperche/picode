### Fixed
- **A terminal whose shell was already gone no longer stays in the list.** The
  creation check races the shell's own start and exit, so under load a dead
  shell could be accepted and left behind as a row that answers "no server
  running" to everything. A second check, which nobody waits for, reaps a
  just-created terminal whose session is gone — the promise is now that a
  terminal which never lived does not *survive*, not that it is refused on the
  spot.
