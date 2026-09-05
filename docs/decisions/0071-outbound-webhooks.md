# ADR-0071: Outbound webhooks — the durable event log is delivered to owner-registered URLs

- **Status**: proposed
- **Date**: 2026-09-03

## Context

ADR-0048 named the anticipated consumer: "Push, the desktop, the phone and
any future consumer (outbound webhooks) read the same feed." The feed now
serves the desktop, the mobile shell and Web Push (ADR-0047); the durable
`events` log has grown a rich vocabulary (agent, inbox, run, task,
automation, terminal, cli, docker, session, setting) — including state the
owner cannot see without a browser open: a run finishing, an inbox item
needing a decision, a Docker job failing.

Today an external consumer has exactly three options, all weaker than the
log: poll the REST API, hold an SSE connection open, or use the one
per-automation notify URL (ADR-0045, a single fixed event type scoped to
one automation). None of them lets a Slack bot, n8n flow, home-automation
rule or phone shortcut react to *any* event.

The daemon is single-user per member (ADR-0051/0052 gateway routes to
per-member daemons), so a subscription belongs to the daemon's owner —
there is no cross-user event filtering to get wrong.

## Decision

Every enabled **webhook subscription** (one registered HTTPS or HTTP URL,
stored in `webhook_subscriptions`, migration 028) receives a POST for each
durable event whose type matches one of the subscription's type prefixes.

- **Delivery tails the log**, not the live feed: the deliverer reads
  `ListEventsSince(cursor)` per subscription and advances the stored
  cursor only after a 2xx — at-least-once, in order, per subscription.
  Ephemeral events (presence, `agent.tui`, `git.updated`, `agent.state`)
  are never delivered — they are lossy by design.
- **One POST per event**, body = the event JSON itself (`id`, `type`,
  `agentId`, `workspaceId`, `data`, `createdAt`), signed with
  `X-Picode-Signature: sha256=HMAC-SHA256(secret, body)` plus
  `X-Picode-Event-Id` and `X-Picode-Event-Type` headers. The secret is
  generated at creation and shown once; it is stored raw (same trust as
  push keys in migration 018) because deliveries sign, not compare, and
  never marshals into an API response.
- **Failure handling**: exponential backoff per subscription, 1 minute to
  1 hour cap, indefinitely — the subscription never disables itself; the
  owner sees `lastError` and disables it. A cursor older than the 7-day
  retention jumps to the newest event and records `missed events
  (retention)` — the same semantics as the feed's `ErrReset`.
- **Type prefixes are mandatory at the UI** (the store accepts an empty
  list as "all", but the Settings UI only writes explicit lists).
  Chatty durable families (`docker.health`, `terminal.state`) must be
  opted into explicitly. Events prefixed `webhook.` are never delivered —
  a subscription cannot spam itself.
- **New subscriptions start at the newest event**, not at the beginning
  of the retained log.
- **API** (mirrors `/api/push/*`): `GET/POST /api/webhooks`,
  `PATCH/DELETE /api/webhooks/{id}`, `POST /api/webhooks/{id}/test`
  (delivers a synthetic ping event through the normal path).
- **UI**: a Webhooks section in Preferences, mirroring the push
  preferences — list with empty state, create dialog (URL + type
  prefixes), per-row enable switch, last-delivery status, Test and
  Remove.

## Consequences

- External integrations stop polling; they receive events in near real
  time and can verify authenticity from the signature. The ADR-0056
  sensors and the Docker monitor make this more valuable: "the guest CLI
  finished", "a Docker job failed" are now deliverable facts.
- The daemon gains an **outbound network egress surface**: it POSTs to
  owner-chosen URLs. On a per-member daemon this is self-inflicted by
  definition; the secret signs so receivers can tell real events from
  forgeries. URLs are never logged with query credentials (v1: logged
  without query string).
- A slow or dead receiver costs one buffered cursor per subscription and
  a capped backoff — it cannot slow other subscriptions or the daemon
  (delivery is a background worker with its own tick).
- The durable log is the only delivery source, so anything it prunes at
  7 days is unreachable to a subscription that waited longer — recorded
  as `missed events`, never silently.
- `webhook_subscriptions` bookkeeping (cursor, last status) deliberately
  appends **no** events — one row per delivery would flood the log the
  feature reads.

## Alternatives considered

- **Proxy the SSE stream out** (an SSE-to-HTTP bridge): same payload, but
  external services speak webhooks, not SSE; reconnect/cursor semantics
  would be re-implemented per consumer.
- **Reuse the automation notify URL**: it exists (ADR-0045) but carries
  one fixed payload type scoped to one automation — generalizing it would
  overload an automation with integration duties it does not own.
- **Ping-only invalidation** ("something changed, fetch the API"):
  pushes the poll problem to the receiver and defeats the durable log
  that ADR-0048 was chosen for.
