### Fixed
- **Development process (ADR-0105 follow-up).** The pre-commit hook lets the
  release cut through (a `CHANGELOG.md` edit that adds a `## [x.y.z]`
  heading) and parses staged changelog fragments so a malformed one fails at
  commit time, not at release. `make deploy` commits only the refreshed
  captures, never whatever else was staged. `make adr` allocates the number
  across every worktree even when run from inside one. The split
  architecture files link to `docs/plans`, `docs/design` and
  `docs/benchmarks` again. Capture tolerance is an absolute 128 px budget
  with the count printed per surface.
