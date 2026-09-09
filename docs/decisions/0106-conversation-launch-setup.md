# ADR-0106: Private communication setup on conversation resume

- **Status**: accepted (owner approval, 2026-09-09)
- **Date**: 2026-09-09
- **Boundary**: Persist conversation-scoped bearer credentials in private launcher files and attach them only when resuming that recorded conversation.

## Context

ADR-0104 delivered the mailbox with manual client configuration. The owner
approved completing launch integration and testing real CLI conversations.
Shared MCP configuration would give unrelated conversations the same identity.
The t3code/Paseo benchmark adaptation keeps runtime adapters small and preserves
native launch and resume behavior rather than introducing another agent engine.

## Decision

Extend ADR-0104 with explicit automatic setup from the Messages tab. The existing
store enrolls the recorded conversation; PiCode saves its bearer in a private
0700 directory with 0600 files under its own data directory. Database hashes
remain the authorization authority. Setup never starts or interrupts a process.
Managed Pi attaches setup on its next start; terminal launchers attach it only
when the effective arguments exactly match the recorded conversation's native
resume recipe. Fresh starts and forks do not inherit it. Pi's terminal recipe
uses its recorded JSONL path, including older pins without resume arguments.

Pi registers an in-memory server with the installed pi-mcp-adapter through its
runtime registration API; Claude Code receives a private MCP file; Codex gets
process-local configuration overrides and a bearer environment variable;
OpenCode gets inline process-local MCP configuration. Pi checks its native
conversation before registration and disposes registration before switching.
Guest native session attribution otherwise retains ADR-0084's best-effort limit.
No global/project config writes, home overlays, background model turns or new
dependencies. HTTPS setup verifies the local certificate against its self-signed
anchor or mkcert public CA, then gives the child a private additive CA bundle
through NODE_EXTRA_CA_CERTS or CODEX_CA_CERTIFICATE. Existing configured CA
bundles are preserved at setup time. TLS and hostname verification stay enabled;
no root is installed globally and no private certificate key is read.

## Consequences

The secret must now survive in a private launcher file to support resume. Same-user
process trust is unchanged; a private file is not process attestation. Credentials
are absent from diagnostic snapshots, feed events and HTTP automatic-setup
responses. Revocation invalidates them before file cleanup. A file-write failure
revokes the new connection; replacement does not resurrect the previous token.
Missing setup is recoverable by replacing a connection. Files left by owner deletion
cannot authenticate; cleanup of those orphaned files is a maintenance concern.

Grok 1.0.24 exposes no per-launch config override in its help. Hermes' installed
config loader resolves config.yaml exclusively from HERMES_HOME. Their automatic
setup remains unavailable until a supported local override is verified; this is
measured compatibility, not a permanent refusal. Manual capability setup remains
available, but must never put one conversation's bearer in shared config.

## Alternatives considered

- Shared config or separate CLI homes: identity leakage or native session/auth drift.
- A polling worker that starts recipients: message transport becomes orchestration.
- Copying a token into every launch: a fresh conversation inherits another identity.
- Reimplementing Pi's MCP client: duplicates the installed adapter.

Acceptance and compatibility: [plan](../plans/peer-communication.md).
