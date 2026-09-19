# 2026-09-19 — no limit on how many annotations (feat/annot-batch)

The owner's call: no cap on the number of annotations. Measured first: the
four-file ceiling is ADR-0089's (the door), one note staged 2 files, so 3+
pins failed the paste visibly and the rows stayed in the store.

## What landed (ADR-0152 amendment)
- **One note per Send** (per pin before): `POST /api/browser/annotations`
  takes `items[]`, stages one crop per pin and writes ONE document — count +
  page, then a section per pin. Rows stay one per pin (same note name).
- **A Send is staged whole or not at all**: every image is decoded before
  anything is written (a bad item no longer leaves half a package).
- The paste is the note plus the crops that fit the door's own cap
  (`pastePaths`, ADR-0089 untouched); the message says how many stayed behind.
  ADR-0089's cap now plainly means "attachments per paste", never "how much
  you may annotate".
- Tests: Go — 12-pin batch (one note, 12 crops, 12 rows, each pin's crop named
  in order), whole-or-nothing refusal, empty set; JS — `pastePaths`,
  the message's "N more screenshots" line, and a guard that the Send posts one
  body.

## Next up
- Owner: deploy + `make desktop-restart` is NOT needed (no shell change), then
  a Send with three or more pins. Expected: one message, one note, crops
  attached up to four files.

## Debts
- A Send of more than three pins attaches only the first crops; the rest are
  named in the note. If the owner wants every picture attached regardless,
  that is ADR-0089's cap and a decision of its own.
