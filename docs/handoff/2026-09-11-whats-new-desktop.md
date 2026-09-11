# 2026-09-11 — feat/whats-new-desktop: a release board, not a 400 px column

Shipped: the desktop What's new dialog opens at 880 px (≥900 px viewports) with
its highlights in a two-column grid of hairline cards; 720–899 px keeps the
intended 560 px, <720 px is unchanged (sheet, ADR-0046). Both shells now map
every published `whats-new.json` icon name (desktop `matrix` → Canvas glyph,
phone → panel grid) and `web/tools/release-note-icons.test.mjs` fails on a name
without one. Files: `web/desktop/src/styles/app.css`, desktop + mobile
`components/WhatsNew.jsx`, `web/tools/release-note-icons.test.mjs`.

Verified: `make ci-scoped` PASS. Scratch instance :8472 (seeded, dark + light):
9 of 9 highlights visible with no scroll; 1700×1000 → 105/105 margins,
1600×800 → 48/48, 1600×700 → 42/42 (internal scroll, never clipped);
390×844 stays a sheet (padding 8/18/20, one column); `__picodeOverlayAudit()`
ok; empty state forced with `-X internal/version.Version=0.0.1` and read.

visual-review: PASS (`var/screenshots/whats-new-desktop-before-after.png`,
`…-after-centered-light-1700.png`, `…-empty-880.png`; card 5/5)

Not done / debts: no headline treatment — the nine highlights carry equal
weight; the header's right side stays empty; the release date sits 880 px from
its version label. Two cascade traps survive in `app.css`: `.dlg-whats-new` is
declared before `.dlg`, so a later generic `.dlg*` rule still wins silently on
width/padding (three instances fixed here; `.dlg-actions`'s `margin-top: 16px`
left as authored-by-accident), and any new early `.dlg-*` block has the same
shape.

Merge: fast-forward ready (main moved since the branch started; `main` merged
into the branch before `make close`).
