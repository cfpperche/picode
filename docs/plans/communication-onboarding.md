# Workspace communication onboarding

Owner-approved outcome: choose participants, apply the connection, test a native
exchange, and follow activity without copying credentials or entering setup commands.
Native TUIs remain in tmux. ADR-0110 amends conversation-only consent.

Benchmark adaptation: t3code's explicit waiting state and retained drafts, and
paseo's workspace grouping from `docs/benchmarks/2026-08-24-adopt-t3code-paseo-cursor.md`.
Progressive disclosure keeps per-conversation repair in Advanced. Desktop/mobile
own their views; pure state derivation is shared.

## Decision table

| Conditions | Action |
|---|---|
| No workspace selected | Choose workspace; never select an unrelated agent by default |
| Selected owner has no conversation | Remember participation; offer Open and connect; no credential yet |
| Exact native identity and current consent | Prepare one private connection idempotently |
| New conversation | Old credential fails; create a new conversation-specific setup |
| Owner moved workspace or changed CLI | Consent does not transfer; require another selection |
| Disable / stale selection | Revoke atomically / refuse stale revision |
| Grok/Hermes with current native identity | Resolve current setup on next shell tool invocation |
| Pi receiver idle with same native session | Register through adapter without changing the editor |
| Pi missing adapter / receiver | Show one repair action; never claim a connected client |
| Terminal working / permission / draft / unknown editor | Wait visibly; no stop, paste or Enter |
| Idle terminal requires MCP reload | Prepare exact resume, revalidate, stop verified old writer, then start |
| Old process survives stop | Refuse replacement; expose error |
| Participant stopped | Explicit Open and connect; never start it from an incoming message |
| Test pending / sender busy | Wait for native input gate |
| Test write ambiguous / crash after claim | Do not automatically resubmit |
| Native send + correlated reply + both ACKs | Pass test |
| Test missing native proof for five minutes | Expire; preserve activity and offer another test |

## Acceptance

Required: store and handler decision coverage, native receiver tests, actual
native preparation/exchange on scratch instances, desktop/mobile empty/blocked/error
and successful views, adversarial review, scoped gates, close, full main CI.
Provider capacity and physical-device/platform limits must be recorded explicitly.

## Verification

Final `make ci-scoped` passed with fmt, vet, hooks, Go (18 packages), JS, build and
docs gates across 52 paths; Vale reported no errors. Evidence:
`var/qa/communication-onboarding/quality-final.log`.

Independent visual review passed for desktop/mobile, light/dark, empty, blocked,
error, activity and test-modal states. Final Advanced and primary-hover captures
were reviewed after the copy fix; overlay audits passed. Visual card:
yes / yes / yes / no / yes. The communication activation implementation is now
integrated in `main` at `00b6791b`; full main CI and the owner-authorized deploy
passed. Provider-specific activation acceptance remains open below.

Rows below follow the decision table's order. The named tests were read; partial
coverage is identified where they exercise a shared helper or only part of a row.
Go tests live under `internal/store/`, `internal/server/`, `internal/communication/`
or `internal/rpc/`; the two state tests are in
`web/shared/domain/peerParticipants.test.js`.

| Row / condition | Existing test evidence | Coverage limit |
|---|---|---|
| 1. No workspace selected | JS `workspace context never defaults to an unrelated first owner` | Covers empty, selected and missing-owner context |
| 2. No conversation | `TestPeerParticipationBeforeConversationAndOwnerDeletion`, `TestWorkspaceParticipationAPI`; JS participation-state test | Consent persists without minting a credential; native Open and connect still requires a conversation |
| 3. Exact identity and consent | `TestPeerParticipationDecisionTable` | Repeated setup reuses the connection; stale session and worker results are refused |
| 4. New conversation | `TestPeerParticipationDecisionTable`, `TestPiLaunchRegistrationFollowsNativeSession` | Old capability fails; new session receives a separate connection; registration follows native identity |
| 5. Owner moved or CLI changed | `TestPeerParticipationDecisionTable`, `TestPeerParticipationDoesNotTransferToAnotherCLI`, `TestPeerParticipationDoesNotTransferToAnotherWorkspace`; JS participation-state test | Changed CLI and an actual owner `workspace_id` change reject existing consent, authentication and stale readiness writes; the new workspace needs a fresh selection and mints a new capability |
| 6. Disable or stale selection | `TestPeerParticipationDecisionTable`, `TestWorkspaceParticipationAPI` | Revocation and stale multi-row rejection are atomic; late workers cannot recreate a disabled connection |
| 7. Grok/Hermes native lookup | `TestNativeCLIIdentity` | Per-call lookup covers current, missing, changed, resumed and ambiguous identity; this onboarding flow was not rerun in real Grok/Hermes |
| 8. Idle Pi receiver, same session | `TestNativePiSetupSharesRegistrationAndPreservesDraft`, `TestReceiverConnectionRequiresCurrentProcess`, `TestNativePiAttentionReceiverDecisionTable` | Registration is shared and session/process fenced; setup sends no model prompt; native Pi setup and exchanges also passed below |
| 9. Missing Pi adapter or receiver | `TestLaunchTamperAndFailure`, `TestReceiverConnectionRequiresCurrentProcess`, `TestPeerOnboardingAdapterRepair` | Missing adapter: the worker reports `adapter-missing`, mints nothing, and a Packages install recovers without re-selection. Stale receiver proof is rejected; receiver readiness itself is covered by rows 8/10 |
| 10. Working, permission, draft or unknown editor | `TestPeerInputDecisionTable`, `TestPeerInputRejectsMultilineComposer`, `TestNativePiAttentionReceiverDecisionTable`, `TestStopIdleFencesConversationAndCommands` | Input/receiver gates and managed stop fences are covered; the full native onboarding matrix was not rerun |
| 11. Idle terminal reload | `TestNativePiResumeComposer`, `TestCodexSubcommandsKeepHookOverrides`, `TestStopIdleFencesConversationAndCommands`, `TestCapturedStopDoesNotStopReplacement`; native Pi scratch preparation | Automatic Pi terminal resume retained the exact session; other native CLIs' new onboarding reload paths remain unverified |
| 12. Old process survives stop | `TestPeerStopReceiptFailsClosed`, `TestPeerStopStubbornChildStaysPending` | Durable live-process, PID-reuse and corrupt-receipt checks fail closed; a real tmux pane whose writer traps TERM/HUP keeps the receipt pending and clears it after the writer dies |
| 13. Participant stopped | `TestPeerCheckWorkerDecisionTable`; JS participation-state test; native stopped participant in `native-terminal-preparation.json` | Two worker passes leave a stopped sender pending and never start either participant; native stopped state is also observed |
| 14. Pending test, busy sender | `TestPeerCheckWorkerDecisionTable`; input and receiver tests from row 10 | Direct worker test holds the sender's control lock and leaves the test pending; live streaming/approval/editor gates are exercised separately |
| 15. Ambiguous test write or crash after claim | `TestPeerCheckWorkerDecisionTable`, `TestPeerAttentionClaimsDoNotConsumeOrRepeat` | Two worker passes preserve attempted, running and uncertain phases without sending as the native participant; crash state is seeded, not a killed live process |
| 16. Native send, reply and both ACKs | `TestPeerNativeCheckNeedsRoundTripAndBothAcknowledgements`, `TestPeerCheckWorkerDecisionTable`; both native scratch checks below | Setup alone, send alone and one ACK cannot pass; correlated exchange plus both ACKs passes |
| 17. Five-minute expiry | `TestPeerCheckWorkerDecisionTable` | Pending and attempted diagnostics expire with time advanced to six minutes; unfinished native activity remains and a subsequent test can be created |

Native evidence is task-owned under `var/qa/communication-onboarding/`:

- `native-managed-result.json` and `native-managed-history.json`: check
  `check_O667E6YDCSQDBVT3HJGA62HU7G` passed for two managed Pi agents; the original
  message and its correlated reply both have acknowledgment timestamps.
- `native-tui-result.json` and `native-tui-history.json`: check
  `check_C5M4F4CICSJ7EWR4R5L5ROESIT` passed from a Pi terminal to managed Pi, with
  both acknowledgments. Automatic Pi terminal preparation resumed its exact
  native session; `native-terminal-preparation.json` records the applied connection.
- Codex remained `waiting-conversation`. Native SessionStart reporting at startup
  before a first turn is unverified. The manual QA restart opened a fresh
  conversation rather than resuming; it is not evidence of correct startup/resume.

Every onboarding decision row now has direct coverage; what remains is vendor behavior, not a PiCode gap.

Vendor facts: initial Codex/Hermes/OpenCode and resumed Codex/Hermes need a first
prompt; fresh Claude waits for a saved first conversation
(`docs/plans/cli-attention-matrix.md`). Six-CLI paired transport acceptance:
`docs/plans/communication-native-finish.md`.

Standing limits: the full onboarding matrix was not rerun on native
Claude/OpenCode/Grok/Hermes (transport acceptance does not establish this flow);
physical mobile and non-Linux recovery are unverified; PTY input rechecks do not
make editor access atomic.
