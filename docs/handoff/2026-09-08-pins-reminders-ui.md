# 2026-09-08 — feat/pins-reminders-ui: ADR-0100 slice 3, reminders on screen
Shipped: `domain/pinReminder.js` (presets → PUT body, morning-hour and
snooze prefs, `reminderLine`/`whenNext`), `notice.js` reminder cards
(`reminderNotice`, `remindersCollapsedNotice`, `reminderPlan`, channel
`reminder`, `onClose`), `client/reminders.js` `watchReminders` used by
both shells; desktop `PinReminderPicker` (Radix Popover) in the studio,
sidebar reminder line, chip follows `pin.updated`; Notice twins draw a
pin glyph and run `onClose` on X; Settings twins gain the reminder switch,
the morning hour and the snooze length; PushPrefs twins the push switch; `goto: pin:` resolved on both shells; mobile route
`pin` + read-only `screens/Pin.jsx` (slice 1's carry-over paid); Inbox
app labels pin rows "Pin". Fix: `PinEditor` loads with `emitUpdate:
false` — a heading/list body was retained as an "unsaved" draft.
Why: owner approved slice 3 in chat, 2026-09-08 (same deploy batch as
slice 2 by the plan).
Verified: node tests (presets, whenNext rounding, notice cards, plan
collapse, mobile route); `ci-scoped` PASS; scratch :8474 — picker sets
"every day at 09:00" (API cron `0 9 * * *` in America/Sao_Paulo), a once
fired into a card with pin glyph / Snooze / Open, X closed the item
(`done` via API), four fires collapsed into "4 reminders" → Open Inbox,
Snooze on a card snoozed the item and withdrew the card, chip followed an
API change live, mobile `#/pins/<id>` renders and shows the same cards.
Not done / debts: no keyboard hotkey to focus the toaster region beyond
sonner's Alt+T; the collapsed card's X only hides it locally (the items
stay); slice 4 (search, starred, archive, migration 037) is next.
Merge: fast-forward ready after `make close`.
