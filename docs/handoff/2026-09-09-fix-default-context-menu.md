# 2026-09-09 — feat/fix-default-context-menu: default menu rows match the terminal menu

Shipped: the desktop generic context menu (Copy, Paste, Reload PiCode,
Toggle theme) renders rows like the terminal menu again. Root cause: `Item`
in `web/desktop/src/components/ContextMenu.jsx` left the icon as a direct
child of `.um-item` (`justify-content: space-between`), so the label sat at
the far edge away from its icon; `TermRow` nests icon+label inside
`.um-item-name`. One-line structural fix, comment added. No shortcut column
on purpose: the app owns no copy/paste chords outside terminals
(termMenu.js: never advertise a chord the app does not answer).

Verified: `make web` PASS; scratch instance (qa-scratch fixctx) — default
menu open, rows aligned, overlay audit ok; real click on Toggle theme acts
and closes; terminal menu regression-checked, unchanged. Screenshots read:
var/screenshots/ctx-default-fixed.png, ctx-term-regression.png.

visual-review: PASS (audit ok + card 5/5 on ctx-default-fixed)

Not done / debts: none known. User-visible on production only after the
owner deploys (the report came from :8445).

Merge: fast-forward ready.
