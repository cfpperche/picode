# 2026-09-13 — snippets-v2c: capture and import (v2 authoring)

Shipped: a selection becomes a snippet — **Save as snippet** in the
composer toolbar, **Save selection as snippet** in the one context menu
(a field's own selection is now read from the field, which also
un-disabled **Copy** there). **Import** in the studio and on the phone
turns `[BRACKETS]` / `UPPER_CASE` into `{{lower_snake}}`, each
suggestion switchable, never inside an existing `{{…}}`.
`writeDraft(..., origin)` records why a draft exists, so a handoff is
not announced as "unsaved changes restored".
Files: `SnipCaptureSheet.jsx`, `snipDraft.js` (+ `titleFromText`,
`detectConversions`, `applyConversions`), `Snippets.jsx`,
`Composer.jsx`, `ContextMenu.jsx`, mobile `Composer.jsx` /
`SnippetsList.jsx` / `SnippetEdit.jsx`, `docs/architecture/snippets.md`.
Absorbs PRs 1–3 of `docs/plans/snippets-v2.md` (live validation, the
editable placeholder table, Try-it), merged today as `feat/snippets-v2a`
(`11d5038e`) and `feat/snippets-v2b` (`0fdeaa38`) without a note.

Verified: `make ci-scoped` PASS; 16 `snipDraft` tests; live on scratch
`v2c` — selection → sheet → Save (snippet created, draft untouched),
the context-menu row, import with both suggestions and with UPPER off,
mobile capture and import, `__picodeOverlayAudit()` ok on both dialogs.
Pixel PASS: capture sheet, import dialog (empty and filled), context
menu, mobile sheet and editor.

Debt: the board sits ~60 bytes under its 12 KB cap, so v2 PRs 5–8
(starters + duplicate, slug check + durable drafts, mobile editor v2,
docs guide) stay in `docs/plans/snippets-v2.md` rather than a bullet.
Merge: fast-forward ready.
