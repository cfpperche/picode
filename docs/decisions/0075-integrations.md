# ADR-0075: Integrations — outbound events and externally implemented connectors

- **Status**: accepted
- **Date**: 2026-09-05
- **Builds on**: 0010, 0036, 0048, 0072

## Context

The owner approved a dedicated Integrations route, generic outbound webhooks
in core, and optional connectors implemented outside core. The old app host's
first-party/in-binary restriction is not a permanent product boundary. We use
existing MCP packages and configuration first; extending the app host is an
option when a concrete connector needs it, not a prerequisite for delivery.

Benchmarks: Cursor's compact, keyboard-reachable surfaces and t3code's
reload-safe routes (`../benchmarks/2026-08-24-adopt-t3code-paseo-cursor.md`);
Zapier's actionable connection status and explicit removal consequences
(`../benchmarks/2026-09-03-providers-view-v2.md`). The owner's ChatGPT/Grok/Claude
screenshots show discoverable service catalogs, not just transport settings.
We adopt that separation of service discovery and connected state without
building a marketplace or implying that webhook delivery grants agent tools.

## Decision

`#/integrations/webhooks` manages outbound event subscriptions;
`#/integrations/connectors` exposes the existing MCP manager as the first
connector implementation. Desktop and mobile own their presentation. Native
MCP config remains authoritative, with no second connection database or vault.
Custom remote/local MCP servers remain installable without recompiling PiCode;
provider-specific execution lives in those external servers/packages. Existing
MCP access remains compatible. Agent capabilities remain Pi-only today.

Webhooks are a stdlib background service over the durable event log:

- One signed JSON event per POST, ordered per subscription, at least once
  within retained history. New subscriptions start at the latest event.
- Explicit nonempty type-prefix selection; `webhook.*` never leaves the daemon.
  Cursor advances over unmatched events without generating another event.
- `X-Picode-Event-Id`, `X-Picode-Event-Type`, `X-Picode-Timestamp`, and
  `X-Picode-Signature: sha256=<hex>`; HMAC-SHA256 input is timestamp + `.` +
  exact body bytes. Secret is the generated string used as UTF-8 key, shown
  only on creation/rotation; receivers check timestamp tolerance and dedupe IDs.
- Retry every non-2xx/transport failure with persisted exponential backoff
  (one minute to one hour). Ten-second requests and bounded concurrency.
  Pausing stops new deliveries, preserving cursor/backlog. Configuration
  changes invalidate stale acknowledgements. An already-issued POST cannot
  be recalled by edit, rotation, pause or removal.
- A retention gap advances to the current end and records an explicit missed
  history warning. Retention means this is not an unlimited delivery guarantee.
- HTTP and HTTPS owner-selected destinations, including LAN/loopback, are
  allowed; no redirects or ambient proxy credentials. Reject URL userinfo,
  fragments and link-local/metadata destinations. HTTP exposes payloads on
  the network; the UI warns. No raw URL or receiver response in audit/errors.
- CRUD and delivery-attempt events commit with their mutations. Scan-only
  cursor bookkeeping deliberately does not append another event, preventing
  a self-sustaining log loop. Signing keys never enter events or normal JSON.
- Test sends a synthetic `webhook.test` using the same signing/HTTP path;
  it does not move the log cursor or clear a failed real delivery's backoff.

## Consequences

Users gain a real external delivery path without adopting an automation host.
The core owns generic lifecycle and UI; optional MCP servers own vendor APIs.
The existing adapter's machine/server-name credentials are not claimed to be
per-agent isolation. Local packages run with user permissions, not a sandbox.
A catalog entry is not proof of successful authentication or live tool access.

Provider-specific OAuth and API maintenance remains with connector maintainers;
a new generic capability can justify a core change. No provider switch in Go,
new plugin runtime, iframe marketplace, or parallel package manager is required.
Future non-MCP connectors may extend ADR-0036 based on an actual working case.

## Alternatives considered

- Preferences placement: rejected by the owner; integrations deserve a route.
- New bespoke connector runtime: adds installation/auth/config duplication
  before the existing external MCP path has been exercised.
- Webhooks as a substitute for tools: incorrect; notifications do not grant
  an agent read/write access to a service.
- In-process vendor adapters: forces binary releases for vendor API changes.
