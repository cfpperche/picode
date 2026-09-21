# 2026-09-21 — feat/drop-tab-view-toggle: drop Chat/Terminal tab-strip toggle

Shipped: removed `AgentViewToolbar` from the desktop/browser tab strip and the mobile agent header switch. Pi Chat versus Terminal is the sidebar / Work list agent menu (Open chat / Open terminal). Composer hint and `cli-terminal-launch.md` / `routes.md` match.

Verified: `make ci-scoped` PASS; scratch `drop-tab-view-toggle` on :8473. Browser 1440×900: tab strip is tabs + globe + inspector. Atlas ⋯ menu lists Open chat / Open terminal (`overlayAudit` ok). Mobile 390×844: header is Inspector / Project tools / Stop, no switch.

visual-review: PASS (scratch HTTP; not the Windows shell and not a physical phone).

Not done: none.

Merge: fast-forward ready after catching main.
