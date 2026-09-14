# 2026-09-14 — feat/apps-tab-search: Apps tab header + search, alphabetical tiles

Shipped: the Apps sidebar tab now follows the Pins pattern — `.pins-head`
title ("Apps"), `.pins-search` live filter ("Search apps", Escape clears),
tiles sorted by name via `visibleApps` (web/browser/src/lib/appsGrid.js,
base-sensitivity localeCompare, same as free agents). Sorting/filtering is
presentation-only; the server keeps registration order (Registry.All).
Empty states split: "No apps yet." vs "No apps match.".
Verified: make ci-scoped green (fmt, vet, hooks, test-js, build);
6 table-driven tests for visibleApps; scratch instance (qa-scratch :8474)
with screenshots read for full / filtered ("do" → Docker) / no-match
("zzz") states; real click on a tile opened #/app/docker; Escape restores;
state survives reload; __picodeOverlayAudit ok.
visual-review: PASS (apps-full.png, apps-filtered.png, apps-nomatch.png; card 5/5)
Not done / debts: none — apps are first-party built-ins, so the header
carries no add action by design.
Merge: fast-forward ready.
