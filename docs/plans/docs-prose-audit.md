# Docs prose audit — docs/architecture/ × code (wave plan)

> Tracker for the prose half of the docs×code adversarial audit (2026-09-23).
> The mechanical half (routes, ADR index, OpenAPI, CLI subcommands, versions)
> shipped in `feat/docs-code-audit`. This plan covers the claim-by-claim
> reading the scripts cannot do. One wave = one branch = one `make close`.

## Method

Per file: extract claims with line refs → verify each in code → verdict
OK / DRIFT / UNVERIFIABLE. DRIFT fixes the docs (code is what ships) unless
the prose records an accepted ADR the code violates — then debt + owner, no
doc rewrite. UI-behavior claims probe the scratch instance. Unverifiable →
`docs/handoff/open/docs-prose-audit.md`.

## Claim taxonomy

constants · defaults · UI labels · behavior absolutes ("never/always/only") ·
flows/states · cross-references (symbols, files, ADRs).

## Verdicts so far

| Wave | Files | OK | DRIFT | UNVERIFIABLE | Branch |
|---|---|---|---|---|---|
| 1 | routes, agent-manager, terminal-bridge, security-model | ~95 | 7 | 0 | landed |
| 2 | cli-terminal-launch, cli-providers, cli-settings, cli-memory, cli-session-handoff, packages, mcp, integrations | ~70 | 5 | 0 | `feat/prose-audit-w2` |

## Wave 2 — Agent CLIs stack

- [x] cli-terminal-launch.md — 5 DRIFT fixed: two `POST /api/clis/<cli>/terminals` mentions (route removed by ADR-0184's one door; today the handoff door and `POST /api/agents`); "npm-backed for pi/codex/claude-code" now names omp and opencode too (`ForMissing`); Muse/Antigravity no longer described as `surface: terminal` (full rows since launch-parity; `SurfaceTerminal` has no catalog row); pane-tab paragraph now lists Memory (six CLIs, ADR-0163) and Models (omp only, ADR-0181)
- [x] cli-providers.md — clean: usage routes, plan windows, `resets[]`, custom providers ADR-0129/0175, unified pane ADR-0165/0169
- [x] cli-settings.md — clean: `cycleOrder`/`modelTags` (clisettings/roles.go), `GET /api/cli-models`, `modellist.Probe`; the `docs/keybindings.md`/`docs/settings.md` mentions are pi's and omp's own upstream files, not repo paths
- [x] cli-memory.md — clean: six-of-nine keepers matches the spec table (editable 4 + readonly 2; Pi/OpenCode none; Antigravity unknown)
- [x] cli-session-handoff.md — clean: handoff routes, transcript/clisession capabilities
- [x] packages.md — clean: verb route family registered, gallery/config/describe live, `guest_view.go` correctly described as removed
- [x] mcp.md — clean: mcp.json targets, seed cards pinned, gallery ADR-0157
- [x] integrations.md — clean: `/api/webhooks` CRUD registered

## Wave 1 — hot subsystems

- [x] routes.md — 5 DRIFT fixed: `web/desktop/src/lib/{fileDocument,useFocusMode,nativeApps}.js` and `web/desktop/src/styles/app.css` → the browser bundle (`web/browser/…`; web/desktop is only the shell bootstrap); user-menu groups were stale (Tools now holds Automations/Snippets/llama.cpp; Integrations sits under PiCode with Browser, Computer, Termset)
- [x] agent-manager.md — 2 DRIFT fixed: workspace menu no longer lists Sessions (removed by `feat/remove-ws-sessions`); the per-CLI session index covers all nine catalogued CLIs, not six
- [x] terminal-bridge.md — clean: backoff 1s→10s/12 bursts, RespawnPaneEnv/PasteText/PaneCommand/PaneSessionID, bornGrace 5s, tmux-guard table (`tmux-guard.log`, `show-environment` probe, `# PiCode intercept` marker), ADR-0138/0139/0180/0112 all verified in code
- [x] security-model.md — clean: guarded() = `/api/`,`/ws/`,`/mcp/communication`; PairingTTL 10min, 5 fails/min + 10min lockout, daily prune after a week; preview ticket DefaultTTL 1h; webapps fetch 4s/256KB/64KB; appCSP `ipc.localhost` desktop-only; `external.rs` allowlist as described

## Wave 3 — native surfaces & apps

- [x] canvas.md — 10 DRIFT fixed, all one class: `web/desktop/src/{components,hooks,lib}/…` paths → the browser bundle (GrantedContacts, CanvasSurface, PatternSwatch, useAgentSocket, fileDocs, nativeApps, routes). Numbers verified (panel min 256×224 = 32×28 units)
- [x] work-browser.md — clean: all cited rs/js files exist, ADRs 0132/0134/0135/0144/0146/0172 exist
- [x] computer-tool.md — clean: closed catalog of 23 confirmed by counting `internal/computer/actions.go`; 1280/480 caps in computer.rs; five commands in main.rs; refusal names in capture.rs
- [x] chrome-extension.md — clean: `/api/extension/*` routes, `doorDeliverUnattended`, native messaging
- [x] file-preview.md — clean: TokenBytes 32, 26-char base32 label, 1 MiB overlay cap (`maxAgentText`), kinds registry, previewHostHandler
- [x] docker-app.md — clean: API 1.44, PICODE_DOCKER_HOST/DOCKER_CONTEXT order, 45s job bound, `docker_operations`, `docker.changed` ephemeral (cmd/picode/main.go)
- [x] docker-maintenance.md — clean: monitor Validate() = intervals 30/60/300, retention 7/30, 32-project cap, 4 concurrent, 128 containers
- [x] tmux-app.md — clean: `StartTmuxServerWatch` at 15s
## Wave 4 — stable

- [ ] automations.md · broker.md · change-feed.md · climetrics.md ·
      compaction-policy.md · credentials.md · data-persistence.md ·
      delivery.md · devservers.md · direct-session-communication.md ·
      gateway.md · llama-manager.md · managed-principals.md · model-roles.md ·
      notices.md · picode-mcp.md · pins.md · rpc-bridge.md · session-reader.md ·
      snippets.md · tui-diff-panel.md · cli-providers.md leftovers

(Per-file verdict lines are appended under each wave as waves land.)
