# 2026-09-08 — feat/pins-reminders: ADR-0100 slice 2, reminders server side
Shipped: migration 036 (`pin_reminders`; `inbox_items` rebuilt as 017 to
admit kind `reminder` / source `pin`); `internal/remind` (pure rules +
one-minute engine, wired in `cmd/picode` beside automations); store
`SetPinReminder` / `DeletePinReminder` / `DueReminderIDs` / `FireReminder`
/ `WakeSnoozedReminders`, reminder on `Pin` and `PinSummary`, DeletePin
closes open items in one tx; `PUT|DELETE /api/pins/{id}/reminder`;
`GET /api/inbox?kind=`; push pref `reminders` + `pin.reminded` case +
`requireInteraction` in `sw.js`; Inbox app tone + "Open pin" action
(`goto: pin:<id>`, shells resolve it in slice 3). ADR-0100 accepted.
Why: owner approved ADR-0100 and slice 2 in chat, 2026-09-08.
Verified: rule decision table + DST test (`internal/remind`), store fire
matrix (once, interval catch-up + dedupe + snooze + wake + done,
completion anchor, gone with pin/rule), server routes, notifier; scratch
on :8473 — a `once` set 70 s ahead became an Inbox item on the next tick,
`nextAt` null, `enabled` false. `internal/cron` looped forever across the
DST spring-forward gap (hour step via `time.Date` goes backwards) —
fixed with a duration step + `TestNextAcrossDSTGap`.
Not done / debts: no UI yet — nothing shows the card, no picker, `goto:
pin:` unresolved by the shells (slice 3, same deploy batch as this by the
plan: a server that fires reminders nobody sees is not done). Live feed
replay of `pin.reminded` not re-checked by hand (store tests cover the
append; the SSE curl hung the batch). Slice 4: search, starred, archive
(migration 037). Mobile pin viewer still owed from slice 1.
Merge: fast-forward ready after `make close`.
