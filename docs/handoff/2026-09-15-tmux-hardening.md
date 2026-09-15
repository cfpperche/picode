# 2026-09-15 · feat/tmux-hardening — phase 1/4 of tmux-resilience

**What shipped.** `StartTmuxServerWatch` (`internal/server/tmux_watch.go`,
wired in `cmd/picode/main.go`, 15 s): `ServerInfo.Running` is probed once per
tick and the transitions are recorded — `terminal.server_lost` (last known
session count, socket, last-seen time) and `terminal.server_back` through
`Store.AppendEvent` (ADR-0048, so the feed carries them), plus one journal
line each; a running server whose session count drops by ≥5 between probes
leaves a journal line only. A failed listing carries the previous count
forward, so it cannot fake a loss. Until now the boot diff (ADR-0085) was
the only reconstruction, which is why the 2026-09-15 death under a live
daemon was invisible until somebody noticed.

**AGENTS.md correction.** The scratch-tmux rule carried the wrong mechanism
(`XDG_RUNTIME_DIR`). Measured 2026-09-15, in three probes: `$TMUX` outranks
`TMUX_TMPDIR` (a client started inside a session prints production's
socket); `-L` overrides `$TMUX`; and `TMUX_TMPDIR` only counts when that
directory exists — missing, tmux silently falls back to the shared
`tmux-<uid>` dir. The contract now carries the safe recipe
(`mkdir -p $dir && TMUX_TMPDIR=$dir tmux -L <unique>`), and the heading
"Two rules" became "Rules" (it had three).

**Tests.** Decision-table test for every row, probe test with a fake prober
(including the failed-listing row), and one integration test that flips
`Running` for real on an isolated socket — with the `#{socket_path}`
assertion that skips instead of touching the live server.

**Gates.** `make ci-scoped: PASS` (fmt, vet, hooks, go×4).

Phases 2–4 (test isolation harvest, guard UI toggle, dedicated socket) are
scoped in `docs/plans/tmux-resilience.md`; phase 2 is next.
