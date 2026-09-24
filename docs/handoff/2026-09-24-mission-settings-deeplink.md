# 2026-09-24 — feat/mission-settings-deeplink: Pi Settings keeps workspaceId

Shipped: `#/clis/pi/settings?workspaceId=…` keeps that id through parse, reload, the layer switcher and Settings/Keyboard tabs. A workspace-only link names the folder on the project layer and does not infer an agent. `GET /api/pi-settings?workspace=` and PUT `workspaceId` load/write that folder. A missing id is “This workspace is no longer available.” plus Global settings.

Verified: `make close` PASS after merging main (fmt,vet,hooks,go[4],test-js,build,living-docs). Scratch `settings-deeplink` on :8473: desktop and phone-width (`?mobile=1`, 390×844) showed Edit Global | QA with no agent, invalid id as the error notice, absent id as Global only. `__picodeOverlayAudit()` ok. Screenshots in `var/screenshots/mission-settings-deeplink/` (not committed). Blind spot: not run inside the Windows shell.

visual-review: PASS — scratch instance; 5-question card all yes/5.

Merge: fast-forward ready.

## Debts

- Mobile Omp Settings/Packages phone-width screenshot still open in docs/handoff/open/clis-workspace-names.md.
