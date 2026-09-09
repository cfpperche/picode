# 2026-09-09 — feat/ci-platform-legs-2: second pass on the macOS and Windows legs

Shipped: the second full-matrix run reached the suites and found the next
layer. macOS: three cwd tests compared tmux's resolved pane path
(`/private/var/…`) with the unresolved `t.TempDir()` (`/var/…`) — the tests
now resolve their expectations (`resolvedTempDir` in `internal/tmux`, a
local `resolve` in `internal/server`); `PaneCommand` accepts `bash` because
macOS's `/bin/sh` is a bash binary. Windows: the drain decision table gave
the 80 ms hard deadline a 150 ms budget and a runner took 153 ms — every row
now has 400 ms of slack; the rows still prove which deadline released the
call and that Close ran.
Verified: the cwd failures reproduce on Linux under a symlinked `TMPDIR`
before the change and pass after; `make close`; a manual full-matrix run is
the proof for Windows.
visual-review: n/a
Not done / debts: none new; macOS and Windows stay tag/manual-only.
Merge: fast-forward ready.
