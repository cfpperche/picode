# 2026-09-07 — feat/fix-favicon: favicon and PWA links restored on /desktop and /mobile

Shipped: the ADR-0072 split gave each app a base path, so Vite rewrote the
root-absolute brand links to `/desktop/favicon.svg`, `/desktop/manifest.json`
(and the mobile twins) — paths nothing serves; tabs fell back to the
placeholder icon and the manifest 404'd. `web/tools/vite-config.mjs` now has
a post-`transformIndexHtml` step that keeps the three brand links
(favicon.svg, apple-touch-icon.png, manifest.json) pointing at the site-root
files the launcher build ships, per app. CHANGELOG entry under [Unreleased].

Verified: scratch instance — `/favicon.svg`, `/manifest.json`,
`/apple-touch-icon.png` all 200; desktop and mobile HTML reference the root
paths; the SVG renders (screenshot read); `__picodeOverlayAudit()` ok;
`make close` green, captures refreshed after merging main (202d5d9a).

visual-review: PASS (fix-favicon-icon.png + fix-favicon-desktop.png; the
browser tab strip itself is chrome — the owner sees the icon on next reload,
hard-refresh if the old 404 is cached).

Not done / debts: handoff.md still carries the `/desktop/favicon.svg` 404
debt line — the shared root tree had another session's uncommitted
handoff.md (git-graph write actions), so this branch leaves it untouched;
drop the clause in the next handoff edit.

Merge: fast-forward ready.
