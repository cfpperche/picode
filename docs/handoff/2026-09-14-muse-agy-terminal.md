# 2026-09-14 — muse-agy-terminal: Muse Code and Antigravity run in a PiCode terminal

Shipped: New terminal now works for Muse Code and Antigravity — the CLI runs in a PiCode tmux terminal with its own defaults and no adapter behind it.
Mechanism: catalog `surface` became capabilities — server sends `integrationCapable` and `launchable`; `cliPanes()` maps them to the pane list (these two get Launch + Terminals only); the new-terminal editor for them is name/workspace/folder only (no CLI combo, no profile, no Customize, no launch fields, no preview); `PUT /api/clis/{id}` stays refused (no launch settings); lifecycle stays refused as unmanaged; "Launch settings" is dropped from the terminal row menu (`termRowMenu`).
Verified: Go tests `TestCatalogCapabilities` + the extended `TestDetectOnlyCLIsAndMuseChannelCheck` (muse terminal created with `launchCli` muse and no injection plan) + `termRowMenu` test; scratch instance on an isolated port with a fake muse/agy on PATH — the terminal ran it (`var/screenshots/muse-terminal-open.png`); pane/editor/mobile/regression screenshots read; overlayAudit ok; `#/clis/muse/sessions` bounces to the launch pane.
visual-review: PASS (var/screenshots/muse-{launch-pane,new-editor,terminal-open,row-menu,mobile-pane,mobile-editor}.png + pi-regression.png; overlayAudit ok; card 5/5)
Not done: launch settings, sessions, activity state, resume and lifecycle for these two remain open. Antigravity update-check debt is already in `docs/handoff/open/agent-clis-native.md`.

## Next up

- Antigravity: launch settings and sessions remain open (muse sessions landed 2026-09-15).
