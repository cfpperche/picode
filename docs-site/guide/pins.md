---
description: Notes with attachments, sketches and reminders, kept beside your agents rather than in another app.
---

# Pins

A pin is a note that lives in PiCode: a title, tags, a markdown body,
attachments, drawings, and — if you want one — a reminder. **Pins** is a
tab in the sidebar; on the phone it is under **More → Pins**.

Pins belong to the machine, not to a workspace or an agent. They are for
the things you would otherwise leave in a scratch file or another app: the
command you keep re-deriving, a screenshot of the bug with arrows on it,
what to pick up tomorrow.

## Writing one

The editor takes markdown and keeps it as markdown — what you type is what
is stored, not a conversion of it.

**Attach** adds files. You can also drag them onto the editor or paste them
straight in. **Sketch** opens a drawing canvas; drop an image into a sketch
and you can annotate it, which is the fast way to point at something in a
screenshot.

Limits are refused, never silently trimmed — if a pin is too big, PiCode
says which limit and by how much rather than cutting your text:

| | Limit |
|---|---|
| Title | 200 characters |
| Tags | 16 tags, 40 characters each |
| Body | 100 KB |
| Images | 8 MB each |
| Other files | 16 MB each, 24 files per pin |
| One drawing | 2 MB |

An unsaved draft is held in the browser until it matches the saved copy
again, so a closed tab does not lose what you were typing. If the same pin
changed somewhere else while you were editing, saving stops and tells you
instead of overwriting the other version.

## Finding one

The list is newest first, with **starred** pins on top. Starring is not an
edit — it does not change the pin's "last updated" time.

The search box matches on title, tags and body: every word you type has to
appear somewhere in the pin. It searches archived pins too and shows live
ones first.

**Archive** takes a pin out of the live list without deleting it; the list
shows how many archived pins there are, and archiving pauses any reminder
the pin has. Taking it out of the archive starts it again.

## Reminders

Open a saved pin and the reminder control sits beside Attach and Sketch.
Nothing is preset — you choose:

- **Once** — a date and a time.
- **Repeat** — every N hours, or every N days at a set time.
- **Count from when I close it** — the next one is scheduled from when you
  mark the reminder done, not from the clock. Use this for "check again a
  week after I actually deal with it".

Repeating reminders run in the time zone your browser is in.

When a reminder is due it becomes an **Inbox** item and a card that stays on
screen until you act on it. The card has **Open** (goes to the pin),
**Snooze**, and a close button that marks it done. Because the card is just
a view of the Inbox item, closing or snoozing it on your phone withdraws it
from your desktop too.

If a reminder came due while PiCode was not running, you get it on the next
load and it says when it was actually due. Nothing is dropped quietly.

With [Web Push](/guide/mobile) enabled and **reminders** turned on in your
notification preferences, a due reminder also reaches you as a push
notification.

## Where pins show up elsewhere

- **[Canvas](/guide/canvas)** — a pin can be a panel, showing its title and
  tags beside your agents and terminals.
- **Inbox** — due reminders arrive here, with an **Open pin** action.
- **The phone** — read a pin under the Inbox tab; create and edit from
  **More → Pins**. The full editor, with attachments and sketches, is on the
  desktop.

## What pins are not

- **Not per-workspace.** Every pin is visible from everywhere in PiCode.
- **Not shared.** Pins are on your machine, behind the same pairing gate as
  the rest of PiCode ([Security and pairing](/guide/security)).
- **Not a wiki.** There is no linking between pins and no folder tree —
  tags, starring and search are the whole organization.
