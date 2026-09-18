# 2026-09-17 — installed-webapps: user-installed web apps as desktop shortcuts

Shipped: ADR-0147; `webapps` table (migration 053, renumbered after main
took 052) with in-transaction
`webapp.installed/updated/removed`; `/api/webapps` (list, resolve, install,
rename, delete, icon) with a bounded, DNS-rebind-guarded fetch — PWA manifest
first, favicon fallback, unreachable site refuses the install, loopback
first-class, URL kept as typed (fragment survives, redirects don't hijack).
Apps grid gains **+ Add**, tiles with icon endpoint (hard CSP/nosniff),
Open/Rename/Remove, opening the work browser on stable `w:app-<id>`; `(N)`
title prefix drives the badge. Adaptation study:
`docs/benchmarks/2026-09-17-installed-webapps.md`.
Verified: `make ci-scoped` PASS then `make close` green; scratch instance
(:8475) end-to-end with agent-browser — install/rename/duplicate(409 +
Open existing)/unreachable refusal/no-shell toast/remove, `overlayAudit`
ok on every overlay; Go store+server suites incl. ADR-0048 invariant rows.
visual-review: PASS (7 screenshots read; empty/hover/menu/dialog states).
Not done: shell-only paths unverified at ship time. Confirmed live by the
owner (2026-09-18, after deploy): the GitHub tile opens the webapp inside
the desktop shell with its login working. Still unobserved: the `(N)`
badge rendering on the tile and a login surviving a shell restart.
Dogfood found one real bug — github.com streams more than the 256 KB
body cap — fixed by sampling the page for metadata (55a7967a) and
redeployed the same night.
Merge: fast-forward ready after main merge.

## Next up

- Verify on the Windows shell: tile click focuses/reopens `w:app-<id>`, badge counts, login survives restart (shared profile).
- Refresh stale docs captures (`make docs-shots`) before the next deploy — Apps grid changed.

## Debts

- docs/handoff/open/installed-webapps.md: per-webapp WebView2 partitions (two accounts of one service) and standalone-window PWAs are v2, accepted in ADR-0147.
