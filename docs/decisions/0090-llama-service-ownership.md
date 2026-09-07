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

## Alternatives considered

- Treating every configured URL as PiCode-owned risks stopping another user's
  service and was rejected.
- Exposing an arbitrary command field would bypass review and reproducibility.
- Deleting unreferenced GGUF files without ownership evidence would make
  recovery impossible and was rejected.
