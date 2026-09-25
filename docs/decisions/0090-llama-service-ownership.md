# ADR-0090: Explicit llama.cpp service ownership

- **Status**: accepted
- **Date**: 2026-09-06

## Context

PiCode currently connects to a user-managed llama.cpp router. Delivery 4 needs
execution presets and lifecycle controls without taking over a service or
deleting files unexpectedly. The browser may be remote from the machine that
runs the router, and a loaded model may have active Pi consumers.

## Decision

PiCode adds a distinct local service profile only when the owner explicitly
creates it. The profile pins a verified llama-server binary, models directory,
loopback bind address and execution settings. External connections remain
read-only for lifecycle and cache operations. Start, stop, restart, update and
cleanup are durable reviewed jobs with preview, revalidation and an explicit
confirmation; restart refuses while active consumers exist unless the owner
confirms the interruption. Updates download and verify a pinned release before
switching, retain the previous binary for rollback, and never execute an
arbitrary shell command. Remote routers expose inspection and model operations
only; local process probes are never used to infer remote ownership.

## Consequences

Users can see the effective command and ownership boundary before a mutation.
Service setup becomes a separate responsibility from connecting to a router.
The first implementation is Linux/WSL-only and must identify the server host;
Windows and remote lifecycle remain unsupported until their own mechanisms are
designed. Cache cleanup can remove only files owned by a PiCode profile that
are not referenced by a model or active job; unknown ownership is preserved.

## Amendment (2026-09-23): lifecycle reviews settle stopped jobs first

Owner-approved with ADR-0083's amendment. The preview for start, restart,
update, rollback and cleanup does not count `interrupted` or `abandoned` jobs,
and before it reviews, model jobs on the owned endpoint are reconciled against
whether the owned process is running: while it is not, they are interrupted.
An unknown job there used to refuse every lifecycle action, including the start
that could have answered it. At start the service also removes what an install
killed mid-way leaves: its download temps, and an unrecorded `release-*` folder
that is incomplete. A complete unrecorded installation is still preserved.

## Amendment (2026-09-25): the daemon clears a killed supervisor's group

Owner-approved (2026-09-25). A supervisor killed outright (SIGKILL, so no pipe
EOF handling runs) let the router die of its parent-death signal but left the
per-model servers the router started running, holding memory. The supervisor
now reports its router's PID — the router's process group — to the daemon on a
dedicated descriptor (`PICODE_LLAMA_ROUTER_FD`, fd 3), which the supervisor
keeps from the router (close-on-exec, variable removed). When the supervisor
exits, the daemon kills that group only if its leader is gone and members
remain: a live process whose PID is the group's number means the number is no
longer the router's, and nothing is sent. The supervisor itself now kills a
router that exited on its own before reaping it, so the group's number cannot
be reused in between. Still unhandled: the daemon and the supervisor both
killed at once (the group ID is not persisted); a systemd stop kills the whole
unit's cgroup anyway.

## Alternatives considered

- Treating every configured URL as PiCode-owned risks stopping another user's
  service and was rejected.
- Exposing an arbitrary command field would bypass review and reproducibility.
- Deleting unreferenced GGUF files without ownership evidence would make
  recovery impossible and was rejected.
