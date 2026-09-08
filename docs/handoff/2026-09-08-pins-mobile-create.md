# 2026-09-08 — feat/pins-mobile-create: reminder form + pins on the phone
Shipped: reminder picker rebuilt as a form (Once: date & time; Repeat:
every N hours / every N days at HH:MM, "count from when I close it"; one
Set + Remove; `formFromReminder` opens on the set rule) — presets and the
morning-hour pref removed; `remind.Rule.First` honours `At` on intervals
and the label says "at HH:MM" for day multiples; store passes `at` for
intervals. Phone: More → Pins list (search, starred first, `+`),
`#/pins/new` and `#/pins/<id>/edit` form (title, tags, markdown textarea,
retained draft, 409 kept), Edit on the pin screen; routes `pin`/`pinEdit`
under the More tab. Owner asked for both in chat (screenshot of the old
dropdown), 2026-09-08.
Verified: node tests (builder matrix, formFromReminder, routes), Go rule
tests (named first fire, past first refused, label); scratch :8476 —
desktop form sets "every 2 days at 18:30" (API interval 2880, at
21:30Z), reopens on the rule, refuses an empty date; phone creates,
edits and lists a pin end to end.
Not done / debts: the phone form has no files, sketch, reminder, star or
archive (desktop studio); agent-browser `fill` cannot type into
`<input type=time>` — set the value through the React setter in QA.
Merge: fast-forward ready after `make close`.
