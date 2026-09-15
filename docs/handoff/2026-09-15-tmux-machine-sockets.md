# 2026-09-15 · feat/tmux-machine-sockets — the machine's sockets in the app

**What shipped.** The tmux app gets a **Sockets** tab (root `sockets`):
`Manager.MachineSockets` lists every socket file in the user's tmux
directory (the default socket and every `-L` name) plus this instance's own
`-S` path, with honest liveness — a plain unix dial, never a tmux client
that would start a transient server on a dead socket just to call it dead.
Rows: name (`· this instance` for ours), badge `running` / `idle` (ours,
not started yet) / `no server` (a leftover file), subtitle `N session(s) ·
M PiCode`, socket path in the meta. The tab badge counts running servers.

**Why.** ADR-0139 made the machine multi-server; the app merged the reads
but never said where anything lived. The owner asked on a production
screenshot whether the app shows sockets — it did not. Honest limit, stated
in the architecture doc: a `-S` socket outside the user's tmux directory is
undiscoverable (no registry), and the UI does not pretend otherwise.

**Visual review (scratch :8475, screenshots read).** First pass exposed two
real defects, both fixed and re-read: a new collapsible block defaulted to
collapsed (the whole point of the screen was hidden on first visit — the
block is now plain and open), and our own not-yet-started socket was
labelled "a leftover socket file" (it now reads `idle` with "the next
terminal this instance opens starts it"). Overlay audit `ok:true`.

**Tests.** `MachineSockets` integration: a live server, a stale file and an
unstarted own socket in an isolated `TMUX_TMPDIR`; counts, `ours` flag, and
the inert probe (second pass still sees the stale socket dead). App view:
scripted rows table + the tmux-missing empty state; the session inventory's
tab assertion moved 2 → 3.

**Gates.** `make ci-scoped: PASS`.
