# 2026-09-08 — remove-send-toast: no success toast on terminal message send

Shipped: sending a message to a terminal no longer toasts "Sent to the
terminal." — the user is looking at the pane and sees the message land, so
the confirmation was redundant chrome. Desktop message bar
(`TermAttachBar.jsx`) and mobile sheet (`TermAttachSheet.jsx`) changed
together; `toastError` stays, so send failures still surface as toasts.
CHANGELOG `[Unreleased] → Changed`.

Verified: `make web` build, `make close` ci-scoped PASS (fmt, vet, hooks,
test-js, build). Scratch instance: seeded a Pi CLI terminal, sent from the
mobile sheet (390×844) and the desktop bar (1440×900) — zero
`[data-sonner-toast]` after send, sheet closes / bar clears to its empty
state, message visibly lands in the pane; `__picodeOverlayAudit()` ok:true
on both. Failure path not exercised live (unchanged code — catch still
calls `toastError`); named as accepted debt.

visual-review: PASS (send-sheet-open, send-after-no-toast,
desktop-send-after-no-toast all read; card 5/5).

Not done / debts: the "Sent." toast on mobile inbox replies
(`respondInbox` in mobile `App.jsx`) was left as-is — different flow
(reply vanishes from the list, no pane to watch); flag to the owner if
they want it gone too.

Merge: fast-forward ready (base a1df5c94).
