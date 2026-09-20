### Added
- **`make land BRANCH=<name>`** — the landing rite from the root: it verifies
  the branch, refuses when the root's local changes overlap it, fast-forwards
  `main`, and runs `make ci`. It never commits, so it cannot sweep another
  session's staged work the way a `git add -A` in the shared checkout once did.

### Changed
- A session note's `## Debts` bullets now say so at closing time: they ride the
  board for 7 days, so the durable ones name the topic file that owns them.
