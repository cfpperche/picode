# 2026-09-11 — feat/app-boundary: an app does not leak into PiCode's interface

**Shipped.** ADR-0109's 2026-09-11 amendment writes the owner's directive as a project rule and, crucially, the **doors**: the tile and its badge, the main tab, the app's own body, the manifest `icon` key the host's map draws, the `host` object, the `#/app/<id>` route. Direction is the test — the host may name an app, an app may never name a host surface. One sentence in AGENTS.md § Style points at it.

Obeying it: Preferences → Appearance lost **CANVAS BACKGROUND** and the `⋯` menu lost `Background…`; the control is now a **Background** submenu on that same menu (Radix `DropdownMenu.Sub`, four `RadioItem`s with real pattern samples, a tick on the current one, `onSelect` prevented so the plane behind stays the preview). `PatternSwatch.jsx` moved into `components/canvas/`, its CSS from `app.css` to `canvas.css`. `canvasPattern.js` did **not** move — same key, same shape; only the control was misplaced. Messages' audit list is declared a door (the grant lives in `peer_connections`, a canvas is only where it was drawn) and reworded as Messages': `CanvasLinks.jsx` → `GrantedContacts.jsx`, **Granted contacts**. `web/tools/app-boundary.test.mjs` enforces the direction in `make test-js`.

**Verified.** `make close` green; scratch `app-boundary` (:8473) with `agent-browser`. Preferences → Appearance = one group (**Theme**), zero "canvas" strings; `⋯` → Background lists Plain/Dots/Grid/Cross with the current one ticked; picking repaints the plane behind the open menu and survives a reload (`data-bg=cross`); keyboard opens (`↓`×5, `→`), walks, selects (Enter) and leaves (`←` submenu, Esc all, focus back on `⋯`); `__picodeOverlayAudit()` `ok` in both themes with it open. Sweep: "Canvas" appears only on its tile, its tab and Messages' declared section — nothing in the Inspector or Preferences.

**visual-review: PASS** (ab-prefs-{light,dark}, ab-bgmenu-{dark,light}, ab-bgmenu-grid-dark, ab-messages-granted-{light,dark}; card 5/5).

**Debts.** No docs-shots capture of the submenu (the existing Canvas capture-profile debt covers it). The guard test names the Canvas by path; a second native app adds a row. Escape inside the submenu closes the whole menu — Radix's own behaviour; `←` is the submenu-only exit.

**Merge.** Fast-forward ready; floating edges land on this branch next.
