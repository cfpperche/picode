# ADR-0199: Persistent missions and versioned outcomes

- **Status**: accepted
- **Date**: 2026-09-23
- **Boundary**: persistence — outcomes outlive native sessions, agents and workspace records.

## Context

The owner approved the M0–M3 scope in [Missions](../plans/missions.md).
Agent activity and Delivery declarations cannot represent a durable objective,
its criteria, checkpoints and explicit human acceptance across assignments.

## Decision

A mission is a first-class store entity. One transaction writes its state,
assignment reservation, monotonically increasing version, durable history,
request receipt and `mission.changed` event. Native provider sessions remain
references. Mission records have no cascading foreign keys to agents,
workspaces, Inbox items or deliveries. A unique agent reservation includes
prepared and uncertain sends. Every mutation requires a request ID; retries
return the original result, and conflicting payloads or versions are refused.

Criteria and evidence have a scope version. Evidence is attributed to its
reporter; the server captures the candidate revision and file digest. Review
requires passing evidence for every criterion and a clean committed Git
candidate. Only the owner accepts. Acceptance history captures the complete
reviewed mission; future work requires reopening. Acceptance never merges or
deploys. Requested changes start a new evidence scope.

## Consequences

History survives runtime restarts and removal of referenced entities. The
change feed remains a notification channel, not an audit log. Bounded text,
criteria, evidence and history retain data on archive; capacity refuses new
work instead of silently pruning evidence. See [architecture](../architecture/missions.md)
for limits. A full SQLite backup retains receipts and historical snapshots.
Filesystem observations are point-in-time checks: the owner must stop writers
before accepting, and PiCode does not lock out unrelated local processes.

## Alternatives considered

- Infer completion from idle/checklists: no proof of outcome or owner acceptance.
- Store missions in native sessions: ties retention and transfer to one CLI.
- Reuse Delivery as the mission: changes its author and repository boundary.
- Use only feed events: retention can remove the history needed for recovery.
