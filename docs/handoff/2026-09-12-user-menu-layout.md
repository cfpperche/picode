# 2026-09-12 — feat/user-menu-layout: layout selector out of the user menu

Shipped: `web/browser/src/components/UserMenu.jsx` no longer renders the
Layout group (Desktop / Auto / Mobile); the `readShellPref`/`setShell` import
is gone with it. One component serves the browser surface (`/browser/`), the
desktop shell (`/desktop/`, `inShell`) and the narrow/mobile layout, so all
three menus lose the control in one edit. Theme (Light / System / Dark) and
the install/version footer are untouched. `setShell` still lives in
`web/shared/client/shell.js` and is used by the phone app's More screen and
its Packages notice — intentionally out of scope (an action, not the radio
selector).

Verified: `make web` builds all four bundles; browser lib tests 294/294;
scratch instance (`qa-scratch.sh start user-menu-layout`, :8473) menu opened
at 1280p `/browser/`, 1280p `/desktop/` and 390p `/browser/` (narrow): only
the Theme radiogroup remains, overlay audit `ok: true` in all three, and the
trigger toggles the menu closed.

visual-review: PASS (`var/screenshots/um-browser-full.png`,
`um-desktop.png`, `um-narrow-scrolled.png`; card 5/5)

Not done / debts: none known.

Merge: fast-forward ready.
