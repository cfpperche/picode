### Fixed

- **No more runaway tmux helper.** A terminal whose PATH held two PiCode
  instances' helpers (a test instance opened inside PiCode) could leave a
  helper process spinning at full CPU; each helper now skips the others and
  reaches the real program.
