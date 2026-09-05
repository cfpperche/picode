# Integrations

Open **Integrations** from the desktop user menu or command palette, or from
**More** on your phone. Connectors give agents tools; webhooks send PiCode
events to another service. Neither changes your LLM provider account.

## Connect a service

1. Open **Integrations → Connectors**.
2. If prompted, open **Packages** and install `npm:pi-mcp-adapter`.
3. Choose a service, enter a custom server, or import a JSON definition
   containing one `mcpServers` entry. Review the destination and scope before
   confirming **Add connector**.
4. Sign in when the service requires it. Existing authentication and server
   support come from the installed adapter, not from a separate PiCode vault.

Choose **This machine**, a workspace, or an available agent-specific folder.
These are configuration scopes, not a security sandbox. Some credentials are
shared by server name on the machine. Desktop changes can reload the selected
running agent; other running agents may need a restart to reload configuration.

An optional local example lives at `packages/pi-connector-deepwiki` in the
repository. Install that directory through **Packages**. Its `pi.mcp` manifest
loads public GitHub documentation tools through the external DeepWiki service;
package-manifest loading was verified with adapter 2.32.1. It is not published
to npm. **Connector packages** reports installation, not successful sign-in or
live tool access. Use **Manage package** to remove it, then restart affected
agents. Alternatively, import `connectors/deepwiki.json` for a directly managed
connection; do not use both unless you want two configurations.Only import trusted definitions. Local commands run with the agent's system
permissions. Import does not install their dependencies. Remote definitions
cannot embed executable credential commands; configure those explicitly in
MCP settings if needed. Removing configuration does not revoke a provider
credential. Use **Sign out** where available and revoke tokens at the service.

| Capability | Compatibility |
|---|---|
| Package install/remove and scope | Same as [Pi packages](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/packages.md) |
| MCP configuration and execution | Provided by [pi-mcp-adapter](https://github.com/nicobailon/pi-mcp-adapter), not native Pi |
| Integrations UI | PiCode-specific; native configuration remains authoritative |
| Supported managed agents | Pi; terminal integrations do not imply connector support for other CLIs |

### Example: Gmail

Gmail has no official MCP server; the catalog features the community
`@gongrzhe/server-gmail-autoauth-mcp`. Its design keeps credentials **outside
PiCode**: nothing is stored in the connector configuration or the PiCode
database.

1. In [Google Cloud Console](https://console.cloud.google.com/), create a
   project, enable the **Gmail API**, and create an **OAuth client ID**
   (application type *Desktop app*). Download the client secret JSON.
2. From a terminal, prepare the one-time sign-in:

   ```bash
   mkdir -p ~/.gmail-mcp
   mv <downloaded-client-secret>.json ~/.gmail-mcp/gcp-oauth.keys.json
   npx -y @gongrzhe/server-gmail-autoauth-mcp auth
   ```

   A browser opens for Google consent; tokens land in `~/.gmail-mcp/`.
3. Add the connector through any one path: the **Gmail** card in the catalog,
   **Import a connector definition** with `connectors/gmail.json`, or install
   `packages/pi-connector-gmail` through **Packages**.
4. Restart the agent so the adapter loads the tools.

The connector can read, draft and send mail — grant it like you would grant a
delegate access. To revoke: remove the connector, delete `~/.gmail-mcp/`, and
revoke the application at your
[Google account permissions page](https://myaccount.google.com/permissions).

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
