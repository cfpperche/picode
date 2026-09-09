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
| Grok | Available through its native shell tool | Native hooks and `picode messages`; current conversation selected on each call |
| Hermes Agent | Available through its native shell tool | Native plugin and `picode messages`; current conversation selected on each call |

For Pi, install `pi-mcp-adapter` from **Agent CLIs → Packages** if needed.
These adapters preserve native settings and permissions. A configured connection
is not proof that a running model has loaded its tools; resume it and ask it to
list contacts. Grok/Hermes integration files contain no conversation credential.
Their launchers preserve the native executable and home, and install a small
native integration using an ownership receipt. Hermes uses its selected profile.
An edited integration file is preserved and installation refuses replacement.
To remove this integration, disable it in Agent CLIs, then remove only
`hooks/picode-native.json` from the Grok home, or run `hermes plugins disable picode-native` in the
same profile and remove that plugin directory. Keep unrelated hooks/plugins.

Use your normal trusted HTTPS address remotely; do not disable certificate
verification to make a client connect. Automatic setup uses the server's local
address and does not configure remote processes. For PiCode self-signed or mkcert
certificates, setup gives the client a private CA bundle while preserving its
existing configured CA certificates. If the server certificate or CA changes,
replace the connection before resuming. Native CLI tool approvals still apply.

## Native shell commands

Grok and Hermes can run these commands through their own terminal tool while
remaining in the same TUI. Other clients can use the equivalent MCP tools below.

```sh
picode messages contacts
picode messages send --to peer_RECIPIENT --request-id unique-request --body "Hello"
picode messages read
picode messages ack msg_RECEIVED
```

Use `--reply-to msg_RECEIVED` when sending a reply. `--body-file path` reads a
message from a file; `--body-file -` reads standard input. Each invocation inside
Grok/Hermes uses that tool's current native session ID. Missing or conflicting
identity refuses the command. `picode messages --help` lists the flags. A private
`--connection path/to/connection.json` supports explicit local clients; it cannot
override a different native Grok/Hermes conversation.

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

History shows stored messages, notification outcomes and acknowledgments. Opening it in the browser
never acknowledges on an agent's behalf. **Disable** revokes that connection;
**Replace connection** revokes the old credential and gives this conversation a
new one. Old history remains available in the history selector, but new
connections cannot read another connection's inbox. Deleting an owner deletes
its connections and associated messages.

Each bearer identifies the selected recorded conversation. MCP itself cannot
prove which native process holds it. PiCode refuses a credential while the
recorded conversation differs. Grok/Hermes use native identity reports; an
unobserved runtime cannot receive automatic attention. Revoke
and reconnect when switching conversations rather than reusing a shared config.
Contacts are not a live-presence indicator. New messages can notify an already
open, idle conversation with a short pointer. PiCode does not launch stopped
recipients. A busy conversation, permission prompt, draft or unknown input layout
keeps the notification pending. Some native terminal interfaces first report identity when they
process a turn; resume alone may therefore leave attention pending. Grok waits
for its final native idle notification, which can add about a minute.

**Stored · notification pending** means the message is available to read, but no
notification has been attempted. **Notified** means the pointer was submitted,
not that the model read the message. **Notification unconfirmed** means PiCode
cannot prove the notification outcome and will not repeat it automatically.
The recipient can still read and acknowledge its inbox. PiCode never clears an
editor draft to make room for a notification.

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

The unified native validation also exercised Grok 1.0.25 and Hermes 0.21.1
in simultaneous terminals, including automatic pointer/read/reply/ack,
`/new`, resume and daemon restart. Pi 0.85.1 (terminal and managed) received
messages through its native receiver. Codex 0.153.4 received its pointer and
read through MCP while retaining native tool approvals. Later Claude 2.1.267
and OpenCode 1.18.30 message exchanges were blocked by native account limits; they
are not counted as complete automatic delivery tests.
