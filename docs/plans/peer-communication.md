# Peer communication

Owner-approved implementation, 2026-09-09. Direct messages between existing
sessions; no task scheduling, recipient startup or automatic model turns.

## Delivery

1. Store authorized connections and durable messages. Scope contacts to the
   same workspace; an explicit connection represents one recorded conversation.
2. Expose four tools through an embedded, stateless Streamable HTTP MCP server
   using the official Go SDK. Authenticate every request with a connection token.
3. Add an owner-facing Messages tab to Agent CLIs, including managed Pi sessions,
   opt-in, revoke, connection setup and history. Preserve independent app views.
4. Exercise real HTTP MCP clients, persistence/retry/revocation, browser states
   on an isolated instance, scoped gates, close and main CI. No deploy.

## Decision table / acceptance

| Conditions | Action / evidence |
|---|---|
| No opt-in or invalid token (including auth off/loopback) | MCP refuses; HTTP tests |
| Scoped token used on owner API | Refuse, including auth off/loopback; auth test |
| Connection enabled for a recorded native conversation | Return secret once; store hash only |
| Recorded conversation changes, connection revoked, owner removed | Old credential cannot operate; store tests |
| Two opted-in connections in same workspace | Contact discovery and durable send |
| Different workspace, self-send or disabled recipient | Refuse without writing |
| Same sender and request ID, identical content | Return original receipt, no duplicate event |
| Same request ID, different content | Conflict, do not silently replace |
| Read repeated or daemon restarts | Same unacknowledged messages remain visible |
| Ack own received IDs | Record acknowledgement once, never implies task completion |
| Ack batch includes another inbox or unknown ID | Refuse whole batch, no partial acknowledgement |
| Reply references a message outside this pair | Refuse |
| Invalid/oversized input, inbox capacity exhausted | Refuse with bounded error |
| UI refresh fails | Retain history; block mutation until successful reload |
| UI owner switches while request in flight | Ignore stale response, discard one-time secret |

## Boundaries

A bearer capability identifies its enrolled conversation; MCP does not attest
which native CLI process holds it. Install it only for that conversation, never
in shared/global CLI configuration. PiCode refuses it once its recorded session
pointer changes; native CLI session discovery remains best effort (ADR-0084).
This is not proof of a live recipient. Setup is explicit; no native config files
are silently rewritten and no running process is restarted.

The first delivery is pull-based. Native automatic notification, launch-time
credential injection and automatic adapter installation are later increments.
Interoperability claims name the tested clients; six terminal integrations do
not imply six working MCP adapters.
