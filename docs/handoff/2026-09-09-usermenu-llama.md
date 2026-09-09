# 2026-09-09 — feat/usermenu-llama: llama.cpp in the desktop user menu

Shipped: the desktop user menu Tools group now lists **llama.cpp**
(`MENU_GROUPS`/`MENU_SECTIONS` in `web/desktop/src/lib/userMenuModel.js`,
icon `IconModel` in `UserMenu.jsx`), mirroring the mobile More screen; the
row is searchable like every other. Navigation reuses `go("llama")` →
`#/llama/models`; no route or API changes. The stale "mobile-only" comment
is gone. Architecture entry-point line updated; CHANGELOG entry under
[Unreleased].

Verified: model tests updated (8/8, incl. new search test); `make
ci-scoped` PASS; scratch QA (qa-scratch usermenu-llama) — menu open,
overlay audit ok, real click on the row navigates to #/llama/models and
closes the menu; search "llama" filters to llama.cpp. Screenshots read:
var/screenshots/um-llama-open.png, um-llama-search.png, um-llama-page.png.

visual-review: PASS (audit ok + card 5/5 on um-llama-open/search)

Not done / debts: QA trap found — a service worker left by a previous
scratch run on the same port served a stale bundle and faked wrong search
results until unregistered; future scratch QA should unregister SWs or
bust cache when results contradict the source.

Merge: fast-forward ready.
