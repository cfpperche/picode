# Pins

> Part of [PiCode's architecture](../architecture.md) (ADR-0105: one file per subsystem). Edit here; the index only links.

Flat, machine-scoped notes (`pins`, migrations 008–010, 035): title, tags,
a markdown body kept as markdown (TipTap + `tiptap-markdown`, `html:false`),
attachments and Excalidraw sketches whose bytes live under
`<data>/pins/<id>/` while `pin_files` holds the metadata. Limits **refuse**
(400 with the limit named) and never truncate: 200 characters of title,
40 per tag and 16 tags, 100 KB of body, 8 MB per image, 16 MB per file,
24 files, 2 MB of drawing per scene — the studio counts the body near the
cap. `GET /api/pins` is a summary list (no bodies); `GET /api/pins/{id}`
carries the body and files. `PATCH` accepts `ifUpdatedAt` (or `If-Match`)
and answers 409 when the row moved on, so two editors never overwrite each
other silently; the studio retains an unsaved draft in `sessionStorage`
(`picode-pin-draft:<id>`) until it matches the server copy again. Feed
events `pin.created` / `pin.updated` carry the same summary the list does,
`pin.deleted` the id; the sidebar follows them (ADR-0048) and refetches on
nothing else. An annotated image is kept **by reference** (`base_file_id`):
the browser strips the picture's bytes from the scene before upload (file
ids prefixed `bg:`) and rebuilds the background from `/files/{id}` on open,
so a 6 MB screenshot annotates within the scene cap. Sketch previews and
scenes are written through a temp file and rename; the row is inserted
first and rolled back if the bytes fail (same order as uploads). Preview
URLs carry `?v=<updatedAt>` because the bytes are cached for an hour and an
edited sketch keeps its id. Downloads name files per RFC 6266
(`filename*`). Boot sweeps `pins/<id>` directories without a row.
`web/shared/domain/pinDraft.js` holds the pure rules (limits, tag folding,
auto-title from the first file, draft retention, scene stripping).

**List v2 (migration 037).** `pins.starred` keeps a pin on top (`ORDER BY
starred DESC, updated_at DESC`; starring is not an edit and leaves
`updated_at` alone); `pins.archived_at` takes it out of the live list.
`GET /api/pins` is the live list with an `archived` count, `?archived=1`
the archived list, `?q=words` a search over both — every word must appear
in the title, a tag or the body (`lower()` on both sides, LIKE
metacharacters escaped), starred first, live before archived. `POST
/api/pins/{id}/starred {starred}` and `/archived {archived}` write the
flags and announce `pin.updated`. Archiving pauses the reminder (the
engine's due query joins `pins.archived_at IS NULL`) and closes an open
reminder item; unarchiving resumes the rule, catching up once. The
sidebar's search input debounces 150 ms and ignores answers to a query
the person has already replaced; the card's star, archive and delete show
on hover, on the active card and on starred cards.

**Reminders (ADR-0100, migration 036).** One `pin_reminders` row per pin:
`kind` ∈ {`once`, `interval`, `cron`}, `at` (UTC) for once, `interval_min`
+ `anchor` ∈ {`schedule`, `completion`} for intervals, a 5-field `cron`
evaluated in the reminder's IANA `tz` (`internal/cron`, which now steps
through the DST gap), and a materialised `next_at`. `PUT` / `DELETE
/api/pins/{id}/reminder` write it; the rule rides the pin and the list
summary with a `label` in words. `internal/remind` is the pure rule set
(`First`, `AfterFire`, `AfterClose`, `CatchUp`, `Label`) plus a one-minute
engine that asks the store for due ids and calls `FireReminder`, one
transaction each: file an Inbox item (`kind = reminder`, `source_kind =
pin`, non-blocking, title = the pin's, body = its first prose line) or,
if one is still open, touch it (or stay silent while it is snoozed); append
`pin.reminded {pinId, reminderId, inboxId, title, label, at, catchUp}`;
advance `next_at` — once → NULL and disabled, cron → next match in `tz`,
interval from the schedule → slot + interval (a backlog restarts from now),
interval from completion → nothing until the item goes `done`, then close
+ interval. A slot older than 90 s at fire time is a catch-up and says
"Was due …". The tick also wakes reminder items whose snooze ended
(clears `snoozed_until`, announces again). The Inbox row is the
acknowledgement state; deleting the pin or the rule closes its open item.
`inbox_items` was rebuilt (as 017) to admit the two enumerations. Push:
`pin.reminded` → tag `reminder:<inboxId>` under the `reminders`
preference, `requireInteraction` in `sw.js` where honoured. The Inbox app
gives the kind an "Open pin" action (`goto: pin:<id>`; both shells resolve
it to the pin) and labels the source "Pin".

**Reminders on screen.** `web/shared/domain/pinReminder.js` holds the
picker's presets (`buildReminder` → the PUT body, with the browser's IANA
zone and the viewer's morning hour from `picode-reminder-prefs`), the
sidebar line (`reminderLine` / `whenNext`: "every day at 09:00 · next
tomorrow 09:00") and the snooze instant. The desktop studio mounts
`PinReminderPicker` (Radix Popover) beside Attach and Sketch on an existing
pin: a form with nothing preset — Once (date and time) or Repeat (every N
hours; every N days at HH:MM, which is a cron for one day and an interval
with a named first fire `at` for more; "count from when I close it") and
one Set; `formFromReminder` opens it on the rule that is set; the chip
follows `pin.updated` so a change from elsewhere shows at once. On the
phone, More → Pins (`screens/PinsList.jsx`, route `more/pins`) lists and
searches; `#/pins/new` and `#/pins/<id>/edit` (`screens/PinEdit.jsx`)
create and edit title, tags and the note with the same retained draft and
`ifUpdatedAt` rules; the read-only pin screen gains Edit. `web/shared/client/reminders.js` (`watchReminders`) is the one shell
integration, called by both apps: on start, `feed.open`/`reset`,
`pin.reminded` and any `inbox.*` for a reminder item it lists
`GET /api/inbox?kind=reminder` and runs `reminderPlan` (`notice.js`) —
one sticky card per open item (`reminderNotice`: `duration: Infinity`,
key `reminder:<inboxId>`, channel `reminder`, actor kind `pin`, never
suppressed by the visible surface; X runs `onClose` → the item goes `done`,
Snooze → `snoozed_until` = now + the viewer's snooze minutes, Open →
`#/pins/<id>`), or above three open items one collapsed card
(`reminders:all`, "Open Inbox"). There is no silent first pass: a reminder
owed on load is shown. The card is a projection of the Inbox row, so a
close or a snooze on one device withdraws the card on every other through
`inbox.updated`. The phone has a read-only pin screen (`screens/Pin.jsx`,
route `pin`, under the Inbox tab) where Open lands; the editor stays on
the desk.
