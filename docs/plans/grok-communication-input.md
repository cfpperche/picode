# Grok communication input regression

The owner's Pi sender persisted its diagnostic message, but Grok received no
attention prompt. Grok Build 1.0.25 displayed a bordered composer with the cursor
at column 6; the existing guard recognized a plain composer at column 2.
The pending message was preserved, but the native recipient never read it.

Support the observed frame without weakening lifecycle, session, process or
draft guards. `testdata/grok-bordered-composer.json` contains the captured empty
editor and footer, excluding conversation history. No new protocol or dependency.

## Decision table

| Conditions | Action | Test evidence |
|---|---|---|
| Captured bordered empty editor | Permit guarded attention | `TestPeerGrokBorderedComposer` |
| Existing plain Grok editor | Preserve support | `TestPeerInputDecisionTable` |
| Draft or cursor moved | Leave pending | `TestPeerGrokBorderedComposer` |
| Multiline draft or continuation row | Leave pending | `TestPeerGrokBorderedComposer` |
| Copy mode or changed width | Leave pending | `TestPeerGrokBorderedComposer` |
| Missing model border, unknown footer or extra composer | Leave pending | `TestPeerGrokBorderedComposer` |
| Pointer cannot fit on one line | Do not claim or paste | `TestPeerGrokPointerFitsBeforeClaim`; fit checked before claim and on recheck |
| Captured pointer and native Enter shortcut after paste | Permit Enter after native identity recheck | `TestPeerGrokCapturedPointer`, `TestPeerGrokBorderedComposer` |
| Post-paste text with idle footer or captured pointer treated as empty | Refuse incomplete redraw / do not treat a draft as empty | `TestPeerGrokCapturedPointer` |
| Text changed after paste or pointer wraps | Refuse Enter | `TestPeerGrokBorderedComposer` |
| Post-paste frame not settled at the first sample (render lag under load) | Poll the exact same guards inside a bounded 1.2 s window; Enter only on a full-frame match | `TestPeerPasteSettleWindow`, `TestPeerAttentionFailureReasons` |
| Frame still unsettled when the window expires, or a draft appears mid-window | Refuse uncertain, never retry, no draft touched | `TestPeerPasteSettleWindow` |
| Unaccepted suggestion, Grok 1.0.30 restyle (styled border/gutter/prompt, italic + dim-gray text, styled closing border) | Recognize as empty together with the exact suggestion footer, empty cursor and complete frame — same as the 1.0.25 shape | `TestPeerGrokNativeSuggestion1030`; native `check_NZ2UPKO3QEVF37…` passed with the suggestion visible |
| Live draft before the paste | Leave pending, refuse silently before any claim | `TestPeerAttentionFailureReasons`; native `check_74DN3QWOTQS7HOGY2J7VTRNCZX` expired with the draft byte-identical |

## Native acceptance

Task-owned Pi-to-Grok exchange verification is recorded under
`var/qa/grok-communication-input/`. Passing requires native read, correlated reply
and both acknowledgments, not merely a recognized input frame. User test terminals
are left untouched during scratch validation.

Native check `check_LCDSXBRMOWZ4XEDP5ULFVGZNSK` passed: managed Pi sent the message,
Grok Build 1.0.25 received the automatic pointer inside its existing TUI, invoked
`picode messages read`, replied `OK` with the original message ID, and both native
participants acknowledged. `native-result.json` and `native-history.json` retain
the result and message receipts. An earlier scratch attempt exposed the changing
Enter shortcut after paste; it stayed uncertain and expired without automatic
resubmission. The successful check used the corrected native post-paste fixture.

## Settle window (2026-09-14)

The onboarding matrix on this date withheld Grok's final Enter on every attempt
under load: the paste always landed, but the single post-paste sample (150 ms)
read the frame before Grok's redraw settled, so the attempt expired uncertain
and was never retried (by design). Fix: the post-paste state is now polled
inside a bounded 1.2 s window (`peerAwaitComposer`) against the unchanged
guards — Enter still goes only after one full-frame match; structural refusals
(pane, runtime, connection, fit) return immediately; expiry refuses uncertain
exactly as the single sample did. Refusals are classified without screen
content (`paste not rendered yet` vs `composer changed or is not ready` vs
`pointer does not fit` vs `pane changed`), so the next withholding names its
check in the log.

Probe evidence (`var/qa/grok-enter-capture/probe-result.json`, Grok 1.0.30,
166×46): the full frame settles 24 ms after paste idle, 30 ms with 8 busy CPU
loops, 44 ms with 16 — the first sample can miss under load and the window
covers it roughly 27×. Synthetic CPU load did not reproduce the matrix
contention; the mechanism is covered by tests and the classified refusal now
reports which guard refuses if it ever recurs.

Native acceptance on the settle-window build (scratch `localhost:8471`, quiet
machine): `check_SSVJYDHRLMC4VKP4VAAO6INZ6U` passed — managed Pi → Grok 1.0.30,
pointer pasted and submitted once, native read, correlated `OK` reply and both
acknowledgments (`native-result.json`, `messages-pi.json`, `messages-grok.json`).
A second check fired with a live draft in the composer expired after five
minutes with the draft byte-identical and no paste (row above).

## Grok 1.0.30 suggestion restyle (2026-09-14)

The correlation re-run under load expired claude → grok with zero attention
attempts: Grok 1.0.30 restyled the unaccepted suggestion (styled border,
gutter and prompt glyph; italic plus a separate dim-gray foreground closed by
`\x1b[0m`; colored padding; styled closing border), so the 1.0.25 capture no
longer matched and the pre-claim empty-editor check silently kept the mail
pending. Both captured styles are now recognized through one matcher with
unchanged requirements — exact suggestion footer, empty cursor, complete
frame, no rows below. `check_NZ2UPKO3QEVF376WNAMKALBJWH` passed natively with
the suggestion frame visible.
