# 2026-09-23 — feat/macos-tests: the two defects the dispatched run exposed

Neither was about sharding, which is what the dispatch was testing: main's pushes classify macOS out of the matrix, so nothing had ever run those tests on this tree.
- `TestKillServerThroughAGuardedPath` — and a second fixture in the same file — bound tmux sockets under `t.TempDir()`, which on a macOS runner blows the 104-byte socket limit. Both use the package's `socketPath` helper now, the treatment 01994907 gave every other fixture in it.
- `TestLocateOMPRules` compared a fixture path with the one the reader resolves out of `PATH`; on macOS `/var` is a symlink to `/private/var`, so the two spellings of one directory never matched. `sameFile` resolves both sides — the idea `internal/tmux`'s `samePath` already documents for its own fixtures.
Both reproduced on Linux deterministically (a symlinked `TMPDIR`, then a 95-character one) and both pass with the fixes; `make ci-scoped` PASS, the Go stage hermetic.
Verified on the hardware that failed: the dispatched run is green on all three platforms — ubuntu, macOS, Windows, `CI gate: success` — the first fully green matrix of the night.
Blind spot: none new. The macOS job's applicability to main pushes is its own bullet in `docs/handoff/open/process.md`.
