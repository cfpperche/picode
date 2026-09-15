# 2026-09-15 — launch-agy: editable Antigravity launch (Fatia 3b)

Fatia 3b of docs/plans/launch-muse-agy.md: editable Antigravity launch (owner approved 2026-09-15), right after deploying Fatias 1+2+3a to production.
Registry flip only: agy drops SurfaceTerminal (no surface:terminal rows remain), summary-only plan branch stating no hook surface; toggle/PUT/seed guards from 3a needed zero new code — hasIntegrationMechanism(agy) is false, PUT integration:true 400s, seed skips it.
Tests: TestCatalogCapabilities terminal set emptied (kept as the shape a future hookless CLI rejoins), lifecycle test asserts agy surface-absent + capable + PUT 200/400, mechanism table already generic. Stale comments updated (config, cliView, guide, architecture).
Verified on scratch: capable True mech False installed True; PUT 400/200; terminal creation 201 no launchError; desktop Launch tab screenshot read (note + editor, no toggle); row menu reads Rename/Launch settings/Terminal settings/Restart/Stop/Remove.
Still missing: native writers (Fatia 4), activity (Fatia 5, with the seed-flag revisit noted in 3a).
visual-review: PASS (var/screenshots/agy-launch.png read; card 5/5)
