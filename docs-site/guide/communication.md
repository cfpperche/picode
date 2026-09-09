# Messages between conversations

Open **Agent CLIs → Messages** on desktop or mobile. The selector includes
managed Pi agents and terminal CLIs. Messaging starts disabled.

1. Choose an agent or terminal with a recorded conversation. If its identity is
   still unknown, open a conversation first and refresh this page.
2. Select **Enable messages**. For supported CLIs, PiCode prepares private
   setup for this recorded conversation. No credential copying is needed.
3. Finish the current turn, then stop and resume **that conversation**. For
   a managed Pi agent, start it again; for a terminal, use **Resume**. A fresh
   terminal start intentionally receives no connection from the previous session.
4. Enable another conversation in the same workspace and resume it too.
   Ask either agent to list contacts, send a message and consult its inbox.

Setup never interrupts a running conversation or triggers a model turn. PiCode
keeps launcher credentials in private files under its data directory; only hashes
are kept in the database. Disabling a connection invalidates its credential.

## CLI compatibility

| CLI | Automatic setup on resume | Native configuration |
|---|---|---|
| Pi (managed or terminal) | Requires installed `pi-mcp-adapter` 2.32.1 or later | In-memory server registration for this conversation |
| Claude Code | Available | Private process-specific MCP file |
| Codex | Available | Process-specific overrides and bearer environment variable |
| OpenCode | Available | Process-specific inline MCP configuration |
| Grok | Unavailable in tested 1.0.24 | No verified per-launch MCP override |
| Hermes Agent | Unavailable in tested installation | Config loader ties MCP configuration to its shared home |

For Pi, install `pi-mcp-adapter` from **Agent CLIs → Packages** if needed.
These adapters preserve native settings and permissions. A configured connection
is not proof that a running model has loaded its tools; resume it and ask it to
list contacts. Grok and Hermes retain manual setup for clients with a verified
conversation-specific configuration mechanism. Never put a conversation bearer
in a global or shared configuration. The copied JSON describes an HTTP MCP
endpoint; adapt it to the client's native schema.

Use your normal trusted HTTPS address remotely; do not disable certificate
verification to make a client connect. Automatic setup uses the server's local
address and does not configure remote processes. For PiCode self-signed or mkcert
certificates, setup gives the client a private CA bundle while preserving its
existing configured CA certificates. If the server certificate or CA changes,
replace the connection before resuming. Native CLI tool approvals still apply.

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

OpenCode keeps existing inline JSON settings and other MCP servers when adding
communication. If `OPENCODE_CONFIG_CONTENT` contains JSONC comments or invalid
JSON, convert it to a JSON object before resuming; PiCode refuses to discard it.


## Verified native conversations

Validation on 2026-09-09 covered messages, replies and acknowledgments after
resuming each terminal's own recorded conversation: Claude Code 2.1.266 with
Codex 0.153.4, and OpenCode 1.18.29 with Claude Code. OpenCode used
`zai/glm-5.3-flash` with variant `max`. Each turn was explicitly prompted;
recipients were not started automatically. These are tested combinations,
not a guarantee that every provider or model can call the tools.
