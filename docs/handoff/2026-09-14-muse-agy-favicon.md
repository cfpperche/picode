# 2026-09-14 — muse-agy-favicon: vendor marks for Muse Code and Antigravity

Shipped: Muse Code and Antigravity catalog badges now use vendor marks (Meta mark / Antigravity mark from the lobehub icon pins; vendor favicon as fallback). `CLI_FAVICONS` had no entry, so the badge fell back to the letters Mu/Ag.
Verified: scratch instance; all 8 catalog badges loaded (`naturalWidth>0`); screenshots `var/screenshots/muse-agy-favicons{,-tall,-mobile}.png` read; overlayAudit ok.
visual-review: PASS (var/screenshots/muse-agy-favicons{,-tall,-mobile}.png; card 5/5)
Also: test-only fix for a red main — `web/browser/src/lib/browserChannel.test.js` expects the `domains` field the nav-gate commit added to `browserChannel.js`. `feat/navgate-fix` carries the identical patch, so whichever lands first the other merges cleanly. Pruned a duplicated board bullet from `docs/handoff/2026-09-14-chart-today.md` (same item lives on `2026-09-13-dashboard-fleet`).
