# Pins v2 — adversarial review and reminders plan

- **Date:** 2026-09-08
- **Status:** proposed — waiting for the owner's call on the scope table at the end
- **Companion docs:** study `docs/benchmarks/2026-09-08-pins-reminders.md`;
  ADR draft `docs/decisions/0100-pin-reminders.md` (number provisional —
  parallel sessions are numbering ADRs too)
- **Code under review:** `internal/store/pins.go`, `internal/store/pin_files.go`,
  `internal/server/pins.go`, `internal/server/pin_files.go`,
  `web/desktop/src/components/{Pins,PinStudio,PinEditor,PinSketch}.jsx`,
  `web/desktop/src/lib/pinFileDrop.js`, migrations `008`–`010`

## Part 1 — Adversarial review of pins v1

Method: read every pin file end to end, then probed the store with a
throwaway test (deleted after the run) for the limits the code claims.
"Probe" = reproduced in that test; "read" = found by reading and not
executed live. Nothing here was fixed yet.

### Findings, most severe first

| # | Sev | Where | Finding | How seen |
|---|---|---|---|---|
| 1 | high | `store/pins.go:44,47` | Title and body are cut by **byte** (`title[:200]`, `body[:100000]`), so a multibyte character on the boundary leaves **invalid UTF-8 in SQLite** and in every JSON that carries the pin. | probe: 199×`a`+`é` → `utf8.ValidString` false |
| 2 | high | `store/pins.go:47`, `PinStudio.jsx:236` | Body over 100 KB is **silently truncated**; the studio still says "Pin saved.". A long paste loses its tail without a word. | probe |
| 3 | high | `store/pins.go:113,137` | `pin.created` / `pin.updated` carry the **whole pin, body included** into the `events` table and over SSE to every connected client — 105 KB per save in the probe. File routes emit the same `pin.updated` type with only `{id}` (`pin_files.go:110,124,166,199`): one event name, two shapes. No consumer reads either (`feedReducers` ignores `pin.*`). | probe + read |
| 4 | high | `PinSketch.jsx:12-37`, `store/pin_files.go:14`, `server/pin_files.go:167` | **Annotate fails on any image above ~1.5 MB.** The base image is embedded as a data URL inside the Excalidraw scene; the scene cap is 2 MB (`MaxPinSceneSize`) while an image may be 8 MB. `readFormFile` overflows and the handler answers **"scene is required"** — the wrong message. | read |
| 5 | med | `server/pin_files.go:184-197` | Sketch save is three steps with no rollback: row, preview, scene. A failure after the row leaves a sketch that 404s "scene missing". Updates overwrite files in place (no temp + rename); a crash mid-write corrupts an existing sketch. Upload (`:73-84`) does roll back the row — the two paths disagree. | read |
| 6 | med | `PinStudio.jsx:137-153`, `server/pin_files.go:117` | **Stale thumbnail after editing a sketch.** `UpdatePinSketch` keeps the file id, so the preview URL is unchanged; the bytes are served with `Cache-Control: private, max-age=3600` and the gallery `<img>` keeps the old `src`. The edited sketch shows its previous picture for up to an hour. | read |
| 7 | med | `server/pins.go` PATCH, `store/pins.go:127` | **Last writer wins.** PATCH replaces title, tags and body with no precondition (`updatedAt`, ETag). Two tabs editing the same pin lose one of the edits silently. | read |
| 8 | med | `Pins.jsx:20-27` | Sidebar reloads `/api/pins` on **every `hashchange`** (any navigation anywhere in the app) and **never on feed events**: a second browser sees no change until it navigates; a single browser refetches full bodies on every tab switch. `ListPins` selects `body` although the card renders title, tags and a count. | read + probe (list payload = bodies) |
| 9 | med | `web/mobile/src/lib/mobileRoutes.js:37` | `pins` → `settings`: a pin link opened on the phone **lands on Settings** with no message. Mobile has no pins surface at all, while `@excalidraw/excalidraw` ships in the mobile bundle for the composer sketch. | read |
| 10 | med | `store/pins.go:49-60` | Tags have **no length cap** (a 5 000-character tag is accepted) and whitespace runs become `--` (`"a b  c"` → `a-b--c`). Count is capped at 16, length is not. | probe |
| 11 | low | `PinStudio.jsx:95-104` | Dropping a file on a *new* pin auto-creates a pin titled **"Untitled"** before the user typed anything; Cancel then leaves it behind. Orphans accumulate. | read |
| 12 | low | `server/pin_files.go:114` | `Content-Disposition` is built with `strconv.Quote`: a name with an accent downloads as `é…` (Go escaping), not RFC 6266 `filename*`. | read |
| 13 | low | `server/pins.go:54`, `packages.go:287` | POST maps errors by **substring** ("title is required") and everything else to 500; PATCH maps everything but not-found to 400 through `statusForStore`, so a real DB failure on PATCH reports as the caller's fault. | read |
| 14 | low | `server/pins.go:83`, `store/pins.go:146` | Delete removes the row (cascade) then the directory; a failed `RemoveAll` leaves orphan bytes forever — there is no sweeper for `pins/` directories without a row. Backup copies `pins/` blindly (`backup/snapshot.go:74`), so orphans are backed up too. | read |
| 15 | low | `PinStudio.jsx` | No unsaved-changes guard: Cancel, a sidebar click or a hash change drops the draft. Mobile v2 retains drafts; the studio does not. | read |
| 16 | low | tests | Coverage is `TestPinsCRUD`, `TestPinFiles` and `TestPinDeleteRemovesDir`. Nothing exercises upload limits, the sketch round-trip, `/scene`, the id regex, truncation, tag caps or the list payload. Findings 1, 2, 4, 5, 6, 10 have no test that could have caught them. | read |

What held up under attack: path handling (`pinIDOK` before any disk
path; ids come from `newID`, never from the client), SQL is
parameterized throughout, uploads are bounded in memory
(`io.LimitReader`), non-image files are always `attachment` with a global
`nosniff`, and the markdown body is stored as markdown with `html:false`
on both sides. The security surface is sound; the failures are data
hygiene, consistency and product reach.

### Product gaps (not bugs — the bar the study measures against)

- No search, no filter by tag, no archive, no "keep on top"; the only
  order is `updated_at DESC`; the only list is the sidebar.
- No date, due, or reminder concept at all. A pin is a note with files.
- Nothing on the phone.
- The editor cannot make a checklist (StarterKit has no task list).

## Part 2 — Pins v2

### What v2 is

Pins stay what they are — flat, machine-scoped notes with files and
sketches — and gain **reminders**: a pin can ask to be surfaced again, once
at a date and time or on a cadence, and the reminder stays on screen until
the person closes it. Around that, v2 pays the hygiene debt above and gives
the list the three organizing tools every benchmark has (search, keep on
top, archive).

### Reminder model (owner: the server, in SQLite — ADR-0100)

```
pin_reminders
  id              TEXT PK
  pin_id          TEXT NOT NULL REFERENCES pins(id) ON DELETE CASCADE
  kind            TEXT NOT NULL       -- 'once' | 'interval' | 'cron'
  at              TEXT                -- once: RFC3339 UTC
  interval_min    INTEGER             -- interval: minutes (>= 5)
  cron            TEXT                -- cron: 5-field, internal/cron
  anchor          TEXT NOT NULL       -- 'schedule' | 'completion' (interval only)
  tz              TEXT NOT NULL       -- IANA zone the wall clock belongs to
  next_at         TEXT                -- UTC, materialized; NULL = finished
  last_fired_at   TEXT
  enabled         INTEGER NOT NULL DEFAULT 1
  created_at, updated_at
```

One reminder per pin in v2 (the UI shows one "Remind me" control; the
table allows more so v3 does not migrate). `tz` is captured from the
browser at save time (`Intl.DateTimeFormat().resolvedOptions().timeZone`)
and every wall-clock rule (`cron`) is evaluated in that zone; `at` is
stored in UTC. "Every 24 hours" is an **interval** measured from the last
fire and therefore drifts across DST; "Every day at 09:00" is a **cron**
and does not. The picker names both so the person chooses knowingly.
An interval also has an **anchor**: from the schedule (fires every N hours
whatever the person did) or from completion (the next fire is N hours
after the person closed the last one — Todoist's `every!`, Things' "after
completion"). Day-only presets use a **morning hour** preference (default
09:00, Preferences → Notifications), as Slack and Keep do.

The picker (progressive disclosure: a chip in the studio, a popover on
click) offers presets first, custom last:

| Preset | Stored as |
|---|---|
| In 1 hour / In 3 hours | once, now + N h |
| Tomorrow 09:00 / Next Monday 09:00 | once, in `tz` |
| Every day at HH:MM | cron `MM HH * * *` |
| Every weekday at HH:MM | cron `MM HH * * 1-5` |
| Every N hours | interval N×60 |
| Pick date & time | once |
| Cron (advanced) | cron — reuses the Automations editor's `cron.js` |
| ☐ Count from when I close it | `anchor = completion` (interval presets only) |

### Firing (a second lane on the one-minute engine)

`internal/automate` already ticks every minute, survives restarts and has
a catch-up rule (ADR-0045). Reminders get a `remind` package with the same
shape and its own `Due`:

- Tick: `SELECT … WHERE enabled = 1 AND next_at <= now`.
- For each due reminder, in one transaction: create an **Inbox item**
  (`kind = reminder`, `source_kind = pin`, `source_id = pin id`,
  non-blocking, title = pin title, body = first prose line of the pin) and
  append `pin.reminded {pinId, reminderId, inboxId, title, at}`; then
  advance `next_at` (once → NULL + `enabled = 0`; interval → fire time +
  interval; cron → next match in `tz`), stamp `last_fired_at`.
- The Inbox row is the **acknowledgement state**: `unread` = the reminder
  is still owed to the person; `done` = closed; `snoozed_until` = snoozed
  (the column already exists, `store/inbox.go:339`).

Decision table (each row: conditions → action):

| Kind | Daemon was down over the slot | Pin deleted | Reminder disabled | Action at tick |
|---|---|---|---|---|
| once | no | no | no | fire at `next_at`; finish |
| once | yes (slot passed) | no | no | fire once at boot with the original time in the body ("was due 09:00"); finish |
| interval / cron | yes, k slots missed | no | no | fire **once** (catch-up), `next_at` from now — the backlog collapses, as automations do |
| any | — | yes | — | cascade removed the row; any open Inbox item for it is closed by the same transaction |
| any | — | no | yes | skip; `next_at` kept so re-enabling resumes |
| any | pin edited after the fire | no | no | the Inbox item keeps the title it fired with; "Open" reaches the live pin |
| interval / cron | fire while the previous fire is still `unread` | no | no | **no second item** — the open item's `updated_at` moves and its notice is re-raised (one card per reminder, never a pile) |
| interval, `anchor = completion` | — | no | no | `next_at` is **not** advanced at the fire; it is set to close time + interval when the Inbox item goes `done` |

### Showing it — sticky notice, closed only by the person

The notice model already has the sticky shape the owner asked for: the
needs-you card (`notice.js:needsYouNotice`) uses `duration: Infinity`, a
`key` so a re-fire replaces rather than stacks, and is withdrawn by the
caller when the question disappears. A reminder notice is the same card
with a pin face:

```
notice := {
  level: "info", channel: "reminder",
  actor: { kind: "pin", name: <pin title> },        // pin glyph, no CLI face
  status: "reminder · every day at 09:00" | "reminder · was due 09:00",
  title: <first prose line of the pin, or the title>,
  actions: [{ label: "Snooze", run }, { label: "Open", hash: "#/pins/<id>", primary }],
  key: "reminder:" + inboxId, duration: Infinity,
  target: ""                                        // never suppressed by the visible surface
}
```

- The **X** on the card marks the Inbox item `done` (`POST /api/inbox/{id}/state`).
  Every other browser sees `inbox.updated` and withdraws its card
  (`dismissNotice(key)`) — closing on the desk closes on the phone.
- **Snooze** posts `snoozed_until` (default 1 h; the Inbox row offers
  10 min / 1 h / tomorrow 09:00) and withdraws the card; the engine's tick
  re-raises it when the snooze ends (`snoozed_until <= now` → `pin.reminded`
  again with the same `inboxId`).
- On **page load / feed open** the shell lists `GET /api/inbox?kind=reminder`
  (unread, not snoozed) and raises one card each. `needsYouPlan`'s
  "first pass only records" rule does **not** apply: a reminder owed is
  shown, that is the point of it.
- **Pile-up guard:** sonner keeps `visibleToasts` (1–5) on screen and queues
  the rest; sticky cards would wall the screen. Above 3 open reminders the
  shell shows **one** card — "N reminders" with **Open Inbox** — and the
  Inbox app is the place to work through them (it already has snooze,
  done, and per-item rows).
- `channel: "reminder"` is a third announce preference beside *finished* and
  *needs you* (default on). Muting it silences the card, never the Inbox
  row or the push.
- **Push** (phone, no host browser online — the existing presence rule):
  `Notifier.OnEvent` gets a `pin.reminded` case → tag `reminder:<inboxId>`,
  `urgency: high`, `Topic` = reminder id (RFC 8030: a retry replaces the
  pending push instead of doubling it), and `sw.js` passes
  `requireInteraction: true` + `renotify: true` for that tag. Honesty about
  reach: `requireInteraction` is Chromium-only and ignored on Android;
  Safari/iOS has neither it nor notification actions. The OS notification
  is a knock; the Inbox row is the state. A new push pref `reminders`
  mirrors the browser channel.
- **A11y:** arrival is announced through the polite live region as today
  (`role="status"`); the card itself is focusable but never focused on
  arrival; the toaster region is labelled and reachable by Sonner's Alt+T
  hotkey; Escape closes only the focused card; each button carries the pin
  title in its accessible name ("Snooze 'Deploy checklist'"); the status
  line says the cadence so a screen reader hears *why* it fired. No
  `role="alert"`: the person scheduled this themselves.

### Hygiene in the same release (fixes findings 1–16)

| Fix | Findings |
|---|---|
| Rune-safe limits that **refuse** (400 with a message) instead of truncating; the studio shows the count near the limit | 1, 2, 10 |
| `pin.created`/`pin.updated` carry `{id, title, tags, updatedAt, fileCount}` — one shape, no body; a `pin.deleted` stays `{id}` | 3 |
| Annotate stores the base image by **reference** (`baseFileId`, already in the row) and rebuilds the scene from `/files/{fid}` on open; the scene never embeds bytes; the wrong error message goes | 4 |
| Write preview and scene to `*.tmp` then rename; insert the row **last**, in the same order as upload; delete both temps on failure | 5 |
| Preview URL gains `?v=<size or updatedAt>`; `UpdatePinSketch` bumps `updated_at` on the file row | 6 |
| PATCH accepts `If-Match: <updatedAt>` (or `updatedAt` in the body) and answers 409 with the server copy; the studio merges or asks | 7 |
| Sidebar subscribes to the feed (`pin.*`), drops the `hashchange` refetch, and reads a summary list (`GET /api/pins` without bodies; `GET /api/pins/{id}` keeps them) | 8 |
| Mobile: `#/pins/<id>` opens a read-only pin viewer (title, body, gallery) — enough to act on a reminder from the phone; the editor stays desktop-only in v2 | 9 |
| Per-tag cap of 40 characters; whitespace runs collapse to one `-` | 10 |
| A dropped file on a new pin uses the typed title or the file name, never "Untitled"; Cancel on a pin created this way offers Delete | 11 |
| RFC 6266 `filename*=UTF-8''…` | 12 |
| Typed store errors (`ErrInvalid`) mapped once in `statusForStore` | 13 |
| A startup sweep removes `pins/<id>` directories without a row (logged, counted) | 14 |
| Retained drafts (sessionStorage, mobile v2's rule): a detour or reload restores the text, with a Discard; `beforeunload` guards the tab close | 15 |
| Tests: handler tests for upload limits, sketch round-trip and `/scene`, id regex, limits as 400s, list payload shape, reminder `Due` table above, notice plan | 16 |

### List v2 (small, from the study)

- **Search** across title, tags and body: `GET /api/pins?q=` (SQLite
  `LIKE`, no FTS in v2), the sidebar's existing list-search pattern
  (`listSearch.js`).
- **Keep on top** (`pinned INTEGER` — the column is named `starred` to
  avoid "pinned pin"): starred first, then `updated_at DESC`.
- **Archive** (`archived_at`): out of the sidebar, in the search, reminders
  paused while archived.
- The sidebar card shows the next reminder as a muted line ("in 3 h",
  "tomorrow 09:00", "every day 09:00").

### Migration

`036_pin_reminders.sql` (035 became `pin_files.updated_at` in slice 1): the table above; `pins` gains `starred INTEGER
NOT NULL DEFAULT 0`, `archived_at TEXT`. `inbox_items.kind` and
`source_kind` are **CHECK-constrained** (`014_inbox.sql:9-10`), and SQLite
cannot alter a CHECK, so the table is rebuilt in place exactly as
`017_inbox_automation_source.sql` did for `automation` — `kind` gains
`reminder`, `source_kind` gains `pin`, indexes recreated with the same
names. `CountInboxBadge` counts a reminder as "other" (dot), never as
blocking (number); the Inbox app renders the new kind with the pin glyph
and its own action row (Done / Snooze / Open pin).

### Delivery slices (each closes green on its own)

1. **Hygiene** — the fix table minus the mobile viewer. No ADR (no boundary
   moves). **Done 2026-09-08** (owner-approved; handoff note
   `docs/handoff/2026-09-08-pins-v2.md`); the mobile viewer carries over. Changelog: "Pins: limits refuse instead of truncating; sketch
   annotate works on large images; edited sketches show their new preview;
   concurrent edits no longer overwrite each other; the sidebar follows the
   feed."
2. **Reminders, server** — migration, store, `internal/remind`, Inbox kind,
   `pin.reminded`, push case, OpenAPI. ADR-0100 accepted with this slice.
3. **Reminders, UI** — picker in the studio, sticky notice + collapse card,
   Inbox rows, announce and push prefs, sidebar line; mobile notice twin
   and the read-only viewer.
4. **List v2** — search, starred, archive.

Slice 2 and 3 land together in one deploy batch; a server that fires
reminders nobody can see is not "done".

### Acceptance (the visual verdict, on a scratch instance)

- Create a pin, set "In 1 hour" → the sidebar line reads "in 59 min"; set
  the clock forward on the scratch (or use a 5-minute interval) → the card
  appears, stays through a reload, closes on X, and is gone on the phone.
- "Every day at 09:00" in `America/Sao_Paulo` fires at 09:00 local on the
  scratch, not 09:00 UTC.
- Stop the daemon over a slot, start it → exactly one card, status "was
  due HH:MM".
- Five open reminders → one "5 reminders" card, Inbox lists five.
- Snooze 10 min → card gone, back after 10 min with the same key.
- Annotate a 6 MB screenshot → saves; edit the sketch → the thumbnail
  changes at once.

### Owner decisions needed before slice 2

| Question | Recommendation |
|---|---|
| Reminders as **Inbox items** (persisted acknowledgement, snooze for free, one notification center) vs a separate `pin_reminder_fires` table | Inbox. It is the product's existing "things owed to you" surface, ADR-0037 provenance already fits (`source_kind = pin`). |
| One reminder per pin in v2 | Yes — the control stays a chip; the table allows more later. |
| "Every 24 hours" as interval (drifts) **and** "Every day at HH:MM" as cron, both in the picker | Both, named plainly. |
| Missed slot policy: fire once at boot | Yes, matches automations and every benchmark that catches up. |
| Push with `requireInteraction` | Yes, mirrors the sticky card. |
| ADR number 0100 | Renumbered from 0099 at merge (0099 = package configuration GUI). |
