# 2026-09-24 — feat/mission-settings-deeplink: Pi Settings keeps workspaceId

Shipped: `#/clis/pi/settings?workspaceId=…` keeps that id through parse, reload, the layer switcher and Settings/Keyboard tabs. A workspace-only link names the folder on the project layer and does not infer an agent. `GET /api/pi-settings?workspace=` and PUT `workspaceId` load/write that folder. A missing id is “This workspace is no longer available.” plus Global settings.

Verified: `make close` after the tab-clip fix. Scratch `settings-deeplink` on :8472: phone-width Settings tabs read Terminals/Sessions/Providers/Settings with no leftover “nch”; invalid id still the error notice; desktop `#/clis/pi/settings` captured at the top (header + Global only). `__picodeOverlayAudit()` ok. Screenshots in `var/screenshots/mission-settings-deeplink/` (not committed). Blind spot: not run inside the Windows shell.

visual-review: PASS — scratch recapture after hiding partial pane tabs; 5-question card all yes/5.

Merge: fast-forward ready.

## Debts

- Mobile Omp Settings/Packages phone-width screenshot still open in docs/handoff/open/clis-workspace-names.md.
