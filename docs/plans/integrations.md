# Integrations implementation and acceptance

Owner-approved direction: ADR-0075. Core owns event delivery and management;
external MCP services/packages own service-specific tools. Decisions about the
app host remain revisable. No new runtime or marketplace is needed for the
first working connectors.

## Implementation

| Area | Implementation |
|---|---|
| Storage | Migration 028; transactional CRUD, secret rotation, revision/cursor guarded acknowledgements |
| Delivery | `internal/webhooks`; stdlib HTTP/HMAC, durable cursor/backoff, four concurrent workers |
| UI | Independent desktop/mobile Integrations routes, Webhooks and Connectors |
| External connectors | Native MCP definition import; optional `packages/pi-connector-deepwiki` uses the adapter's `pi.mcp` contract |
| Package management | Existing Packages surface; installation metadata shown separately from configured/live services |

Benchmark adaptation: Cursor density and keyboard reachability; t3code
reload-safe routes; Zapier status plus next action and explicit removal
consequences. User-supplied ChatGPT/Claude/Grok screenshots motivated a service
catalog rather than requiring everyone to start with a transport form.

## Webhook decision table

| Conditions | Action | Automated coverage |
|---|---|---|
| Valid HTTP/HTTPS + explicit prefixes | Create/edit; new cursor at current end | `TestWebhookCRUD`, `TestWebhookValidation`, `TestWebhookAPI` |
| Invalid URL, metadata address, empty/internal/wildcard prefixes | Reject before mutation | `TestWebhookValidation` |
| Event append fails during mutation | Roll back mutation and secret/cursor change | `TestWebhookTransactionRollback` |
| Enabled + matching event + 2xx | Sign and POST; acknowledge once | `TestDeliveryDecisionTable/success` |
| Enabled + 401/429/5xx/redirect | Keep event, persist bounded backoff | `TestDeliveryDecisionTable` |
| Network failure | Retry; do not persist URL/query/response credentials | `TestNetworkFailureDoesNotLeakURL` |
| Paused | Do not send or discard backlog | `TestDeliveryDecisionTable/paused` |
| Prefix mismatch or internal webhook event | Scan forward with no event feedback loop | `TestDeliveryDecisionTable/filtered`, `TestBackoffAndFilters` |
| Restart before retry deadline | Honor stored deadline | `TestDeliveryDecisionTable` |
| Failure then success then next event | Retry identical ID before next ID | `TestRetryKeepsEventOrder` |
| Retention gap | Record missed history, resume at current end | `TestRetentionGapAndTestIsolation` |
| Edit/pause/rotate/delete/another acknowledgement during POST | Old acknowledgement cannot update new state | `TestWebhookStaleDeliveryDecisionTable`, `TestEditWhilePostingInvalidatesAcknowledgement` |
| Explicit test, including paused | Same signer/HTTP; no cursor/backoff change | `TestRetentionGapAndTestIsolation`, `TestWebhookAPI` |
| Concurrent test and worker | One request in flight for this subscription | `TestEditWhilePostingInvalidatesAcknowledgement` |
| Missing engine, receiver rejects test, stale edit | Visible unavailable/failure/conflict, never fake success | `TestWebhookTestFailureAndUnavailable`, `TestWebhookAPI` |

## Connector decision table

| Conditions | Action | Coverage |
|---|---|---|
| Adapter absent | Show one installation action | Desktop/mobile blocked browser review |
| One standard remote definition | Review destination/scope, add via existing MCP API | `integrations.test.js`; real DeepWiki import browser check |
| One local command definition | Review literal command/system permissions; preserve argument vector | `integrations.test.js` |
| Malformed/multiple/unsupported/oversized definitions | Refuse without silently dropping settings | `integrations.test.js` |
| Remote credential command in imported file | Refuse implicit code execution | `integrations.test.js` |
| Installed package declares `pi.mcp` | Show installed metadata; adapter owns runtime discovery | `TestConnectorPackageDiscovery`; adapter 2.32.1 live config/protocol check |
| Ordinary/invalid package or removed native setting | No connector-package row | `TestConnectorPackageDiscovery` |
| Package installed but no agent tool invocation | Say Installed, not Connected | Separate package rows; no model-turn claim |

## Live acceptance scope

Use disposable HOME/data and a local HTTP receiver; never configure test
webhooks against the production daemon. Receiver receipts must prove a real
durable event, not only `webhook.test`. Verify signature bytes independently.
For the external connector, native adapter config discovery and a real MCP
`read_wiki_structure` call against public `golang/go` validate the package and
remote protocol without spending a model turn or using owner credentials.

### Observed acceptance — 2026-09-05

- Isolated daemon/receiver on ports 18575/18576; 11 HTTP receipts included
  durable events, explicit tests and retries. Durable event 19 retried the same
  ID after roughly 61/121/241/481 seconds and succeeded after correcting its URL.
  Fresh `inbox.created` event 33 passed an independent Python HMAC verification.
- Browser creation, editing, pause, paused test, secret replacement and removal
  passed. DeepWiki definition import was reviewed and saved through the UI.
- Adapter 2.32.1 discovered `pi-connector-deepwiki__docs`; the package's
  `verify.mjs` listed three tools and successfully read `golang/go` wiki structure.
  This used public remote MCP, not a mocked response or a model turn.
- `make ci` and targeted store/webhooks/server/MCP/URL-policy `-race` tests passed.
  Curated `docs/screenshots/integrations-*` cover empty, blocked, validation,
  retry, catalogs and mobile editing. Screenshots were read; final audits passed.
  Fixtures were disposable; no production configuration was changed.

OAuth-provider matrices and model-driven connector usage are separate acceptance
work, not certified by a public/no-auth connector check. Updating the adapter,
revoking credentials and restarting already-running agents remain explicit user
operations. A package installation is not a security sandbox.
