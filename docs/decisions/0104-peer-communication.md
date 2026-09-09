# ADR-0104: Embedded MCP for direct session messages

- **Status**: accepted (owner approval, 2026-09-09)
- **Date**: 2026-09-09

## Context

The owner wants communication between terminal CLIs and managed Pi without
repeating Tachyon's orchestration complexity. ADR-0091 keeps guests in terminals;
MCP tool calls can add a narrow communication capability without adopting an
agent runtime protocol. Terminal/session attribution remains ADR-0084's best
effort native discovery. Existing broker messages target agents and create tasks,
which is the wrong boundary for direct conversation inboxes.

## Decision

One embedded Go module serves `list_contacts`, `send_message`, `read_messages`
and `ack_messages` at `/mcp/communication`, using the official Go MCP SDK and
stateless Streamable HTTP. Existing owner APIs enable/revoke a connection to a
recorded conversation; a random, separately scoped bearer is returned once and
stored only as a hash. Each operation revalidates identity and opt-in inside the
store transaction. Connections discover and message enabled peers only in the
same workspace. Two SQLite tables store connections and messages; no scheduler,
outbox service, sidecar or external broker. Replies are messages with `reply_to`.
Store mutations emit credential-free, body-free invalidation events (ADR-0048).

Send success means persisted. Read is non-destructive, acknowledgement explicit;
it does not mean work completed. Per-sender request IDs deduplicate retries and
conflicting reuse fails. Histories survive revocation and restart, until their
owner is deleted. Bounds: 16 KiB text, 100 messages per page/ack, 1,000 pending
messages per recipient and 10,000 messages per connection pair direction.
Capacity refuses further new sends; existing receipts remain retrievable.

A token is a capability, not native-process attestation. The operator configures
it only for the selected conversation. Recorded session changes invalidate it;
undetected native changes cannot be guaranteed. No generic paste fallback or
automatic wake is part of this release. Managed Pi and guest CLIs keep existing
runtime paths. Launch-time injection is a later adapter-specific increment.

## Consequences

The same binary, database and browser feed remain the deployment stack. The
SDK is the one new direct dependency: it maintains protocol negotiation, tool
schemas and HTTP behavior instead of a homemade MCP implementation. The second
small table makes token revocation independent of message history rather than
encoding security state in generic settings. Only hashes reach the database;
owner setup displays the credential once and never puts it in an event or URL.

Explicit setup costs a per-conversation client configuration step. Clients must
support HTTP MCP and custom authorization headers. This is a capability to
consult an inbox, not proof a model has seen its contents, a live-presence
service, or an agent orchestration API. If native session attribution is wrong,
the operator must revoke the connection; the UI states the recorded identity.
The owner is an administrator and can inspect all histories; peers cannot.

## Alternatives considered

- Sidecar / broker: another lifecycle and deployment without a demonstrated need.
- Existing agent task queue: conflates message acceptance with model execution.
- Credentials in the generic settings table: fewer tables, less explicit ownership.
- Automatic paste or universal push: lacks recipient/session delivery proof.
- Protocol client per CLI: reopens ADR-0091 without helping the basic mailbox.

Acceptance matrix: [implementation plan](../plans/peer-communication.md).
Benchmark adaptation: Paseo's thin tool boundary and Orca's separate receipt and
acknowledgement, while omitting their orchestration layers. The existing
[t3code/Paseo/Cursor adaptation](../benchmarks/2026-08-24-adopt-t3code-paseo-cursor.md)
also motivates a small shared contract with app-owned presentation.
