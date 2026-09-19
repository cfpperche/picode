# 2026-09-19 — the label that clipped (feat/annot-style-label)

Seen in the step-5 visual review and fixed before closing it: the inspector's
64 px label column rendered "Background" as `Backgroun…`. Labels are now 72 px
and the card 328 px; measured with a Range over the text (66 px in a 72 px
box — `scrollWidth` cannot see this, it reports the box).

## Next up
- Nothing here; v2c step 5 is complete with this fix.

## Debts
- None. (The wider card costs 8 px of text-field width; the value fields are
  still ~172 px.)
