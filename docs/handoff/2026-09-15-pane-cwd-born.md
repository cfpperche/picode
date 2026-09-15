# 2026-09-15 — feat/pane-ready: the cwd race, closed at the read

`make ci` on main came back red right after the harness-guard merge, on a test
my previous fix had not covered: `TestTerminalText` got **404** from
`GET /api/terminals/{id}/text?path=a.go`. Same root cause as the creation
response — the live `#{pane_current_path}` read races the pane's own process:
tmux answers with the *server's* directory (the daemon's cwd) until it has
polled the new pane. `freshTermView` fixed the creation answer only; `/text`
resolves the requested path against the live cwd, so it 404'd.

This time I measured for a discriminator instead of guessing: `#{pane_pid}`
comes back **set** in every wrong answer (33 of 40 creations, `pidZero=0`), so
"the pid reads as ready" cannot separate the states. What can: the folder we
created the session in. `Manager.bornCwd` records it for five seconds and
answers with it while tmux is still reporting the server's directory, dropping
the record the moment tmux reports anything else — so a shell that `cd`s right
after opening shows its new folder immediately, and a session with no record is
read from tmux exactly as before.

Tests: `TestPaneCwdIsRightFromTheFirstInstant` (12 creations, each session
ended so the next starts a fresh server) fails on iteration 0 without the fix
with the race's exact message; `TestPaneCwdFollowsAShellThatMovedRightAfterCreation`
pins the other half. `internal/tmux` ok, `internal/server` ok twice (the first
run flaked in a test I could not name — the log was not kept; the shard runner,
which is the shape that failed in CI, passed 3/3).
Not verified: macOS (no `/proc` involved here, but the race is tmux's; the
measurement is Linux), and the CI shard that failed is load-dependent — the
tmux-level test is the deterministic net.
visual-review: n/a (no UI change)
