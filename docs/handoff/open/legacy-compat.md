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
- [x] stripLegacyUserClaudeHooks + claudeSetWiring + groupHasMarker (owner's ~/.claude/settings.json had no PiCode hook) — retired 2026-09-25 (feat/legacy-dead-code)
- [x] inbox-burst reconciliation (production DB held 0 such tasks; ADR-0059 amended) — retired 2026-09-25 (feat/legacy-dead-code)
- [x] #/clis/<cli>/launch alias rewrite — retired 2026-09-25 (feat/legacy-dead-code)
- [x] /api/apps/inbox/{view,action} aliases (UI uses /api/inbox; ADR-0208 amendment extended) — retired 2026-09-25 (feat/legacy-dead-code)
- [x] Pre-stamp port fallback in apps/tmux.go tmuxElsewhere + Host.LoopbackURL (all 20 live sessions stamped; ADR-0141 amended; unstamped sessions are now reapable) — retired 2026-09-25 (feat/legacy-dead-code)
- [x] WebTab pre-ADR-0161 hide-and-freeze path (coverDecision, verifyPreviewUrl, subscribeFloatingLayers, still backdrop, data-covered; ADR-0161 amended) — retired 2026-09-25 (feat/legacy-dead-code)
- [x] Pi package mutation responses in pipkg shape: POST/DELETE /api/packages and update now answer the unified report (writePiPackageReport) — retired 2026-09-25 (feat/legacy-dead-code)
- [ ] Owner: first live run of the work-browser tab after the pre-ADR-0161 path removal (menu/palette/toast over a web page)
- [x] preview.rs comment still calls btab_preview the "legacy hide/capture path"; it now serves annotation captures only — fix at the next shell rebuild — paid 2026-09-25 (feat/legacy-sweep-2)
- [x] QA scripts on retired links (qa-cli-settings, -recovery, qa-cli-packages, qa-cli-providers, qa-llama, qa-mobile-settings) moved to canonical CLI addresses; qa-cli-providers' unsupported row now an unknown CLI id — retired 2026-09-25 (feat/legacy-sweep-2)
- [x] cmd/uicheck (stale, unreferenced) and the picode-dash-scope localStorage cleanup — retired 2026-09-25 (feat/legacy-sweep-2)
- [x] Go tray migration (ADR-0142 amended): --tray/runRetiredTray, startup-repair --retarget-shell, RetargetTask/retargetTask/CanRetarget, TrayArgs, task.ps1 'retarget' and '--tray' — retired 2026-09-25 (feat/legacy-sweep-2)
- [x] recoverPiReceiverRuntime /proc environ fallback (hello without runtimePid recovers nothing) — retired 2026-09-25 (feat/legacy-sweep-2)
- [x] desktop-shell: disk_report --stream retry fallback; btab_cdp_call optional domains (now required Vec<String>) — retired 2026-09-25 (feat/legacy-sweep-2)
- [x] store.CreateTerminal (callers use CreateTerminalIn with FreeWorkspaceID) — retired 2026-09-25 (feat/legacy-sweep-2)
- [x] qa-cli-providers.mjs is stale since ADR-0169 (2026-09-21): waits for #providers-view .prov-bar; needs a rewrite for the one-pane Providers — fixed 2026-09-25 (feat/qa-scripts-fix)
- [x] qa-cli-settings.mjs fails the keyboard density assertion at line ~158 on picode-docs-fixture; its later sections (edited 2026-09-25) have not run — fixed 2026-09-25 (feat/qa-scripts-fix)
- [x] qa-cli-packages, qa-llama, qa-mobile-settings, qa-cli-settings-recovery not run after the 2026-09-25 link edits — fixed 2026-09-25 (feat/qa-scripts-fix)
- [ ] Owner: first build/run of the shell after btab.rs/disk.rs edits (make desktop-restart) and of startup-repair after the task.ps1 edit
- [ ] Owner: in the unified Packages pane a missing/mismatched agentId no longer blocks the machine-scope install up front — intended? — seen 2026-09-25 (feat/qa-scripts-fix)
- [ ] No browser QA for PackageTarget's moved-location guard (config pages) — seen 2026-09-25 (feat/qa-scripts-fix)
- [ ] No browser suite for the Providers pane account flows (pause/activate/OAuth) since ADR-0169 — seen 2026-09-25 (feat/qa-scripts-fix)

## Kept on purpose (not debts)

- Plaintext credentials vault import; M1 workspaces.json import.
- Non-partitioned webapps; third-party CLI file formats. (The pre-ADR-0161 shell path was retired 2026-09-25, feat/legacy-dead-code.)
- cli DEFAULT 'pi' and UI `cli || "pi"`: data rule (empty cli is Pi); server IsPi and others rely on it.
- notice.js toast shim.
- termGroups/workBack null guards (comments corrected 2026-09-25).
- terminalCli top-level cli: the server still copies it from hook state (term_state.go).
- Outcomes skills .catch: error tolerance, not legacy.
- Codex `agent-turn-complete`: Codex's own notify when its hooks are off (third-party).
- Empty-runID hook reports: hook processes without the wrapper env still exist.
- btab_preview (Rust): annotation captures.
- loadPackageReport: internal readers (roles, config pages).
