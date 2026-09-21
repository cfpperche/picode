# 2026-09-20 — feat/browser-computer-ux: browser computer UX pass
Shipped: full-width set-page + 36px ctl-h; workspace chips via /api/workspaces; grants toolbar (search/ws/tier, paging 100); expandable audits (search/outcome, paging 50); audit limit 20→100.
Verified: qa-scratch — grants search 1-of-2, refused-step expand, overlay audit ok; ci-scoped + make close green (blind spot: not run inside the Windows shell).
visual-review: n/a
Not done / debts: none new; virtualization only if paging measures badly at 1000 rows (see docs/handoff/open/work-browser-tabs.md).
Merge: not merged.
