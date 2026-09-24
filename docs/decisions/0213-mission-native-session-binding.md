# ADR-0213: Pi mission first-session attribution

- **Status**: proposed; amends ADR-0200
- **Date**: 2026-09-24
- **Boundary**: process and persistence — the daemon derives the first Pi session from PiCode-managed runtime state and may bind it to an assigned mission on acknowledgement.

## Context

ADR-0200 requires mission writes to match the assigned native session, but a
mission can be assigned before Pi creates its first session. On the first
managed turn, `PICODE_AGENT_ID` and the mission packet may exist while the
assignment session is still empty. Requiring a client-supplied session path
would let the reporting process choose its own attribution, while rejecting
the write leaves this legitimate first acknowledgement unusable.

The `/api/missions/tool` request contract remains unchanged: Pi sends inherited
agent or terminal identity and the ordinary mission mutation fields; it does
not send a session path or `nativeSession` field. The daemon observes session
identity through PiCode's managed pending-session ledger, terminal runtime,
or pinned last session.

## Decision

For an assigned agent or terminal mission write, the daemon resolves the
current native session from authoritative PiCode state. If the assignment has
no session yet, the first valid acknowledgement may bind the observed session
using the existing store transaction. Every later write must match that
binding. A live terminal runtime with no current session is not attributed to
an older `lastSession`; a request-provided path is never authoritative. Exact
idempotent retries replay their original receipt before current-session
revalidation and never repeat a prompt submission. `show` and `context` keep
their assigned-agent identity gate but do not require a bound native session
or mutation fields.

## Consequences

First acknowledgement works when assignment precedes the managed session,
without adding a wire field or widening client authority. Session discovery
depends on PiCode's runtime and pending-session records; if they cannot prove a
current session, the daemon refuses the write and the owner must reopen or
reassign through PiCode. A defect in those observations could bind the wrong
session, so tests cover initial managed binding, terminal runtime authority,
and stale-session refusal.

## Alternatives considered

- Require the client to post its session path: rejected because a client-chosen
  path is not evidence of PiCode ownership and changes the reporting contract.
- Reject every write while assignment session is empty: rejected because a
  mission may be assigned before the first Pi session exists.
- Bind from persisted `lastSession` while a live terminal runtime is empty:
  rejected because it can attribute a new process to a stale session.
