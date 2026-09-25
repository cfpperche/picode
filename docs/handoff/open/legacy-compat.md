# Legacy compatibility paths

The low-risk group was retired 2026-09-25 (`feat/legacy-low-risk`). What is
left needs the owner's call before it goes.

## Debts

- [x] GET /api/packages (legacy Pi JSON, Report.Legacy): only QA scripts and 4 Go tests read it; porting them to /api/packages/report is the cost. loadPackageReport/Report.Legacy have 5 internal callers. — retired 2026-09-25 (feat/legacy-api)
- [x] /api/pi-keys: no UI caller; 5 Go tests, the qa-cli-settings scripts and the docs (as "pi's own door") still use it. — retired 2026-09-25 (feat/legacy-api)
- [x] GET /api/packages/updates (LegacyUpdates): live caller in App.jsx; migrate that caller first. — not legacy: the live badge read for all CLIs; reclassified 2026-09-25 (feat/legacy-api)
- [x] workspaces[].agent ("first agent, kept for older clients"): still read by Palette.jsx, App.jsx, tree.js, providerIcon.js. — retired 2026-09-25 (feat/legacy-api)
- [x] /ws/term?session=picode-<id> rewrite for old agent tabs. — retired 2026-09-25 (feat/legacy-sessions-drain)
- [x] High risk: legacyInteractive Pi agents (ADR-0162) — needs the owner to confirm no pre-change session is alive. — retired 2026-09-25 (feat/legacy-sessions-drain)
- [x] High risk: internal/tmux/drain.go default-socket drain (ADR-0139) — same confirmation. — retired 2026-09-25 (feat/legacy-sessions-drain)
- [ ] TestOpencodeGUICredentials flakes under make ci-scoped load (30s timeout, 'OpenCode's server did not start'); passes alone in 3.6s — seen 2026-09-25
- [ ] TestAttachInterruptOnTmux/working_row_gone,_then_sends flakes under make ci-scoped load (409 not-stopped); passes 3/3 alone — seen 2026-09-25

## Kept on purpose (not debts)

- Plaintext credentials vault import; M1 workspaces.json import.
- Non-partitioned webapps; pre-ADR-0161 desktop shells; third-party CLI file formats.
- cli DEFAULT 'pi' and UI `cli || "pi"`: data rule (empty cli is Pi); server IsPi and others rely on it.
- notice.js toast shim.
