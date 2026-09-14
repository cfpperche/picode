---
description: Connect agents in a workspace so they can talk.
---

# Communication between agents

Connect managed Pi agents and native Agent CLI terminals in a workspace so they can talk.

- **Where:** workspace menu → **Communication**, or **Agent CLIs → Messages** and choose a workspace.
- **Not this:** not the [Inbox](/guide/inbox-tools) (questions to you) and not moving a conversation between CLIs ([Agent CLIs](/guide/agent-clis)).

Desktop and mobile show the same participants.

1. Check the participants you want to connect and select **Apply and connect**.
2. If one is stopped, select **Open and connect**. Start a normal conversation
   if it does not have one yet. PiCode prepares its connection automatically.
3. Once two participants are connected, choose **Run test**. The agents exchange
   a message, reply and acknowledge receipt using their native tools.
4. Follow the result and message history in **Activity**. For everyday use, ask
   an agent to contact another participant by name from its usual conversation.

Selection applies to future conversations of that participant in this workspace.
Every conversation still receives its own credential; older credentials cannot
access the new conversation. Clear a participant’s checkbox and apply to disconnect it.
An older connection labeled **Current conversation only** remains scoped to that
conversation until you apply the persistent selection.

**Waiting to connect** means a turn, approval or draft needs to finish. PiCode
preserves native input and resumes only the exact idle conversation when a CLI
needs to reload its connection. Pi can load it directly through its connection
adapter. Codex, Grok and Hermes discover prepared setup on each native tool call
without restarting. Claude and OpenCode retain the panel dimensions when resuming.
An older managed Pi process may offer **Reconnect**; this waits for an
idle process and resumes its existing history. Setup does not start a model turn.

**Connected** describes setup readiness. **Test passed** requires the
native message, reply and both acknowledgments. A test uses the configured
models and may wait for their normal tool permissions. If it expires, open the
conversations to check approvals or model capacity, then run another test.
An unconfirmed submission is never retried automatically.

**Advanced** contains per-conversation replacement and manual configuration.
PiCode keeps private setup under its data directory and stores only credential
hashes in the database. Moving a participant to another workspace or changing
its CLI requires selection in the new context.

## CLI compatibility

| CLI | Connection preparation | Native configuration |
|---|---|---|
| Pi (managed or terminal) | Requires installed `pi-mcp-adapter` 2.32.1 or later | In-memory server registration for this conversation |
| Claude Code | Available | Private process-specific MCP file |
| Codex | Available through its native shell tool | Native hooks and `picode messages`; connects without restarting an identified conversation |
| OpenCode | Available | Process-specific inline MCP configuration |
| Grok | Available through its native shell tool | Native hooks and `picode messages`; current conversation selected on each call |
| Hermes Agent | Available through its native shell tool | Native plugin and `picode messages`; current conversation selected on each call |

For Pi, install `pi-mcp-adapter` from **Agent CLIs → Packages** if needed.
These adapters preserve native settings and permissions. Use **Run test** to confirm that both running agents can use their tools. Grok/Hermes integration files contain no conversation credential.
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
Codex shell calls need access to the local PiCode server: if its sandbox blocks
network access, approve that specific native tool request. Do not disable the
sandbox or certificate verification.

## Native shell commands

Grok, Hermes and Codex can run these commands through their own terminal tool while
remaining in the same TUI. Other clients can use the equivalent MCP tools below.
Codex introduces these commands through its native session-start hook. Older
versions without that hook require asking it to run `picode messages --help`.

```sh
picode messages contacts
picode messages send --to peer_RECIPIENT --request-id unique-request --body "Hello"
picode messages read
picode messages ack msg_RECEIVED
```

Use `--reply-to msg_RECEIVED` when sending a reply. `--body-file path` reads a
message from a file; `--body-file -` reads standard input. Each invocation inside
Grok/Hermes/Codex uses that tool's current native session ID. Missing or conflicting
identity refuses the command. `picode messages --help` lists the flags. A private
`--connection path/to/connection.json` supports explicit local clients; it cannot
override a different native Grok/Hermes/Codex conversation.

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

## Connection and activity

Each participant shows its activity (Idle, Working, or Needs your input) and its
connection separately. Syncing / Reconnecting means PiCode is recovering the
conversation identity; it does not mean the CLI is asking for approval. Open the
terminal if the conversation has not been identified. A fresh conversation needs
its first native event. Connection failed shows a setup problem and its action.

On Linux/WSL, observed conversations recover after the PiCode server restarts
while their terminals stay open. Terminals running an older integration begin
recording on their next native event. Pi receiver presence renews automatically;
PiCode does not restart a terminal just because its receiver has not reconnected.

After updating the Hermes integration, stop and resume its conversation once to
load the adapter. Its message calls then use the current native conversation even
if an internal background review changes Hermes' environment. Keep one
`picode messages` command per tool call; native permission prompts still apply.

Enabled participants stay in From and To while reconnecting, with their availability
shown. Run test becomes available when both selected conversations are connected.
Test passed records a completed exchange; it is separate from current activity.

If Messages shows **State unavailable / Connection failed**, state recording
could not recover. Restore storage access, then use **Terminal controls** to
restart the affected CLI. Opening the existing conversation or refreshing the
page cannot reset this persistent failure. Native approval and draft protections
remain in effect.
