# 2026-09-15 · feat/tmux-test-isolation — phase 2/4 of tmux-resilience

**What shipped.** `internal/tmuxtest` — a `TestMain` harness that puts a
whole test binary on a private tmux namespace and tears it down: private
`TMUX_TMPDIR`, `TMUX`/`TMUX_PANE` scrubbed (they outrank `TMUX_TMPDIR` —
the 2026-09-15 measurement), the private server killed twice with a 300 ms
grace, and the environment deliberately **not** restored (a straggler
watcher ticked after `m.Run` and leaked a session during this very work).
Wired into `internal/server`, `internal/tmux` (external test package —
`tmuxtest` imports `internal/tmux`) and `internal/term`;
`TestSuiteIsTmuxIsolated` fails loudly if a `TestMain` is removed.

**Harvested from `feat/tmux-orphans`** (stalled, 241 behind, now empty of
uncommitted work worth keeping): `SocketDirEnv`, `DefaultSocketDir()`,
`KillIsolatedServer` — which refuses the user's socket directory — and
`isolation_test.go` (its 140-session incident comment kept). **Fixed its
loaded gun**: the child env was `append(os.Environ(), …)`, so an inherited
`$TMUX` still won; `IsolatedEnv(dir)` scrubs first. The branch's remaining
diff is superseded by this commit; the worktree can be removed.

**Also fixed.** A tmux client with no server starts one and, for a command
needing no session, watches it exit — "server exited unexpectedly".
`serverAbsent` now folds that into "no server", so the tmux app renders an
empty server instead of an error after a crash (found by the harvested
test, which is exactly the state it asserts). And the terminal settings
catalog — read from the running tmux — answered 500 on a machine whose last
terminal closed; it now answers 503 with "Start a terminal to read the tmux
option list." (an isolated shard exposed it: the test had passed only
because production's server happened to be alive).

**Flake evidence (not fixed, recorded).** `TestCLIAdapterPreviewMatchesExecution`
— already an undiagnosed one-off in `docs/handoff/open/terminal.md` — failed
once in a shard-3 run with `TempDir RemoveAll cleanup: directory not empty`
(a fixture process still writing as the temp dir is removed) and passed in
the verbose rerun and in two later shard-3 runs. Same-shard tests share one
private tmux server now, which makes the window slightly wider; the test is
self-contained (passes 5× alone). Next session: make its cleanup kill the
session and wait for the process before the temp dir goes.

**Evidence.** Before/after `tmux ls` diffs across `internal/server`,
`internal/tmux` and `internal/term`: 26 → 26 sessions, zero new, zero
leftover temp dirs. The fourteen pre-isolation orphans were removed by
exact name.

**Gates.** `make ci-scoped: PASS`.

Phases 3 (guard UI toggle) and 4 (dedicated socket) remain —
`docs/plans/tmux-resilience.md`.
