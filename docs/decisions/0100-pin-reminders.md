# ADR-0100: Pin reminders — a scheduled fire that lives in the Inbox

- **Status**: proposed
- **Date**: 2026-09-08
- **Number**: 0100 (renumbered from a provisional 0099 at merge; 0099 is the package configuration GUI)

## Context

Pins (migrations 008–010) are flat notes with files and sketches and no
notion of time. The owner asked for reminders: a pin can be brought back
once at a date and time, or on a cadence ("every 24 hours"), and the
reminder must stay on screen until the person closes it. Three boundaries
move: **persistence** (a new table and two new Inbox enumerations),
**protocol** (a new feed event and a new push tag), and the **scheduler**
(a second lane on the one-minute engine). The review that precedes this
decision and the study it draws on are `docs/plans/pins-v2.md` and
`docs/benchmarks/2026-09-08-pins-reminders.md`.

Facts that shape the choice:

- A browser toast is tab memory. "Visible until closed" across a reload, a
  second browser and the phone needs a server-side acknowledgement state.
- `inbox_items` already is the product's "owed to you" surface (ADR-0037)
  with `snoozed_until`, `state`, provenance (`source_kind`, `source_id`)
  and a badge; the phone and the push notifier already consume it.
- `internal/automate` (ADR-0045) already ticks every minute from SQLite,
  survives restarts and collapses a missed backlog into one catch-up.
- The notice model (`notice.js`) already has the sticky, keyed, withdrawn-
  by-the-caller card for needs-you.
- "Every day at 09:00" is a calendar rule in a named zone; "every 24 hours"
  is a duration. RFC 5545 and every benchmark keep them apart.
- `inbox_items.kind` / `source_kind` are CHECK-constrained; SQLite cannot
  alter a CHECK (precedent: migration 017 rebuilt the table).

## Decision

A reminder is a row in `pin_reminders` owned by one pin (cascade on
delete): `kind` ∈ {`once`, `interval`, `cron`}, with `at` (UTC) for once,
`interval_min` + `anchor` ∈ {`schedule`, `completion`} for interval,
`cron` (5-field, `internal/cron`) for calendar rules, an IANA `tz`
captured from the browser at save, and a materialised `next_at` (UTC).
`internal/remind` ticks on the same one-minute engine as automations and,
for each row with `enabled = 1 AND next_at <= now`, in one transaction
creates an Inbox item (`kind = reminder`, `source_kind = pin`,
`source_id = <pin>`, non-blocking), appends `pin.reminded {pinId,
reminderId, inboxId, title, at}` and advances `next_at` — once → NULL and
disabled; cron → next match in `tz`; interval anchored to the schedule →
fire + interval; interval anchored to completion → left alone until the
Inbox item goes `done`, then close + interval. A missed slot (daemon down)
fires **once** with the original time in the body; the backlog never
replays. A fire while the previous item is still `unread` creates no second
item: it touches the open one and the card is re-raised. The Inbox item is
the acknowledgement state: `unread` = owed, `snoozed_until` = snoozed,
`done` = closed; closing anywhere closes everywhere through
`inbox.updated`. Browsers project unread reminders as sticky notices
(`duration: Infinity`, `key = reminder:<inboxId>`, `channel = reminder`),
re-raised on feed open and capped at three before collapsing into one
"N reminders" card; the push notifier sends `reminder:<inboxId>` with
`requireInteraction` and `renotify` as progressive enhancement. Migration
036 rebuilds `inbox_items` (as 017 did) to admit the two enumerations and
adds `pins.starred` and `pins.archived_at`.

## Consequences

Easier: the phone, the desk and the push all read one state; snooze,
done and provenance come from code that already has tests; the scheduler
has one tick, one catch-up rule and one reconcile; a deleted pin takes its
reminders and their open items with it in one cascade + transaction.

Harder: `inbox_items` is rebuilt again (fourth schema touch), so the
migration must be proven on a populated copy before deploy (the
2026-09-06 boot-on-empty-scratch lesson); the Inbox app gains a kind it
must render and act on, on both apps; `cron` limits calendar rules to what
five fields say (no "last Friday", no yearly) — the study records RRULE as
the alternative if that is ever asked for.

Accepted cost: an interval anchored to completion cannot be predicted in
the sidebar until the item is closed (the line reads "after you close the
last one"); `requireInteraction` does nothing on Android and Safari, so
the OS knock can vanish while the Inbox row cannot.

Who breaks if we are wrong: a reminder that fires twice or never is the
failure that erodes trust fastest, so `remind.Due` ships with the decision
table in the plan as its test, and the fire is idempotent per
(`reminderId`, `slot`) by the unique open-item rule.

## Alternatives considered

- **A `pin_reminder_fires` table plus its own panel.** A second
  notification center beside the Inbox, with its own snooze, badge, push
  and phone view to write and keep in step. Lost on duplication.
- **Browser-side scheduling (`setTimeout` / service worker alarms).**
  Fires only while a tab lives; the phone and a closed laptop see nothing;
  two tabs fire twice. Lost on correctness.
- **RFC 5545 RRULE storage.** The right reference, the wrong dependency
  for the rules asked for; `internal/cron` already exists and the
  Automations editor already has its preset layer. Lost on ADR-0086's
  cost bar; recorded as the upgrade path.
- **Kind `fyi` with `reason = reminder` to avoid the table rebuild.**
  Saves a migration, loses typed filtering, badge rules and the Inbox
  app's ability to render the kind; every consumer would sniff `reason`.
  Lost on clarity.
- **Auto-closing cards with a long timer.** The owner's requirement is the
  opposite, and VS Code / Slack / Linear all keep an owed item until acted
  on. Not considered further.
