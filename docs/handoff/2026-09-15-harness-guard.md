# 2026-09-15 — feat/harness-guard: the harness stops what it started

Two follow-ups from the tmux flake session, both about "who owns this server".

**`qa-scratch.sh`.** `stop` ran `fuser -k <port>/tcp`, so it ended *whoever*
held the recorded port: with a stale port file that is a neighbouring scratch's
daemon — and its own survived (exactly what I found after a day of QA rounds:
four of my daemons still listening, their worktrees long gone). Now: the port
and pid come from the daemon's own `data/server.json` (fallback: the pid file,
then the port file), terminals are removed through the API first (so no session
is orphaned), a port held by a *different* pid is left alone with a warning,
the daemon is ended by that pid and the stop verifies the port is quiet.
`start` refuses a port someone else holds instead of killing its holder, and
records the pid; `status` prints port/pid/alive/health.
Verified by hand in this worktree: start+seed+status, the stale-port regression
(B survived with its terminal while A's daemon was stopped), the occupied-port
refusal, and a normal stop that reports the port free.

**`tmux.Manager`.** With `-S` in a directory that did not exist, `new-session`
printed `error creating …` and **exited 0** — `NewSession` reported success with
nothing behind it (measured while giving two tests their own sockets).
`NewWithSocket` now makes the directory and `runStdin` promotes tmux's socket
complaint to an error; two new tests pin both halves (`…MakesItsDirectory`,
`…ReportsAnUnusablePath`).

**Fixtures.** `TestTmuxServerWatchIntegration` and
`TestKillIsolatedServerEndsAServerInItsOwnDirectory` used to rebind the
process-wide `TMUX_TMPDIR` and then dismantle that server — anything else in
the binary (background goroutines included) landing on it in that window died
with it. Both now own a socket with `-S`; the watcher test asserts the suite's
namespace is unchanged, the kill test keeps asserting the directory guard.
Gates: `internal/tmux` ok, `internal/server` ok (226 s), `make ci-scoped` PASS.
Not verified: a daemon whose socket path is unusable (the new error path is
pinned by test, but production's data directory always exists).
visual-review: n/a (no UI change)
