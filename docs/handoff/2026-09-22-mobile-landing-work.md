# 2026-09-22 — mobile-landing-work: Landing work and workspace settings on the phone

Shipped: mobile Preferences gains the **Landing work** tab (`web/mobile/src/components/LandingWork.jsx`): machine
rules, "No rules yet" + Set rules, and the "Who follows" list; **Edit** opens `WorkspaceSettingsSheet.jsx`
(MobileSheet: name + Landing work, same choice and save table as the desktop dialog). The rules editor is a mobile
copy (`LandingRulesFields.jsx`) — the apps share only `web/shared`, where the logic already lives. `More` passes
`workspaces` and `fleetReady` into Settings. Both apps: with fast-forward off the empty line says only
"No checks.", and the machine-checks help speaks of workspaces.
Verified: `make ci-scoped` PASS; visual-review PASS on scratch at 390×844 over three rounds (44px targets, even
footer halves, focus on the sheet itself so no keyboard, focus back to Edit, 409 inside the sheet, dark mode,
no horizontal scroll, overlayAudit ok).
Pre-existing, left alone: a double divider under the Preferences tab bar; the mobile Preferences tab resets to
Appearance on reload.
