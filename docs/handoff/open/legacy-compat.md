# Legacy compatibility paths

The low-risk group was retired 2026-09-25 (`feat/legacy-low-risk`). What is
left needs the owner's call before it goes.

## Debts

- [ ] GET /api/packages (legacy Pi JSON, Report.Legacy): only QA scripts and 4 Go tests read it; porting them to /api/packages/report is the cost. loadPackageReport/Report.Legacy have 5 internal callers.
- [ ] /api/pi-keys: no UI caller; 5 Go tests, the qa-cli-settings scripts and the docs (as "pi's own door") still use it.
- [ ] GET /api/packages/updates (LegacyUpdates): live caller in App.jsx; migrate that caller first.
- [ ] workspaces[].agent ("first agent, kept for older clients"): still read by Palette.jsx, App.jsx, tree.js, providerIcon.js.
- [ ] /ws/term?session=picode-<id> rewrite for old agent tabs.
- [ ] High risk: legacyInteractive Pi agents (ADR-0162) — needs the owner to confirm no pre-change session is alive.
- [ ] High risk: internal/tmux/drain.go default-socket drain (ADR-0139) — same confirmation.

## Kept on purpose (not debts)

- Plaintext credentials vault import; M1 workspaces.json import.
- Non-partitioned webapps; pre-ADR-0161 desktop shells; third-party CLI file formats.
- cli DEFAULT 'pi' and UI `cli || "pi"`: data rule (empty cli is Pi); server IsPi and others rely on it.
- notice.js toast shim.
