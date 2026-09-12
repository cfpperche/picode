# Grok first-message attention

The CognixSE Pi-to-Grok test stored a message but never notified its recipient.
The recipient was a new Grok 1.0.30 TUI: its empty welcome composer displays a
right-aligned `[stable]` footer. Previous tests had sent a warm-up prompt first,
which replaced that footer with shortcuts and missed this initial state.

The existing guarded attention path accepts the captured empty welcome frame.
Typing dismisses welcome; the exact post-paste pointer still requires the native
Enter shortcut footer. Native identity, Idle, pane/process, draft, width and
non-retry checks are unchanged. No new dependency or protocol.

| Condition | Expected behavior |
| --- | --- |
| Empty native welcome frame, exact right-aligned `[stable]` | Permit guarded first attention |
| Draft, moved cursor, copy mode or changed width | Leave pending |
| Unknown/misaligned footer, broken frame or extra editor row | Leave pending |
| Actual welcome-to-pasted-editor transition | Submit only the exact pointer with Enter footer |
| Pasted pointer but stale `[stable]` footer | Refuse submission |
| Untouched fresh Pi and Grok sessions | Require native message, reply and both ACKs without warm-up |

Regression: `TestPeerGrokWelcomeComposer`, existing `TestPeerGrok*` safeguards.
Production CognixSE sessions remain untouched.

## Acceptance — 2026-09-12

- Focused `TestPeerGrok*` regression tests and `make ci-scoped` passed; adversarial review passed. The guarded negative cases cover drafts, cursor movement, copy mode, changed width, unknown/misaligned footer, broken frame, extra editor rows and stale welcome footer after paste.
- Fresh Pi and Grok terminals completed `check_7A43BKRBORLDW67SLH26CUV6WK` without a preparatory prompt or model turn in either terminal. The request and reply were both notified and acknowledged; the reply references the original message and reverses its sender/recipient identities.
- Native evidence: `var/qa/grok-welcome-attention/native-result.json` in the root checkout (local, uncommitted).
- Scratch visual review passed on desktop/mobile, with screenshot reading and a passing overlay audit. Evidence: `var/screenshots/grok-welcome-attention/` in the root checkout (local, uncommitted).
- This verifies the previously missed cold-start state. It does not claim a production retry or deployment; the owner's CognixSE terminals remain untouched.
