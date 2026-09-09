# Direct session communication (ADR-0104, ADR-0106)

> Part of [PiCode's architecture](../architecture.md) (ADR-0105: one file per subsystem). Edit here; the index only links.

`internal/communication` serves four tools through the official Go MCP SDK at
`/mcp/communication` in the existing server process. Stateless Streamable HTTP
keeps protocol sessions out of identity. An explicit connection bearer is
mandatory in every auth mode; the main Host/Origin gate still applies and that
bearer is refused on owner APIs. This does not change local-process trust or
promote terminal CLIs to managed agents (ADR-0091).

Owner routes: `GET/POST /api/communication`, `DELETE /api/communication/{id}`,
`GET /api/communication/{id}/messages?before=`. The Messages tab at
`#/clis/messages[/<kind:ownerId>]` lives in each app's Agent CLIs frame and includes
managed agents. React state remains app-owned (ADR-0072); both listen for
`peer.*`, owner changes and feed reconnect/reset. Owner reads never acknowledge.

`peer_connections` binds a hashed credential to an agent's recorded session path
or a terminal's native session ID plus CLI and workspace. Every tool validates
that binding inside its store transaction. Discovery is same-workspace opt-in;
a bearer is a capability, not native-process attestation. Session discovery can
lag (ADR-0084), so credentials must be installed per conversation, never globally.
ADR-0106 adds explicit automatic setup for Pi, Claude Code, Codex and OpenCode.
Private launcher files retain the credential (0700 directories, 0600 files), while
SQLite retains its hash. Managed Pi attaches a session-checked adapter registration
on start; terminal launchers attach only to the exact native resume recipe. Tokens
never enter diagnostic snapshots or the shell returned to after a CLI exits.
Local HTTPS setup verifies the server certificate and adds its public self-signed
or mkcert CA to a private child trust bundle, preserving existing configured CAs.
No TLS bypass or global trust-store write is involved.
OpenCode inline JSON settings retain unrelated keys and MCP servers; malformed
or non-object inline config fails launch visibly. JSONC inline merging is not supported.
Grok and Hermes still need a verified per-process configuration surface. Setup
never changes native config files, installs packages, wakes agents or restarts them.

`peer_messages` is both durable inbox and history. Per-sender request IDs return
the same receipt on exact retry and fail on conflicting content. Read leaves
messages pending; ack validates the whole batch before writing. `peer.connection`,
`peer.message` and `peer.ack` events carry IDs only and commit with the mutation.
History remains after revoke/restart, cascades with owner deletion, and is bounded
by explicit capacity refusals. Details and decision table: [plan](../plans/peer-communication.md).
