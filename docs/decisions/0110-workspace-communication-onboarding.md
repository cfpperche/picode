# ADR-0110: Workspace communication preferences and guided connection setup

- **Status**: accepted (owner approval, 2026-09-09)
- **Date**: 2026-09-09
- **Boundary**: persistence and security model — owner participation spans future conversations; each conversation keeps a separately revocable credential. Explicit owner diagnostics can request a native exchange.

## Context

ADR-0107 delivered the transport but required operators to understand native
session identity, enable each connection, stop and resume, and type diagnostic
commands. The owner approved selecting workspace participants and letting PiCode
prepare their connections, with advanced setup hidden and a real exchange test.

## Decision

Amend ADR-0104/0106/0107: store explicit participant consent by owner, workspace,
CLI and revision. Consent survives a new conversation; credentials do not. A
validated native identity creates an idempotent setup for that conversation.
Missing identity never becomes a guessed address. Moving to another workspace
or CLI does not transfer consent. Disabling consent and revoking capabilities
commit together. Stale selection and worker writes fail revision checks.

Preparation shares the existing attention worker's feed and local-input tick.
Grok/Hermes discover the current setup on each native shell call. Pi's receiver
registers the prepared server through its adapter while idle. Other native MCP
clients resume their exact recorded conversation after preparation, with the
existing lifecycle/editor gates and verified process exit. Stopped participants
require the explicit Open and connect action. Setup never sends a model prompt.

The owner may request a connection test between two current workspace addresses.
A short guarded native prompt asks the sender to read its inbox. The existing
read tool also returns the owner-requested diagnostic's recipient and retry key.
The native sender sends, the recipient replies, and both acknowledge. Only that
correlated exchange passes the test. Owner handlers never impersonate native
senders or acknowledge their messages. Ambiguous submission is never retried.
Tests expire rather than claim success after a model or permission failure.

## Consequences

The normal interface shows participants, preparation, test and activity. Advanced
per-conversation setup remains available. There is no new dependency, daemon,
or orchestration engine. Applying a connection can resume an idle native terminal;
process/input checks still cannot make the PTY boundary atomic. Unknown input
layouts, permissions and drafts block the operation visibly. Non-Linux automatic
terminal replacement remains unavailable until process ownership is verified there.

## Alternatives considered

- Workspace-wide bearer: loses per-conversation isolation and revocation.
- Automatic test ACK from the backend: proves only the backend, not native use.
- Generic terminal restart: can create a new conversation or overlap an old writer.
- Copying a scheduler: unnecessary for a bounded connection preparation pass.
