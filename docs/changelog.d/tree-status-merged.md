### Fixed

- `make worktree-status` no longer reports a finished branch as a stalled one:
  a worktree whose branch `main` already contains reads *merged — `make
  worktree-gc` can remove it*, and the handoff board's **In flight** section
  lists only trees with unfinished work (a dirty tree is never called merged).
