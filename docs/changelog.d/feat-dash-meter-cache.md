### Changed

- **The dashboard refreshes about 20× faster.** When one agent CLI writes,
  only that CLI's numbers are recomputed; a warm refresh went from ~275 ms
  to ~14 ms on a machine running nine CLIs.
