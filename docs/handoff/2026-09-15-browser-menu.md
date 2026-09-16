# 2026-09-15 — browser-menu: the ⋮ menu matches the reference, over the page

Shipped: the options menu (Find in page · Print · Zoom · Take a screenshot ·
Passwords and autofill › · Downloads · History · Clear browsing data ·
Browser settings), `btab_preview` + a hidden native view so the menu floats
over a frozen page (the `MENU_H` slide is gone), and application dialogs hide
the work webview (`appOverlays.js`, owner's "New workspace cut in half").
Find rides the WebView2 Find API, zoom/print the controller, and the dialog
items ride `browserDialogs.js` into Settings ▸ Browser (option A).
Chain: based on `fix/x-oauth-popup` + a merge of `feat/ask-prompt`, so the
three ship in that order without conflicts.
Verified: `cargo xwin build` ✓; web tests +4; scratch forced-render read
(menu, submenu, find bar; overlay audit ok) and the stub calls show
`btab_visibility false` while the menu still or an open dialog covers the tab,
`true` again on close.
visual-review: PASS (menu-open.png + menu-submenu.png + find-bar.png; the
still capture and native hiding are shell-only — owner's live check).
Not done: Show device toolbar, Import cookies and passwords (topic file).
Merge: last in the popup → ask-prompt → browser-menu chain.

## Next up

- Owner: with the shell restarted, use the ⋮ menu on a loaded page (the
  page stays put under it) and open New workspace over the page.
