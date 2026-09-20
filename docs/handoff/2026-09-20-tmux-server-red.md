# 2026-09-20 — tmux-server-red: the red internal/server suite was dash, not a socket

Shipped (423fec89): `defaultShell()` prefers `$SHELL` only when it names an existing executable, else `/bin/bash`,
else `/bin/sh`; `shellTakesRcfile()` passes `--rcfile` only when the shell resolves through symlinks to bash;
`ensureShell` proves the session is alive after `new-session` and otherwise fails the create naming the shell.
`internal/server/cli_launch.go` builds its CLI launch script from the same helper.

Root cause: with no `SHELL` in the daemon's environment (systemd, a container, a test binary) the fallback was
`/bin/sh` — dash on Debian-family systems — and `ensureShell` handed it `--rcfile`, which dash rejects
(`Illegal option --`, status 2). The pane exited immediately, a tmux server with nothing else on it followed
under `exit-empty`, and `new-session` still answered 0, so the API reported `201 running:true` for a terminal
that never lived and every later tmux call answered "no server running" — hence `patch … = 400`,
`GET /api/terminals/settings/catalog = 503`, `mouse=""` and the cwd failures across all four CI shards.

Tests (`internal/server/terminals_test.go`): a live session with `SHELL` unset and with a bogus `SHELL`; a shell
that exits at once is a 500 with no row left; `shellTakesRcfile` follows a symlink chain. All fail pre-fix
(simulated old fallback + unconditional rcfile: `create = 500`; pre-fix the API said 201 with a dead session).

Verified: `./scripts/go-test.sh ./internal/server/` → ok in 4 shards, longest 88.9s; `make ci-scoped` PASS (fmt,
vet, hooks, go[4]); merge of main (`pi-settings-parity`) then `make close` green. Blind spot: the failure was
reproduced with the old fallback simulated in a test, not against a systemd-launched daemon. visual-review: n/a.

Docs with it: `docs/architecture/terminal-bridge.md` (the shell contract) and `docs/changelog.d/tmux-server-red.md`.
The earlier `docs/handoff/open/terminal.md` guess — per-instance socket, or fixture namespace — was wrong on both;
that bullet is `[x]` with the real cause.
