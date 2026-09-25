# Restarts, processes and tmux

> Reference for [AGENTS.md](../../AGENTS.md) "Rules for the agent itself".
> AGENTS.md keeps the rules; this file keeps the measurements and incidents
> behind them.

- **Owner-grade restarts are serialized, one at a time, verified.** `make
  deploy`, `make desktop-restart` and a direct `picode deploy` all take
  `/tmp/picode-mutate.lock` — never run two of them concurrently, and never
  work around the wait. Each one finishes and is *verified* (daemon health,
  `tasklist` for the shell) before the next starts; a turn never ends with a
  mutation command still running, and `--force` is never used while any
  background job of yours is in flight (2026-09-21: a forced deploy racing a
  desktop-restart's cargo build wedged WSL's IO; the VM died with all 21
  sessions and the resident was dead until a machine reboot).

- **Never `pkill -f <pattern>`** (or `killall`) from a session: the pattern
  matches the command line running it, so the shell that issued it dies first
  (a session killed its own `make` chain this way, and an earlier
  `grep '^picode-'` sweep killed 29 live terminals). Use the script's own verb
  (`./scripts/qa-scratch.sh stop <name>`, `fuser -k <port>/tcp`), or an exact
  PID you started and can still see.
- **Never `tmux kill-server`** — not even with `TMUX_TMPDIR` set, which is how
  a QA session took the production tmux down and cost the owner every running
  terminal (2026-09-15). Kill the exact session you created
  (`tmux kill-session -t picode-sh-<id>`), and let
  `./scripts/qa-scratch.sh stop <name>` do it for a scratch instance.
  **`$TMUX` outranks `TMUX_TMPDIR`** — measured 2026-09-15: a client started
  inside a session talks to the server named in `$TMUX` no matter what
  `TMUX_TMPDIR` says (a probe with `TMUX_TMPDIR=/tmp/x` printed production's
  `/tmp/tmux-1000/default`). `-L` does override `$TMUX`, but then
  `TMUX_TMPDIR` counts only if that directory exists — missing, tmux falls
  back to the shared `tmux-<uid>` dir silently. The safe scratch recipe is
  `mkdir -p $dir && TMUX_TMPDIR=$dir tmux -L <unique-name> …`; a session
  launched with neither lands in the owner's tmux (and in every other
  agent's `pkill` radius). Since ADR-0139 each PiCode instance — production,
  `qa-scratch`, the docs fixture — runs its own server on a socket in its
  data dir, so a scratch's terminals no longer land in the owner's tmux;
  tools that shell out to `tmux` themselves must still pass `-S <socket>`.
  Clean up by exact session name, or check the
  scratch's own Terminals list before blaming the server. PiCode terminals
  refuse `kill-server` through the tmux guard — but it is a guardrail, not a
  boundary: an absolute `/usr/bin/tmux kill-server` reaches that terminal's
  own server, and raw kills on any socket still bypass the guard.
  A Go test that launches a real terminal cleans it up with a context of its
  own: `t.Context()` is canceled *before* cleanup functions run, so every
  tmux call made with it fails silently and the fixture leaks one session
  per run into the shared server (four `picode-sh-feed-fixture-*` sessions
  were found in the owner's `tmux ls` this way).
- **Know which tree you are in.** `make dev`, `make ci-scoped` and `make close`
  print `<worktree> on <branch>` before doing anything; the same line answers
  "did I edit the root checkout by mistake?" (`make worktree-status` lists every
  tree with its dirty count). The root checkout is shared — a stray edit there
  is the failure mode that blocks every other session (AGENTS.md §5).

- **`/tmp` is a shared 16 GB tmpfs in memory.** Every session's scratchpad,
  every test's `t.TempDir` and, until 2026-09-25, every test binary's link
  output lived there; that day it filled and `make ci` on `main` failed three
  times with "no space left on device" while the disk had 772 GB free.
  `scripts/go-test.sh` now gives each run its own
  `~/.cache/picode-gotmp/run.*` as `GOTMPDIR` and removes it on exit (a value
  the caller sets wins): link outputs and, with TMPDIR unset, the tests'
  `t.TempDir` go to disk; the suites' tmux sockets stay under `/tmp`. Do not
  delete `~/.cache/picode-gotmp` while a gate runs — its tests live there. Large scratch output belongs on disk too
  (`var/` in your worktree), and a finished session's scratchpad can go.

## Live CLI checks with the owner's logins (2026-09-25)

A scratch instance has its own HOME, so the agent CLIs in it are signed out
(only Pi's login is copied). `QA_LOGINS=1 ./scripts/qa-scratch.sh start <name>`
copies each CLI's login files and PiCode's vault into the scratch (mode 600),
lists every copy in `var/qa/<name>/logins.list`, marks the worktree trusted in
the copied Claude Code and Codex configs (Grok's trusted-folders file is copied
as is), and `stop` removes exactly the listed files. The originals are never
written. **The owner runs the start**: Claude Code's safety classifier refuses
an agent copying credentials, and that refusal stands — ask for
`! QA_LOGINS=1 ./scripts/qa-scratch.sh start <name>` and drive the scratch
through its API. A fresh scratch HOME makes some CLIs do first-run work
(Hermes installs a browser tool on its first command, ~28 s, longer than
PiCode's 20 s preflight): the second launch goes through.
