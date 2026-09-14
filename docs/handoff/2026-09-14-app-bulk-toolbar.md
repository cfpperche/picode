# 2026-09-14 — feat/app-bulk-toolbar

Follow-up to app-parity, owner's ask: on Inbox's Done tab "Clear all done"
trailed the list; it now sits right-aligned on the card toolbar row, beside
the filter — where every route page keeps its list actions.

Shipped: host rule in `AppSurface` — an actions block declared `Pane: "list"`
is the list's bulk action and renders in `.app-card-bulk` at the toolbar's
right end; it is lifted out of the list pane (never rendered twice). The
filter gains `flex-basis: 240px` so on a narrow row it wraps below instead of
collapsing. Detail-pane rows (an item's Done/Snooze) and unpaned rows (tmux's
"Back to the inventory", Docker's "Check again") keep their places. Mobile's
own AppSurface is untouched.

Decision table, all rows verified on scratch `bulkbar` (:8473):
list-pane row → toolbar (screenshot, 36/36 aligned, audit ok); stacked 760px →
same row; detail-pane row → detail pane; unpaned row → body; no bulk (Docker)
→ toolbar unchanged; button opens the app's confirm, Escape cancels.

Verified: `make test-js` (+1 guard in `web/tools/app-surface.test.mjs`),
`make ci-scoped`, `make close`.
visual-review: PASS (done-toolbar.png, done-clear-confirm.png, done-narrow.png,
docker-bulk-regression.png all read; overlayAudit ok, [data-align-row] 36/36).
Merge: fast-forward ready. The parity branch's debts
(`docs/handoff/open/ui-chrome.md`) still stand.
