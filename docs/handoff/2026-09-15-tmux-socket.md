# 2026-09-15 · feat/tmux-socket — phase 4/4 of tmux-resilience (ADR-0139)

**What shipped.** Managed sessions moved to a per-instance socket:
`tmux -S <dataDir>/tmux.sock` (production `~/.picode/tmux.sock`), with the
default-socket Manager riding along as the **drain** for sessions created
before the move. `Manager` carries the socket (`NewWithSocket`), every
command gets `-S` in the one run path, and the public API is unchanged: the
session-scoped methods are private implementations plus wrappers that fall
back to the legacy server (`HasSession`, receipt/kill/send-keys/type/paste/
clear, pane reads, env/options/respawn, input snapshot), pane-addressed
commands retry by trial, and the server-wide reads (`ServerSessions`,
`ListSessions`, `ListOwned`, `ServerInfo`) merge both sides. The bridge
attaches with `SocketFor(session)`. `cmd/picode` and the docs fixture wire
primary + legacy.

**Why this is safe without a fleet restart.** tmux has no live migration, so
the drain is the migration: legacy sessions stay on the default socket until
they end or are restarted, the daemon follows both meanwhile, and the
wrappers disappear with the last legacy session. A pane's processes inherit
`$TMUX` naming their own server, so in-pane commands — the guard's probes
included — work on either socket with no code change; verified on a scratch
via `/proc/<pane>/environ`.

**Verified.** `-S` is carried (and absent for `New()`); a legacy session is
found, tailed, pid-read, socket-resolved and killed through the primary; a
pane-addressed paste/submit falls back and reaches the legacy pane; reads
merge; a lone Manager behaves as before. On a scratch: its sessions land in
its own socket, **the owner's tmux sees none of them** (the AGENTS.md trap
about scratches is fixed by construction), and the guard marker/wrapper are
in the pane. AGENTS.md, the browser skill, the architecture docs and the
docs-site attach flow now describe the new topology.

**Honest note.** Twice during verification I used `send-keys` carelessly
(once into another agent's scratch pane, once into my own with an empty
target). Nothing executed in either case, both buffers were cleared, and
production never moved; the rule adopted is: no `send-keys` without a
target resolved on the correct socket, and prefer read-only checks.

**Gates.** `make ci-scoped: PASS` (fmt, vet, hooks, go×4, test-js, build,
docs); `go test ./internal/server/` re-run clean after the method renames.
