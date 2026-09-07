# 2026-09-07 — feat/cli-detect-fix: DetectMethod resolves symlinks and wrapper scripts

Shipped: `clilifecycle.DetectMethod` now `filepath.EvalSymlinks` the
executable before classifying and scans the script head (4 KB) for install
roots when the path itself matches nothing. Fixes the production report where
every CLI showed "Update check failed / No managed lifecycle" because real
installs are symlinks (`~/.local/bin/claude` → versions dir) or wrappers
(`hermes` execs the venv path).
Verified: new regression test (symlink + wrapper + unknown cases); DetectMethod
run against the real machine paths → pi/codex npm, claude native, grok vendor,
hermes git; ci-scoped green.
visual-review: n/a (no UI change)
Not done / debts: none new; ADR-0087 debts stand.
Merge: fast-forward ready.
