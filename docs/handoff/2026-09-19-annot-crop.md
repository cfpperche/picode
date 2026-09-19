# 2026-09-19 — the annotation arrived, the picture did not (feat/annot-crop)

The owner's first successful Send landed in the agent session (message +
`@.picode/drop/annotation-….md`), and the row showed `shot: ""`. The note,
the HTML and the computed styles were all there; the crop never was.

## Root cause
`cropFromPreview` called `btab_preview` with the **React** tab id (`w:3`);
`capture_png` resolved `label(id)` only, so `btab-w:3` never existed and the
invoke rejected into a `.catch` — a silent failure inside a Send that
otherwise worked. Same id-shape family as the earlier "no such tab" trap,
one call site deeper.

## What landed
- `find_webview(app, id)`: label, then the id's tail — used by `capture_png`
  and the three annotate commands (which each had their own copy).
- The crop's call site passes the native id, with the reason in a comment.
- Guards: JS source scan (14/14 on the annotate file; it fails if the
  prefixed id returns), `make desktop-shell` cross-build green.

## Next up
- Owner: desktop-restart + deploy, then Send one note with the camera on —
  the note and the crop should both land, `shot` non-empty.

## Debts
- The crop itself is Windows-only (WebView2 capture): the evidence here is
  the resolver, the cross-build and the guard; the live shot is the owner's.
