# ADR-0210: Workspace overview reads a bounded activity summary

- **Status**: accepted (owner, 2026-09-24)
- **Date**: 2026-09-24
- **Boundary**: protocol and security model — `GET /api/workspaces/{id}/activity` exposes a workspace-scoped summary of durable store events, never their raw payloads.

## Context

The first workspace overview shows current state but cannot answer what changed since a visit. The change feed retains seven days of orchestration events. Its raw payloads can include Inbox bodies and agent configuration, and several events carry no durable workspace identity. The Git graph already supplies recent commits through an owner-scoped read.

## Decision

The new read returns at most 50 events from the last seven days. It selects only `mission.changed`, scoped `inbox.created`/`inbox.updated`, and `agent.added` with an unambiguous workspace ID. The response includes event ID, kind, entity ID, short title, action, state and time. The server discards raw payloads. A read or snooze of an Inbox question does not become a timeline entry; a resolved question does. New mission events record action, state and title; older events appear as generic updates. The browser joins this read with at most ten Git commits and stores the previous visit time locally per workspace and browser. A visit older than retention is labeled as recent changes, not complete history.

| Event conditions | Timeline action |
|---|---|
| Missing or different workspace identity | Exclude. |
| Mission change with this workspace ID | Include a summary. |
| Inbox creation or resolution with this workspace ID | Include; ignore read/snooze updates. |
| Agent creation with this workspace FK | Include. |
| Transient agent/terminal signal or unscoped event | Exclude. |

## Consequences

The timeline has a defensible scope and does not disclose an event's full data. It is intentionally incomplete for agent runtime transitions, terminal activity, PR changes and events older than seven days. Local visit time is browser-specific and is not synced across devices. If users need a cross-device audit, that would require a distinct persisted cursor and retention decision. The query is bounded and uses the existing event time index, but large installations should measure it alongside the workspace metrics read.

## Alternatives considered

| Alternative | Why it lost |
|---|---|
| Replay raw SSE events as history | The feed is live transport; raw payloads contain data this page need not expose. |
| Attribute every terminal event by its current workspace | A moved terminal could rewrite historical ownership. |
| Promise all changes since the last visit | The seven-day retention cannot support that claim. |
