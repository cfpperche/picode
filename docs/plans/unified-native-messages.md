# Unified native messages — execution and acceptance

Owner-approved scope: a common mailbox, CLI/MCP clients, native session reporters,
TUI attention and Messages UI. No ACP runtime, orchestration or deployment.

## Sequence

1. Prove Grok/Hermes native session context with two TUIs, `/new`, resume and restart.
2. Add the common CLI using the existing MCP service and transactional store.
3. Replace Grok/Hermes interception with native hook/plugin integration; bind live
   reports to wrapper incarnations and keep native session discovery out of routing.
4. Add persistent attention outcomes and gated pointer delivery; never duplicate an
   ambiguous write. Keep user input and permission prompts intact.
5. Show setup, live status, attention outcome and history in desktop/mobile Messages.
6. Exercise native Pi, Claude, Codex, Grok, Hermes, OpenCode and managed Pi. OpenCode:
   `zai/glm-5.3-flash`, variant `max`. Independent review, scoped gates, close, main CI.

## Decision table

| Conditions | Action / evidence required |
|---|---|
| Explicit private connection outside a native conversation | Existing bearer authorization on every server operation |
| Native session matches exactly one private connection | CLI calls the same MCP tools |
| Missing / contradictory / mismatched native identity | Refuse before network access |
| Multiple local connections for one native session | Refuse ambiguous setup |
| Two TUIs share a Grok leader | Each hook and tool reports its own native ID |
| `/new` or fork | Old setup cannot authorize the new conversation |
| Resume same conversation / restart server | Durable mailbox remains; live runtime must report again |
| Revoked connection / workspace or owner changed | All operations refuse; history remains |
| Identical request ID and body | Same receipt, one stored message |
| Conflicting request ID / invalid ack batch | Refuse atomically |
| Live recipient, matching session/run, idle, input empty | Persist attention attempt; revalidate before paste and Enter |
| Working / permission prompt / user draft / missing observation | Retain pending message; do not type |
| Process/session/input changes between paste and Enter | Do not submit; expose interrupted attention |
| Write result ambiguous / process crashes during attempt | No automatic repeat; pending mail remains readable |
| Native integration install repeated / unrelated files changed | Preserve user content; refuse ownership conflicts |

## Evidence

Native fixtures and logs: `var/qa/unified-native/` (ignored), 2026-09-09.
Sessions and messages below belong only to the scratch instance on port 8474.
No row is accepted solely from source inspection or a mock transport.

| Native surface | Observed result |
|---|---|
| Grok 1.0.25, two simultaneous native terminal interfaces sharing a leader | Native CLI contacts identify each conversation; automatic pointer, read, reply and acknowledgment in both directions |
| Hermes 0.21.1, two simultaneous native terminal interfaces | Native CLI contacts; automatic pointer, read, reply and acknowledgment in both directions, repeated after resume |
| Pi 0.85.1 terminal | Native MCP contacts; automatic receiver pointer, read, reply and acknowledgment |
| Managed Pi 0.85.1 | Native MCP contacts; automatic receiver pointer, read and acknowledgment |
| Codex 0.153.4 | Native MCP contacts and automatic pointer/read; explicit diagnostic follow-up sent a reply and acknowledged it, with per-tool native approvals |
| OpenCode 1.18.30, `zai/glm-5.3-flash`, `max` | Native MCP contacts and pointer submission; later model calls blocked by provider balance/resource-package error |
| Claude Code 2.1.267 | Native MCP configured and pointer submitted; model roundtrip blocked by session usage limit in this run |

Durable receipts: Grok `native-g1-g2-001` / `native-g2-pong-001`, Hermes
`native-h1-h2-001` / `native-h2-h1-001` and `native-matrix-hermes2-001`,
Pi `native-matrix-pi-001` and `native-matrix-managed-001`. All these original
requests were acknowledged. A notification is not a guarantee that a model
will obey a message; Codex read the diagnostic and requested explicit follow-up.

Two native Grok/Hermes `/new` operations after daemon restart invalidated their
old connection; actual shell tool calls refused the new identity. Resuming the
original IDs recovered the existing connection and history. Owned multiline
drafts in both native inputs remained unchanged while incoming messages stayed
pending. Codex's native permission dialog kept incoming work from being submitted.
The recorded uncertain Claude paste was not repeated after restart.

Automated coverage includes CLI/MCP interoperability, per-call native identity,
ambiguous/mismatched/revoked connections, request deduplication, atomic ack,
conversation-address reservation across resume, ordered native publication,
child/delayed-event rejection, owned native installs and Hermes profiles,
attention claims/crash ambiguity/backlog fairness, native input frames/drafts,
and Pi receiver idle/permissions/drafts/session-change decisions.

Desktop/mobile regression scripts and independent screenshot review passed,
including live status, setup errors, explicit refresh, notification history and
mobile history after scrolling. Overlay audits passed.

## Acceptance still outside this delivery

- Repeat Claude and OpenCode full model roundtrips after native account capacity
  returns; do not substitute another account/model or bypass vendor limits.
- Physical mobile devices and non-Linux pane/process recovery are unverified.
- Codex custom resume arguments with leading global options or `--` are not the
  recorded recipe and do not attach (`TestPeerResumeExactDecisionTable`,
  `TestPeerLaunchCustomResumeDoesNotAttach`).
- PTY input is not an atomic native editor transaction. Rechecks detect observed
  changes; uncertainty is durable and never automatically retried.

Final close and main CI are recorded in the session handoff and commit history.
