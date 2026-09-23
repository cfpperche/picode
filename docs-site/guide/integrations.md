---
description: Connect MCP servers on a CLI and PiCode webhooks. Neither changes your provider account.
---

# Integrations

Connectors live on the selected CLI in **Agent CLIs**. Webhooks stay under
PiCode (**Webhooks** in the user menu).

- **Where:** **Agent CLIs → Connectors** for MCP; user menu → **Webhooks** for incoming URLs.
- **Not this:** not [Providers](/guide/providers) (API keys). Connector details: [MCP](/guide/mcp).

## Connect a service

1. Open **Agent CLIs**, pick the CLI, then **Connectors**.
2. On Pi, if prompted, open **Packages** and install `npm:pi-mcp-adapter`.
3. In **Marketplace**, search the catalog and press **Add** on a card.
   PiCode's own connectors sit at the top; the rest is a curated slice of
   the official MCP Registry. Set **Save to** (Global, a workspace or an
   agent) before adding. **Custom server…** opens the server form (URL or
   local command) for anything the catalog does not list.
4. Sign in when the service requires it. Existing authentication and server
   support come from the installed adapter, not from a separate PiCode vault.

Choose **Global**, a workspace, or an available agent-specific folder.
These are configuration scopes, not a security sandbox. Some credentials are
shared by server name on the machine. Desktop changes can reload the selected
running agent; other running agents may need a restart to reload configuration.

Shipped connector pages: [Gmail](/guide/mcp-gmail) and [DeepWiki](/guide/mcp-deepwiki). How to add any server: [MCP](/guide/mcp).

Only add connectors you trust. A local command runs with the agent's system
permissions and adding it does not install its dependencies — install those
separately, following the server's maintainer. Remote servers cannot embed
executable credential commands; configure those explicitly in MCP settings
if needed. Removing configuration does not revoke a provider credential. Use
**Sign out** where available and revoke tokens at the service.

| Capability | Compatibility |
|---|---|
| Package install/remove and scope | Same as [Pi packages](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/packages.md) |
| MCP configuration and execution | Native in every CLI but Pi; on Pi, provided by [pi-mcp-adapter](https://github.com/nicobailon/pi-mcp-adapter) |
| Integrations UI | PiCode-specific; native configuration remains authoritative |
| Supported CLIs | All nine; each Connectors pane writes that CLI's own configuration file |

## Add an outbound webhook

1. Open **Integrations → Webhooks → Add webhook**.
2. Enter a receiver URL and select events. Prefixes such as `agent.` match
   that event family. Multiple prefixes are comma-separated in the form.
3. Create the webhook and copy its signing secret to your receiver.
4. Use **Send test**. Success means the receiver returned a 2xx response,
   not that its downstream workflow completed.

Events may contain project details and messages. Choose destinations and event
families deliberately. HTTPS encrypts delivery. HTTP is supported for trusted
local networks but exposes payloads in transit. The receiver must understand
PiCode event JSON: a Slack incoming-webhook URL does not automatically translate
it into a Slack message. Use a receiver or adapter that performs that mapping.

The secret appears only on creation or replacement. **Replace secret** invalidates
the old key for future requests immediately; update your receiver accordingly.
An already-issued request can still carry the previous key.

### Delivery behavior

- Only new, durable events are delivered. Ephemeral presence/terminal notices
  and internal `webhook.*` events are excluded.
- Each subscription delivers in order, with possible duplicates. Persist an
  event ID before applying the same effect again.
- Non-2xx responses and connection failures retry after one minute, doubling
  to a one-hour cap. Retry state survives daemon restart. There is no automatic
  disable on failure.
- **Off** pauses future requests and keeps the backlog. **On** resumes it.
  If retained events have expired (normally after seven days), PiCode reports
  missed history and resumes with new events. There is no unlimited retention
  guarantee.
- Editing configuration or replacing a secret clears the previous delivery
  status and makes pending work eligible again. Neither rewinds the cursor.
- Removing a webhook stops future deliveries; an in-flight POST cannot be
  recalled. Requests time out after ten seconds; redirects are not followed.
- **Send test** works while paused. It uses a synthetic `webhook.test` event
  with ID `0` and does not advance the cursor or clear a real retry schedule.

### Verify the signature

The request carries:

| Header | Value |
|---|---|
| `X-Picode-Event-Id` | Durable event ID; `0` for a synthetic test |
| `X-Picode-Event-Type` | Event type |
| `X-Picode-Timestamp` | Unix seconds for this attempt |
| `X-Picode-Signature` | `sha256=` followed by a hexadecimal HMAC-SHA256 |

The signed bytes are `timestamp + "." + raw request body`. Use the secret
string as the UTF-8 HMAC key, not as decoded base64. Compare signatures in
constant time and reject timestamps outside your tolerance, such as five
minutes. Verify before decoding or acting on the payload. Track duplicate durable
event IDs separately; signatures change when the same event is retried.

### HTTP API

All routes use PiCode's ordinary authentication gate. No new public inbound
endpoint is exposed by this feature.

| Route | Purpose |
|---|---|
| `GET /api/webhooks` | List subscriptions without secrets |
| `POST /api/webhooks` | Create with `{url, types: ["agent."]}`; returns `{webhook, secret}` |
| `PATCH /api/webhooks/{id}` | Update `url`, `types`, or `enabled`; include the current `revision` |
| `DELETE /api/webhooks/{id}` | Remove |
| `POST /api/webhooks/{id}/secret` | Replace secret; include `revision` |
| `POST /api/webhooks/{id}/test` | Send synthetic test; inspect `delivered`, `status`, and `error` |

A stale revision returns 409; reload before editing again. The service permits
up to 32 subscriptions and 32 prefixes per subscription. URLs with embedded usernames or passwords,
fragments, and metadata/link-local destinations are refused. LAN
and loopback receivers are supported; requests do not inherit proxy settings.
