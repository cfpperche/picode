# 2026-09-07 — desktop-usermenu-v2: user menu refactored to the mobile v2 pattern

Shipped: `web/desktop/src/components/UserMenu.jsx` now groups rows with
subtitles (Tools / Agents and connections / PiCode / Continue), has an
in-menu search, an empty state with Clear search, and hides the footer
while searching. Theme/layout radios stay (Vercel/Geist pattern). The
matcher moved to `@picode/shared/domain/listSearch.js` (mobile re-exports
it); menu model + decision-table tests in
`web/desktop/src/lib/userMenuModel.js`. Only destinations with desktop
routes are listed (Apps, llama.cpp, Notifications stay mobile-only).
Verified: web tests 395 pass, `make ci-scoped` PASS; scratch instance on
:8471 — overlayAudit ok; real clicks: theme radio (light→dark), Providers
item navigates to `#/providers` and closes the menu.
visual-review: PASS (open + scrolled, light + dark, search match, empty search; card 5/5)
Not done / debts: a Playwright click aimed at a below-fold radio can race
the popover's programmatic scroll (harness flake; after scrollIntoView the
events land inside the menu and the action applies).
Merge: fast-forward ready.
