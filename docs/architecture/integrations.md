# Integrations (ADR-0075)

> Part of [PiCode's architecture](../architecture.md) (ADR-0105: one file per subsystem). Edit here; the index only links.

The core owns `#/integrations`, subscription lifecycle and generic outbound
HTTP delivery. Vendor tools remain external MCP servers or optional Pi
packages; there is no in-process vendor adapter, second package manager or
credential database. `packages/pi-connector-deepwiki` exercises the existing
adapter's `pi.mcp` manifest contract (verified with 2.32.1). The UI reports
package installation separately from configured/live services. Native package
settings own removal; already-running agents may need a restart. The original
`#/mcps` route stays compatible. File-config changes emit a credential-free
`mcp.config` invalidation through the feed, not a copy of native config.

`internal/webhooks.Engine` runs with the daemon context, four requests at most
in parallel, one in flight per subscription. Migration 028 stores filters,
secret, cursor, revision, failures and next attempt. CRUD/rotation and attempt
status events commit transactionally. Revision and starting cursor guard
acknowledgements against concurrent edits; an issued HTTP request cannot be
recalled. Scan-only cursor advancement deliberately emits no event, avoiding
self-sustaining feedback from skipped `webhook.*` rows.

Each matching durable event becomes one POST with event ID/type, Unix timestamp
and HMAC-SHA256 of `timestamp + "." + body`. Retries are ordered, at least once
within retention, and back off from one minute to one hour; expiry records a
missed-history warning and resumes at the current end. New subscriptions start
at the current end. Tests use the same HTTP/signing path with event ID zero but
never advance the cursor or clear real retry state. Configuration changes clear
previous attempt status without rewinding the cursor. Secrets are returned only
on creation/rotation and excluded from normal JSON and event payloads.

Requests have a ten-second bound, no redirects and no ambient proxy. Owner-chosen
HTTP/LAN/loopback destinations are supported; URL userinfo/fragments and
link-local/metadata addresses are refused, including at DNS dial time. No raw
receiver response or credential-bearing URL enters delivery errors or audit.
This is outbound data disclosure to an owner-selected service, not agent tool
access. The ordinary device gate protects all `/api/webhooks` CRUD/test/secret
routes. See [acceptance tables](../plans/integrations.md) and the
[public guide](../docs-site/guide/integrations.md).
