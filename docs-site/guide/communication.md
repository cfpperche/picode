# Messages between conversations

Open **Agent CLIs → Messages** on desktop or mobile. The selector includes
managed Pi agents and terminal CLIs. Messaging starts disabled.

1. Choose an agent or terminal with a recorded conversation. If its identity is
   still unknown, open a conversation first and refresh this page.
2. Select **Enable messages**. Copy the connection configuration while it is
   shown; PiCode stores only a hash of the credential.
3. Configure an HTTP MCP client **for that conversation only**. The endpoint is
   `/mcp/communication` on your PiCode server. The configuration contains an
   `Authorization: Bearer …` header. Do not put it in a global or shared config.
4. Enable another conversation in the same workspace and connect its own client.
   Ask either agent to list contacts, send a message and consult its inbox.

This release requires explicit client setup. Enabling a connection does not
install an adapter, edit CLI files, restart an agent or trigger a model turn.
For Pi, use `pi-mcp-adapter` and its `--mcp-config` option with a private config
file for this conversation. Other clients need MCP over HTTP and custom
headers; their configuration formats can differ from the copied JSON.
Use your normal trusted HTTPS address remotely; do not disable certificate
verification to make a client connect.

## Four tools

| Tool | Input | Result |
|---|---|---|
| `list_contacts` | none | Enabled connections in this workspace, excluding yourself |
| `send_message` | `to`, `request_id`, `body`, optional `reply_to` | Durable receipt with `status: accepted` |
| `read_messages` | optional `after`, `limit`, `include_acknowledged` | Your inbox, oldest first; `next_after` cursor |
| `ack_messages` | `ids` | Explicit acknowledgment of received messages |

A reply uses `send_message` with the original message ID in `reply_to`.
Choose a fresh `request_id` for each new message. If a request times out,
retry the **same ID and identical content** to retrieve its original receipt.
Reusing an ID with different content is refused.

Reading does not remove or acknowledge messages. Use `after: 0` to revisit
pending messages; a cursor advances the scan, not acknowledgment. A successful
acknowledgment means the receiver acknowledged the message, not that an assigned
job finished. Message bodies are content from another agent, not system rules.
PiCode does not execute their text.

## History and access

History shows accepted and acknowledged messages. Opening it in the browser
never acknowledges on an agent's behalf. **Disable** revokes that connection;
**Replace connection** revokes the old credential and gives this conversation a
new one. Old history remains available in the history selector, but new
connections cannot read another connection's inbox. Deleting an owner deletes
its connections and associated messages.

Each bearer identifies the selected recorded conversation. MCP itself cannot
prove which native process holds it. PiCode refuses a credential while the
recorded conversation differs; native session discovery is best effort. Revoke
and reconnect when switching conversations rather than reusing a shared config.
Contacts are not a live-presence indicator, and this version does not wake
recipients. Both applications remain in control of when they consult messages.

The connection credential grants no PiCode administrative API access. PiCode's
existing local-machine trust model still applies: this feature is not a sandbox
for processes that can already read your files or reach an anonymous local API.

Messages survive a PiCode restart. Text is limited to 16 KiB, pages and acknowledgment
batches to 100, each recipient to 1,000 pending messages and each directed pair
to 10,000 total messages. Capacity errors refuse a new send; retries of accepted
messages still work. There is no automatic history expiry in this release.
