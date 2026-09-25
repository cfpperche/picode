# Dead code — what the 2026-09-25 sweep left for the owner

`feat/dead-code-sweep` removed what nothing in the repo reaches (never-imported
JS files and exports, unused icons, Go functions dead even from tests, CSS
classes no source names). What follows has a caller that may live outside the
repo, a test that would go with it, a name built at runtime, or a documented
decision behind it — each is the owner's call. Method: grep + `deadcode -test`
+ staticcheck, 2026-09-25 on main 8cef9f00c.

## Debts

- [ ] Routes — POST /api/providers/{id}/usage/reset and /api/providers/{id}/accounts/{aid}/usage/reset: no caller, no test (slash_ops.go). They are public API in openapi.json.
- [ ] Routes — GET/POST /api/workspaces/{id}/principals and DELETE /api/managed-clis/{id} (alias of agent delete): only principals_test.go calls them; an external CLI might.
- [ ] Routes — DELETE /api/browser/annotations/{id} (WebTab only POSTs) and POST /api/browser/permissions/clear (only its test; the UI deletes per row or prunes).
- [ ] Routes — POST /api/agent-exits/{id}/undo: only its test; ADR-0211 made Undo = restore.
- [ ] Routes — POST /api/auth/logout (only spec_test.go; no UI sign-out calls it), POST /api/agents/{id}/login and /api/agents/{id}/command (no caller found).
- [x] Routes — packages_describe.go registerPackageDescribeRoutes (never called; duplicated the live /api/packages/describe registration in packages_config.go) — removed 2026-09-25 (feat/dead-code-sweep)
- [ ] Tauri — 7 commands registered but never invoked from webview JS: clean_list, disk_compact_dry_run, computer_displays, computer_preview, btab_zoom (Rust calls it directly; only the registration is dead), lab_open, computerlab_open. Removing needs a shell build.
- [ ] Scripts — referenced by nothing, may be run by hand: install-systemd.sh (superseded by `picode install`), verify-browser-capture-sidecar.mjs, qa-peer-communication.mjs, qa-agent-menu.mjs, qa-agent-tui.mjs, qa-mobile-v2.mjs.
- [ ] Go reachable only from tests — session/stats StatsRoot + statsAcc; tmux New/IsolatedEnv/KillIsolatedServer/MarkIsolated; the cli* packages' Supported/Read/List/Panes/Installed; clicreds.ProvidersFor/CanUse/EnvVar; ~60 in all. Removing each means deleting its tests.
- [ ] JS — 35 exports used only by their own tests (e.g. agentEvents markSent/markUndelivered/markAborted, routes sessionsRoute/providersLlama); CSS — 141 classes marked uncertain because a runtime-built name (`is-${x}`, `cli-${x}`) could produce them.
- [ ] ADR-0121 frame mode: no shell sets data-picode-frame today, so the :root[data-picode-frame] rules never apply — retire the convention (amend the ADR) or restore the setter.
- [ ] Flake: TestResolveInstalledCLIRetriesThroughAnUpdateWindow failed once under `make ci-scoped` load (2026-09-25); passed 3/3 alone and on rerun.
