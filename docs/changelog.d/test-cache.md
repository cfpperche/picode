### Changed

- **Go tests are cached across worktrees.** `scripts/go-test.sh` builds with
  `-trimpath`, so the test cache no longer depends on the tree's absolute path:
  a package tested once is reused in every worktree and terminal (measured 10.4 s
  per package before, `(cached)` after). Sessions that touch several packages
  get those minutes back on every iteration.
- `make ci` keeps its full output in `var/ci-last.log` and says so when it
  fails. One full-matrix run failed unreproducibly with nothing but a bare
  `FAIL` left in the transcript; retries would hide that, evidence does not.
