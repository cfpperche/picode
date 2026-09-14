# 2026-09-14 — feat/app-parity

Docker, Inbox and tmux now render inside the same page frame as every system
route (owner's ask; scope `/desktop` + `/browser`, mobile and Canvas out).

Shipped: `AppSurface` draws `.settings-wrap` + `.settings-head` (title,
Refresh, Close) + `.settings-card`, the view's tabs as an underline nav with
count chips, the filter in the card toolbar, the split contained by the card
(sticky detail, one page scrollbar, stacking ≤880px), one line + one action for
empty/blocked/error. `.app-blank` and `.ft-*` untouched (Canvas, NativeDemo).
Three states came with it: a gone deep-linked item says "This item is no
longer in the list." + **Back to the list** (was raw `not found` + a futile
*Try again*); the tmux tab badge is the session count (shared view; the phone
sees it too); the split's placeholder says "from the list".

Verified: scratch `appparity` (:8471), screenshots read — page, group, detail,
split, narrow, narrow-bottom, empty (Inbox Done), gone, recovered,
remove-confirm, desktop shell; `__picodeOverlayAudit()` `ok:true` (dialog
inside on all four edges, `[data-align-row]` 36/36). `make test-js` (+4 in
`web/tools/app-surface.test.mjs`), `make ci-scoped` PASS, `make close` clean.
visual-review: PASS (card 5/5 on every capture). Debts: the blocked
(needs-a-newer-PiCode) card is copy-only and not captured;
`internal/server/apps.go` answers 500 for a gone entity where 404 is honest —
both in `docs/handoff/open/ui-chrome.md`. Plan and outcome:
`docs/plans/app-surface-parity.md`. Merge: fast-forward ready.
