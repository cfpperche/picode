# 2026-09-08 — feat/mobile-ws-favicon: workspace favicons on the phone's Work list

Shipped: the Work screen's workspace groups (`web/mobile/src/screens/Work.jsx`)
now wear the project's favicon like the desktop sidebar — new
`web/mobile/src/components/WsFavicon.jsx` (same `hasFavicon` advertisement +
`GET /api/workspaces/{id}/favicon`, folder-icon fallback, per-page-load
failure memory), one CSS rule in `mobile-lists.css`. No API change.

Verified: `make ci-scoped` PASS; scratch instance (qa-scratch, :8471) with a
`public/favicon.svg` project, the picode worktree and a no-icon folder —
favicon, fallback and pi mark all render at 17px, rows aligned,
`__picodeOverlayAudit()` ok. Screenshot: `var/screenshots/mobile-work-favicons-390.png`.

visual-review: PASS (mobile-work-favicons-390.png read; card 5/5; audit ok)
Not done / debts: none known — desktop already had the behavior.
Merge: fast-forward ready.
