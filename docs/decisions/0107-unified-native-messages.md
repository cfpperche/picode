# ADR-0107: One mailbox with CLI and native TUI integrations

- **Status**: accepted (owner approval, 2026-09-09)
- **Date**: 2026-09-09
- **Boundary**: protocol and security model — expose the existing mailbox to native shell tools, resolve conversation identity at invocation, and separate durable delivery from terminal attention.

## Context

The owner approved an Orca-inspired common communication mechanism while preserving
native TUIs in tmux. ACP adapters and private HOME overlays do not satisfy that goal.
A shared Grok leader can reuse an HTTP MCP bearer across conversations. Native Grok
shell tools and hooks expose a per-conversation GROK_SESSION_ID; Hermes exposes
HERMES_SESSION_ID to shell tools and session identity to native plugin hooks.
These are local capabilities and routing context, not hostile-process attestation.

## Decision

Extend ADR-0104 with `picode messages contacts|send|read|ack`, an MCP client of the
same four tools in the existing Go server. SQLite, opt-in, request deduplication,
revocation, acknowledgement and workspace isolation remain owned by that store.
There is no new daemon, agent engine or orchestration framework. Explicit private
connection files remain available to clients. Automatic Grok/Hermes lookup uses
the native session ID at each invocation, never an inherited terminal ID or bearer;
missing, mismatched or ambiguous identity refuses the operation.

Amend ADR-0084: Grok/Hermes session pins come from native reporters. Enrolled
mailboxes never change identity through cwd/latest-session discovery, including
after daemon restart. An unobserved runtime stays unknown.

Amend ADR-0106 to permit PiCode-owned native hooks/plugins installed using supported
vendor configuration surfaces. Installation preserves unrelated settings, is
idempotent and reversible, and never grants blanket tool approval. Shared native
configuration contains integration code and routing information, never a conversation
bearer. The bearer remains in PiCode's private per-connection directory.

Amend ADR-0089 only for explicit communication attention: persist the message first,
then consider a pointer notification in the existing recipient TUI. Revalidate the
live conversation, runtime incarnation, lifecycle state and empty input before
pasting and again before Enter. Unknown identity, user input, permission prompts
and stopped processes retain pending mail. Ambiguous writes are recorded and are
not automatically repeated. A notification is neither acknowledgement nor task
completion. This permission does not extend to Inspector command automation.

## Consequences

CLI and MCP clients interoperate without separate message stores or routing rules.
Small vendor integrations observe lifecycle and identity; the common backend owns
communication. Replacing a CLI does not require replacing the mailbox. Native
integration compatibility and terminal attention must be validated against real
TUIs before advertising support. Existing recorded-session discovery alone is not
a sufficient gate for automatic terminal input. Reconnect does not imply delivery.

## Alternatives considered

- Inherited shared MCP bearer: measured Grok leader cross-conversation identity leak.
- Per-conversation HOME, Python monkeypatch or headless ACP: changes native operation
  and duplicates configuration or bypasses the TUI requirement.
- Copying Orca's orchestration layer: scheduling, workers and task graphs exceed this
  communication boundary. Only its durable mailbox / separate attention pattern is adapted.

Sources: [Orca messaging](https://github.com/stablyai/orca/blob/main/skill-guides/orchestration/references/messaging-and-gates.md),
[Orca pointer delivery](https://github.com/stablyai/orca/blob/main/src/main/runtime/orchestration/mailbox-pointer-delivery.ts),
[Herdr integrations](https://herdr.dev/docs/integrations/).
Acceptance: [work plan](../plans/unified-native-messages.md).
