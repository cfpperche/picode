# 2026-09-16 — cli-mobile-parity: Install/Update gated on the wrong flag

Owner reported lifecycle actions still missing on mobile after the muse/agy
lifecycle backend landed. Diff of `web/browser` vs `web/mobile` AgentClis.jsx:
Install/Update were gated on `cap.integration` (integrationCapable surface
flag) — inverted (`!cap.integration`) on mobile — instead of the lifecycle
flags the surrounding menu already used. Update-capable rows without
integration (muse everywhere; everything integrable on mobile) never showed
Update.

Fix (2 lines per file): both buttons now follow only `installed` +
`lifecycle.canInstall` / `lifecycle.canUpdate` + `diagnostic.updateAvailable`,
in browser and mobile identically. The problem-notice split (`!cap` vs `cap`
for docs-vs-repair notices) is intentional and untouched.

Verified: `make web` + `ci-scoped` PASS (incl. test-js); mobile scratch with
real vendor installs (agy 1.2.3 `updateAvailable` to 1.2.4): Update button
renders, menu holds Check for updates + Reinstall, guided uninstall line
present; desktop screenshot read. No JSX unit tests exist for these
components. Real update jobs not run (would mutate owner binaries). Deploy
not done (owner's call).
uiux-review: PASS (conditional fix, no new chrome; buttons/menu reuse existing primitives)
visual-review: PASS (var/screenshots/agy-mobile-update.png read; card 5/5)
