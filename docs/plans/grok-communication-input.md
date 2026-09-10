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
