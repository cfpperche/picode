# ADR-0139: tmux dedicated socket (per-instance)

- **Status**: accepted (owner, 2026-09-15 — "vamos resolver", after the plan
  in `docs/plans/tmux-resilience.md`; the guard's ADR-0138 named this as its
  next step if incidents continue)
- **Date**: 2026-09-15
- **Boundary**: process — where managed sessions live, and what a
  `kill-server` on any one socket can reach. Changes the attach path and the
  blast radius between PiCode's terminals, the user's own tmux, and other
  PiCode instances (scratch QA included).

## Context

Every PiCode session — every agent terminal, every shell — lives on the
user's **default** tmux server, beside their personal sessions. Three
incidents followed (29 sessions on 2026-09-06; 140 on 2026-09-13;
production twice on 2026-09-15). ADR-0138's guard closed the realistic
accident (plain `tmux kill-server` typed in a managed terminal is refused),
but it is a guardrail, not a boundary: a raw `/usr/bin/tmux kill-server`
still reaches everything, and PiCode's own sessions sit inside any
collateral damage the user's own tooling causes (and vice versa).

Two facts measured during this work shape the decision:

- **`$TMUX` outranks `TMUX_TMPDIR`** (2026-09-15): isolation attempts that
  only set the directory do nothing inside a session. A socket **path**
  (`-S`) or a socket **name** (`-L`) is what actually selects a server.
- **Panes are self-referential**: a tmux pane's `$TMUX` names its own
  server, so any `tmux` command typed inside a managed terminal already
  targets the right server after migration — the guard's probes included.
  Only the daemon (no `$TMUX`) and the bridge's attach need the socket
  spelled out.

A dedicated server also fixes an adjacent trap recorded in AGENTS.md:
scratch instances (`qa-scratch`, docs fixture) currently create their
sessions on the owner's server. A per-instance socket puts each instance's
sessions in its own server by construction.

tmux has **no live migration** between servers — a session is anchored to
its server process. The migration is therefore a **drain**: new sessions go
to the dedicated socket, existing ones stay where they are until they end
or are restarted (`Restart terminal` already respawns on the new socket).

## Decision

Managed sessions live on a **per-instance socket path**: the daemon starts
its tmux work with `tmux -S <dataDir>/tmux.sock` (production:
`~/.picode/tmux.sock`; each scratch and test instance gets its own path, so
instances can never share a server by accident). Implementation shape:

- `tmux.Manager` gains a socket path; every command it runs carries `-S`
  when one is set. `New()` (no socket) keeps today's behavior, so the
  existing suites and any caller that does not opt in are untouched.
- **The drain lives inside the Manager**, not in a new type: a Manager with
  a socket may hold a second, legacy Manager (the default socket). Every
  session-scoped operation falls back to the legacy server when the session
  is not on the primary one, and the server-wide reads merge both lists.
  This keeps `Deps.Tmux` a concrete `*tmux.Manager` and the ~120 call sites
  unchanged.
- The **bridge** asks the Manager for the socket to attach with
  (`SocketFor(session)`: the session's own socket during the drain, empty
  for the default), because it spawns `tmux attach-session` itself.
- The guard wrapper needs no change: its probes inherit `$TMUX` from the
  pane, which names the right server on either socket. The ADR-0138 comment
  claiming the probe "always asks the default server" is stale and gets
  corrected with this ADR (fail-closed remains for probes made outside a
  pane).
- Rollback is the previous binary: the dedicated server keeps running
  unused, sessions can be killed by exact name from either side, and no data
  moves.

## Consequences

Easier: a `kill-server` (typed raw, or run by any tool) on one socket cannot
reach the other side; the user's personal sessions stop being collateral
for PiCode incidents and vice versa; scratch instances stop polluting the
owner's tmux; each instance's sessions die with its own server, which makes
instance cleanup a single operation.

Harder / accepted cost: user-facing attach changes to
`tmux -S ~/.picode/tmux.sock attach -t picode-…` (docs-site teaches it); the
drain keeps two servers readable until the last legacy session ends — an
adoption window whose code is deleted after; a stale `tmux.sock` file after
a hard crash is normal tmux behavior (the next client starts a fresh
server on that path); socket path length caps near 108 bytes, which the
data dir easily fits (`~/.picode/tmux.sock` is 30).

Who breaks if we're wrong: the drain's fallback path (session not found on
primary → ask legacy) is the risky seam; if it is wrong, a legacy terminal
shows as absent and its pane cannot be attached. The tests drive both
servers for every session-scoped method the bridge and terminal operations
use, and the fallback is fail-visible (the terminal surface reports the
session gone rather than pretending).

## Alternatives considered

- **`tmux -L picode` (a named socket in the shared directory)**: same
  isolation, but the name is global — two instances (production + a scratch)
  would share one server, exactly the class of accident this ADR removes.
  A per-instance path wins.
- **`TMUX_TMPDIR` per instance**: rejected by measurement — `$TMUX` outranks
  it inside sessions, and the directory has to pre-exist or tmux silently
  falls back to the shared one (both 2026-09-15).
- **Kill and restart every session at the switch** ("migrate" the fleet):
  no tmux primitive moves a session; doing it by restart is a fleet-wide
  interruption the owner explicitly did not want while work is running. The
  drain gets the same end state without an interruption.
- **Do nothing; rely on the guard (ADR-0138)**: leaves the raw-bypass and
  cross-instance blast radius open, and ADR-0138 itself named this ADR as
  the follow-up.
