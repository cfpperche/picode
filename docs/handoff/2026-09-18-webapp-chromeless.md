# 2026-09-18 — webapp-chromeless: installed web apps open app-mode in the shell

Shipped: owner chose option B (in-shell, no separate window). An installed
web app whose manifest declares `standalone`/`fullscreen`/`minimal-ui`
opens chromeless. First round kept a minimal titlebar; the owner cut it
the same day (`feat/webapp-appmode-fullbleed`, landed 4d11fe1d): the tab
surface is **entirely the page** — engine accelerators (F5 / Ctrl+R,
Ctrl ±), the page's own right-click menu, and Ctrl+F for find remain.
`display: browser` and plain work tabs keep the full toolbar. Installed
tabs are labeled with the app's name and manifest icon in the tab strip.
ADR-0147 amendments record both rounds; the standalone-window door stays
refused while the owner works in-shell.
Verified: `webappChromeless` matrix + launch-rule tests (8 JS tests);
scratch /desktop/ with a `__TAURI__` stub — full-bleed surface, Ctrl+F
find bar, regular work tab keeps its toolbar (screenshots read).
visual-review: PASS for layout; the page canvas is stub-blank by
construction — confirmed live by the owner (2026-09-18, both rounds:
app mode and the full-bleed cut).
Merge: landed and deployed the same day (captures refreshed by deploy).

## Next up

- (paid 2026-09-18) topic closed by the owner; record in open/installed-webapps.md.

## Debts

- docs/handoff/open/installed-webapps.md: standalone-window (Chrome-style) stays refused while the owner works in-shell; revisit costs an ADR.
