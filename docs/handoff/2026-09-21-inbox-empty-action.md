# 2026-09-21 — inbox-empty-action: an empty Active queue with history names the way to it

The ui-chrome debt "Inbox empty lines name no next action" is paid for the case that has a
destination (`internal/apps/inbox.go` rootView). When the Active queue is empty **and** Done holds
items, the root no longer falls to the view-level blank slate: it emits a list block with
`Empty: "Nothing needs you right now."`, `Items: nil`, and one item-less action — `open-done`,
labeled "See the done item" (plural "See N done items" beyond one). Action `open-done` returns
`ActionResult{Path: "done"}`, the same destination the Done tab names, so the dead end now has a
door. "Inbox zero" was rejected: with history visible in Done it would wrongly imply the whole
mailbox is empty.

A mailbox with nothing at all keeps `v.Empty` and no action. Reason recorded in
`docs/handoff/open/ui-chrome.md`: an app reaches the host only through ADR-0109's closed doors, none
of which is an honest "start something" from the Inbox — a button there would be theatre.

Evidence — `TestInboxEmptyActiveNamesDone` pins the block shape, the singular label, and that firing
`open-done` yields `Path "done"`; `TestInboxRootView` still pins the truly-empty surface (`v.Empty`,
zero blocks, `Layout ""`). `go test ./internal/apps/` green; `make close` green. Scratch instance
(localhost:8471, since stopped): the truly-empty capture shows the one line and no action; after
answering one item the same surface shows the line + "See the done item"; clicking it landed on
`#/app/inbox/done` with the answered item visible.
`var/screenshots/inbox-empty-{truly,with-action,done-after-action}.png` (gitignored).
`window.__picodeOverlayAudit()` = ok, no uncovered or misaligned rows (row heights [36, 36]).

visual-review: PASS (truly-empty and with-action captures read; overlayAudit ok; click landed on Done).
