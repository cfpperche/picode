# Study: pin reminders and sticky notifications

- **Date:** 2026-09-08
- **Sources:** vendor documentation read on 2026-09-08 (list at the end).
  Facts that only third-party write-ups confirm are marked *third-party*;
  what no source documents is marked *not documented*. Nothing was
  cloned or run.
- **In-house counterpart:** `internal/store/pins.go`, `internal/automate`
  (ADR-0045 scheduler), `internal/store/inbox.go` (snooze column),
  `web/shared/domain/notice.js` (sticky needs-you card),
  `internal/push/notifier.go` + `web/public/sw.js`.
- **Scope:** how the products the owner's audience already uses model a
  reminder's cadence, what they do when a reminder is missed or done, how
  they keep a notification on screen until a person acts, and what makes
  a pinned note feel worth keeping. What PiCode adopts; what it refuses.
  The review of pins v1 and the plan are `docs/plans/pins-v2.md`.

## Why now

The owner asked for pins v2 with reminders: a pin carries a cadence — one
date and time, or every N hours — and PiCode shows a reminder card that
stays until the person closes it. Before designing that, the question is
what the mature products settled on, because every one of them shipped a
first version that piled up missed occurrences or auto-closed the card.

## Cadence models

| Product | One-shot | Recurring | Sub-day | Timezone | Snooze | Missed | Done vs recurrence |
|---|---|---|---|---|---|---|---|
| Apple Reminders | date + time, "early reminder" offset | presets hourly…yearly + custom = frequency × interval, "end repeat" | yes (hourly, every N) | device local | notification "Later → choose when" | *not documented*; a one-shot "reappears until you complete it" | completing advances to the next instance on the fixed schedule |
| Google Keep | date + time; Morning/Afternoon/Evening chips with user-set hours | daily / weekly / monthly / yearly / custom with end date (*third-party*) | no | *not documented* | none — it re-notifies **once, 24 h later**, then stops | one re-ping | reminders now live in Google Tasks |
| Todoist | natural language ("in 3 hours") | NL: "every 12 hours starting at 9pm", "every mon, fri at 20:00", "everyday for 3 weeks" | yes | *not documented* | user-configured snooze list | "only schedules recurring tasks on **future** dates" — no pile-up | `every` = next from the **schedule**; `every!` = next from **completion** |
| Notion | `@remind tomorrow`, date-property offsets | **none** (roadmap) | — | explicit tz on the date property | none | — | — |
| Slack | Remind me: in 20 min / 1 h / 3 h / tomorrow / next week / custom | every day / weekday / Tuesday / two weeks | **refused** ("not in time increments") | day-only reminders default to **9 a.m. in your zone** | reschedule from Later | waits in Later → In progress | complete from Later |
| Microsoft To Do / Outlook | later today / tomorrow / next week / pick | daily / weekdays / weekly / monthly / yearly / custom; Outlook adds **"regenerate N days after completion"** and **skip occurrence** | no | *not documented* | Outlook snooze / dismiss / dismiss all | Q&A threads report dropped and stuck reminders on recurring tasks | completing creates the next instance |
| Things 3 | date + time on the template | "after completion" **or** fixed with day filters | no | device local | — | only the next copy exists | rescheduling asks **Make exception** vs **Update rule** (3.23) |
| Obsidian reminder plugin | inline `(@date time)` on a checklist item | via the Tasks plugin | — | local | modal: done / later (presets) / mute / open note | mute persists; nothing fires while the app is closed | done checks the box |
| Linear | inbox snooze: an hour, tomorrow, next cycle; NL date | **none** | — | *not documented* | snooze hides; "Remind me" shows a **banner on the issue** | reappears in Inbox | reschedule or cancel from the banner |
| GitHub | no per-user remind; team scheduled reminders → Slack | days × times with explicit tz | — | tz dropdown | — | — | — |

Three things every product that got it right shares:

1. **Two recurrence anchors** — from the schedule (Todoist `every`, Things
   fixed) and from completion (Todoist `every!`, Things "after
   completion", Outlook "regenerate"). The tools with only the first one
   own the "reminder stuck in the past" support threads.
2. **Nobody replays missed occurrences.** Todoist jumps to the next future
   date, Keep re-pings once, Slack leaves the item in Later. Quartz names
   this `FIRE_ONCE_NOW` and systemd `Persistent=true` ("triggered
   immediately if it would have been triggered at least once while the
   timer was inactive" — at most once).
3. **"Every 24 hours" and "every day at 09:00" are different features.**
   RFC 5545 expands a calendar rule in a named zone (`DTSTART` + `TZID`);
   instances at a non-existent local time (the DST gap) "MUST be
   ignored". A duration rule drifts an hour against the clock after each
   transition. Storing one as the other is the classic bug.

## Sticky notifications

| System | Auto-dismiss | What makes it stay | Cap / overflow | A11y |
|---|---|---|---|---|
| VS Code | toasts hide on a timer **unless they carry actions** | actions; the notification center keeps items "until deleted pressing the X" | at most 3 toasts; with the center open, no toast — the item goes to the top of the list | `focusToasts` / `focusNextToast` commands; bell badge; Do Not Disturb (errors still through) |
| Sonner (ours) | 4 000 ms | `duration: Infinity`; timers pause on hover, interaction and hidden document | `visibleToasts` = 3, rest queued | list `aria-live="polite"`, hotkey Alt+T; `id` dedupes and updates |
| Radix Toast | 5 000 ms | per-toast `Infinity`; pauses on hover, focus, window blur | viewport region "Notifications (F8)" | `Toast.Action` **requires `altText`** |
| Slack / Linear / Notion | OS push + badge | the item lives in Later / Inbox until acted on | a list, not a stack | keyboard model (`J/K`, `H` snooze) |
| Web Notification API | Chrome desktop ~20 s | `requireInteraction: true` — **Chromium only, ignored on Android, absent on Safari**; `actions[]` only from a service worker, **absent on Safari/iOS** | `tag` replaces, `renotify` re-alerts; merge into "N reminders" via `getNotifications()` | OS |

ARIA: `role="alert"` is assertive and "must be used sparingly"; if the
user is expected to close it, MDN says use `alertdialog`. `role="status"`
is polite and must not take focus. The workable mapping for a card that
waits: announce arrival politely, render the card focusable inside a
labelled region with a hotkey, never steal focus, label every button with
the reminder's title.

## Pinned notes

| Product | What makes it feel good |
|---|---|
| Google Keep | pin to top of the feed; labels; archive distinct from delete |
| Apple Notes | pinned notes "always appear at the top", synced; tags; checklists |
| Raycast Notes | one hotkey away; pin binds to ⌘0–⌘9; markdown with checklists |
| Obsidian Bookmarks | pins *targets* (notes, headings, blocks, searches) into a sidebar |
| Slack Later | saved item + reminder time; In progress → Archived / Completed |
| Linear "Remind me" | the reminder is a **banner on the object**, reschedulable in place |

Common denominators: sub-second capture, pinned-to-top as an **ordering
rule** rather than a folder, tags plus search, an **archive** state, a
checklist in the body, and a reminder that surfaces the note *in place*.

## What PiCode adopts

| Adopt | From | Into |
|---|---|---|
| Presets first, custom last: in 1 h / in 3 h / tomorrow 09:00 / next Monday 09:00 / pick date & time; every day at HH:MM / every weekday / every N hours / cron | Slack, To Do, Apple, Todoist | the studio's "Remind me" chip |
| A user-editable **morning hour** (default 09:00) behind every day-only preset | Slack, Keep | Preferences → Notifications |
| Two anchors: **schedule** and **completion** (`anchor` column) | Todoist `every!`, Things, Outlook | `pin_reminders.anchor` |
| Calendar rules in a named zone; durations as minutes; never `+24h` for "daily" | RFC 5545, Nylas | `kind` = once / cron / interval, `tz` stored per reminder |
| Missed slot fires **once**, then the next future instance | Quartz `FIRE_ONCE_NOW`, systemd `Persistent`, Todoist | `remind.Due` |
| The fired-but-unacknowledged state lives in the database, not the tab | VS Code notification center, Slack Later, Linear Inbox | an Inbox item per fire; the card is a projection of it |
| Sticky card with a hard cap; above it, one "N reminders" card that opens the list | VS Code "at most 3", `getNotifications()` merge | `notice.js` + Inbox app |
| Snooze moves this occurrence only, never the rule | Things "make exception", Linear `H` | `inbox_items.snoozed_until` |
| Push with `tag` + `renotify` + `requireInteraction` as progressive enhancement; `Topic` = reminder id so retries collapse | RFC 8030, Chrome, web.dev | `notifier.go`, `sw.js` |
| `role="status"` arrival, focusable card in a hotkey region, no focus steal, buttons named with the pin title | MDN, Radix, Sonner | `Notice.jsx` |
| Keep on top as an ordering rule; archive; search | Keep, Apple Notes, Raycast | `pins.starred`, `pins.archived_at`, `?q=` |

## What PiCode refuses

- **RRULE strings.** RFC 5545 is the right reference but a parser is a
  dependency or a week of code; `internal/cron` already covers daily,
  weekday, weekly and hourly in the stdlib and the Automations editor
  already has the preset layer (`web/shared/domain/cron.js`). Monthly
  "last Friday" and yearly rules are not asked for; if they are, that is
  the moment to revisit.
- **Replaying missed occurrences.** Every source that tried it regretted it.
- **Natural-language dates.** Todoist's NL parser is its product; ours
  would be a worse copy. Presets plus a date-time input reach the same
  cases the audience needs.
- **A homemade notification center.** The Inbox is one already (ADR-0037),
  with snooze, done and provenance; reminders join it rather than
  spawning a sibling.
- **`role="alert"` for reminder cards.** Assertive interruptions for
  something the person scheduled themselves.

## Sources

Apple Reminders: https://support.apple.com/guide/reminders/add-dates-or-locations-to-reminders-remnd4b206fb/mac · https://support.apple.com/guide/reminders/manage-reminder-notifications-remn4e53b572/mac · *third-party* https://www.howtogeek.com/681620/how-to-set-up-recurring-reminders-on-iphone-and-ipad/
Google Keep: https://support.google.com/keep/answer/3187168 · https://support.google.com/tasks/answer/16540694 · https://support.google.com/keep/answer/6191044
Todoist: https://www.todoist.com/help/articles/introduction-to-recurring-dates-YUYVJJAV · https://www.todoist.com/help/todoist/features/introduction-to-reminders-9PezfU
Notion: https://www.notion.com/help/reminders
Slack: https://slack.com/help/articles/208423427-Set-a-reminder · https://slack.com/help/articles/13453851074067-Save-it-for-Later · https://docs.slack.dev/reference/methods/reminders.add
Microsoft: https://support.microsoft.com/en-us/office/add-due-dates-and-reminders-in-microsoft-to-do-064d9696-08d1-4433-bfdd-f661dc97491f · https://support.microsoft.com/en-us/office/turn-off-or-postpone-a-reminder-03751562-9584-490f-bea7-fd70bb69ba79 · https://learn.microsoft.com/en-us/previous-versions/office/developer/office-2007/bb647540(v=office.12) · https://learn.microsoft.com/en-us/answers/questions/4639753/reminders-for-recurring-tasks-on-microsoft-to-do-s
Things 3: https://culturedcode.com/things/support/articles/2803564/ · https://culturedcode.com/things/blog/2026/08/repeating-to-dos-refined/
Obsidian: https://uphy.github.io/obsidian-reminder/guide/notification.html · https://obsidian.md/help/plugins/bookmarks
Linear: https://linear.app/docs/inbox · https://linear.app/changelog/2023-01-31-issue-reminders
GitHub: https://docs.github.com/en/organizations/organizing-members-into-teams/managing-scheduled-reminders-for-your-team · https://github.com/orgs/community/discussions/17653
Raycast: https://www.raycast.com/raycast/apple-reminders · https://manual.raycast.com/notes
Apple Notes: https://support.apple.com/guide/notes/sort-and-pin-notes-apdb54e469b6/mac
VS Code: https://code.visualstudio.com/api/ux-guidelines/notifications · https://github.com/Microsoft/vscode/issues/44319
Sonner: https://sonner.emilkowal.ski/toaster · https://github.com/emilkowalski/sonner/blob/main/src/index.tsx
Radix Toast: https://www.radix-ui.com/primitives/docs/components/toast
ARIA: https://developer.mozilla.org/en-US/docs/Web/Accessibility/ARIA/Reference/Roles/alert_role · https://developer.mozilla.org/en-US/docs/Web/Accessibility/ARIA/Reference/Roles/status_role
RFC 5545: https://www.rfc-editor.org/rfc/rfc5545.txt · Nylas: https://www.nylas.com/blog/calendar-events-rrules/
Quartz misfire: https://nurkiewicz.com/2012/04/quartz-scheduler-misfire-instructions.html · systemd.timer: https://man7.org/linux/man-pages/man5/systemd.timer.5.html
RFC 8030: https://www.rfc-editor.org/rfc/rfc8030
Notifications: https://developer.mozilla.org/en-US/docs/Web/API/ServiceWorkerRegistration/showNotification · https://developer.mozilla.org/en-US/docs/Web/API/Notification/requireInteraction · https://developer.chrome.com/blog/notification-requireInteraction · https://web.dev/articles/push-notifications-common-notification-patterns · https://caniuse.com/mdn-api_notification_requireinteraction · https://caniuse.com/mdn-api_notification_actions
