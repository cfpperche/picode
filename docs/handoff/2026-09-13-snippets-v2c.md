# 2026-09-13 — snippets-v2c: capture and import (v2 authoring)

Shipped: selection → **Save as snippet** in the composer toolbar and
**Save selection as snippet** in the one context menu (a field's own
selection is now read from the field, which also un-disabled **Copy**);
**Import** in the studio and on the phone converts `[BRACKETS]` /
`UPPER_CASE` → `{{lower_snake}}`, each suggestion switchable, never
inside an existing `{{…}}`; `writeDraft(..., origin)` records why a
draft exists, so a handoff is not announced as "unsaved changes
restored". Files: `SnipCaptureSheet.jsx`, `snipDraft.js` (+
`titleFromText`, `detectConversions`, `applyConversions`), `Snippets.jsx`,
`Composer.jsx`, `ContextMenu.jsx`, mobile `Composer.jsx` /
`SnippetsList.jsx` / `SnippetEdit.jsx`, `docs/architecture/snippets.md`.

Absorbs PRs 1–3 of `docs/plans/snippets-v2.md` (live validation, the
editable placeholder table, Try-it), merged earlier today as
`feat/snippets-v2a` (`11d5038e`) and `feat/snippets-v2b` (`0fdeaa38`)
without a note of their own.

Verified: `make ci-scoped` PASS; 16 `snipDraft` tests; live on scratch
`v2c` — selection → sheet → Save (snippet created, draft untouched),
context-menu row, import with both suggestions and with UPPER switched
off, mobile capture + import, `__picodeOverlayAudit()` ok on both
dialogs. Pixel PASS: capture sheet, import dialog (empty/filled),
context menu, mobile sheet and editor.
Merge: fast-forward ready.

Debt: the board sits ~47 bytes under its 12 KB cap, so v2 PRs 5-8
(starters + duplicate, slug check + durable drafts, mobile editor v2,
docs guide) live in `docs/plans/snippets-v2.md` instead of a bullet.
