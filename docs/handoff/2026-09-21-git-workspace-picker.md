# 2026-09-21 — git-workspace-picker: one Git workspace selector

Shipped: History and Delivery share the workspace selector in the Git header.
Workspace changes retain the current view, including existing repository tabs;
failed lookups preserve the current owner and view. Controls use the shared height.

Verified: make close passed; focused workspace-picker and route tests passed.
Scratch browser checks covered sibling/new/already-open repositories and failed lookup.
The architecture decision table records destination resolution and failure behavior.
visual-review: PASS — screenshots read in a separate reviewer, including both views,
open/no-results picker, empty/missing-target Delivery, and settled error toast;
overlay audits passed. Evidence: var/screenshots/git-workspace-picker/.
Browser screenshots do not establish physical Windows-shell acceptance.

Not done: no deployment; delivery/deploy observation scope is unchanged.
Merge: fast-forward ready after reconciling main; root landing still runs full CI.
