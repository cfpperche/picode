# 2026-09-12 — text panels

`feat/canvas-text-panels` → `main` (912c01e4). ADR-0108 amended; migration
**046** adds `content` to `canvas_panels`.

## The shape

Text is the first kind that is not a reference — the words *are* the panel.
It binds to its own id (so `ref NOT NULL` and `UNIQUE(canvas_id, kind, ref)`
hold and two empty ones never collide), the content is a column rather than a
table (nothing else in PiCode can point at a text block; a shareable one is
what a **pin** already is), and `canvas.panel.content` carries the whole
string so the last writer wins.

Saved on blur and after an 800 ms pause, flushed on unmount — the chunk
loader can unload a panel mid-sentence and must not eat it. 2 000 characters,
refused rather than truncated.

Placing one skips the picker (`PICKS_A_TARGET`): nothing to choose, so it is
born where it was drawn. The rectangle goes to `addPanel` **as an argument** —
`setPlaced` and `addPanel` run in the same tick, and state read there is the
previous rectangle.

## The bug the tests could not see

`MODEL_KEYS` decides when a memoized panel wrapper is stale. `content` was
missing from it, so after a reload the header showed the saved words (built
inside `buildModel`) and the box showed nothing. Any new field on a model
has to be added there or it is invisible to every memo on the plane.

## Verified

Isolated instance at `/browser/`: the tool row reads Agent / Terminal / Pin /
Text; drawing with **Text** armed created a panel at the rectangle with **no
dialog**; typing saved to the API within the pause; a reload came back with
the words in the box and in the header.

## Known red, not from here

`make ci-docs` fails on stale public captures — the same failure main already
has. `make deploy` recaptures, and the Canvas capture profile (the owner's
item 4) is the real fix.
