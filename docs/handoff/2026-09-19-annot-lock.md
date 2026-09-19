# 2026-09-19 — card loses the draft on a stray click (feat/annot-lock)

Owner: typing the note for pin 1 (Home), the card jumped to a new pin 2
and the draft evaporated. Any page click — a 1px miss on the input is
enough — created a pin and stole the card, discarding unsaved keystrokes.

## What landed
- Lock: with a card open, page clicks create nothing and pins/chips/menus
  do not reopen (script + fallback preview). Silent by design — the strip
  already teaches "Save the open note".
- Guards: Rust lock test (9/9). Chromium harness: stray click keeps 1 pin
  + intact draft + open card; Save then click pins normally.
- The lost draft itself is unrecoverable (it was never saved).

## Next up
- Owner: desktop-restart (new injected script), retype the Home note.
