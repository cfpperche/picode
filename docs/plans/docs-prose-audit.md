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
| 1 | routes, agent-manager, terminal-bridge, security-model | ~95 | 7 | 0 | `feat/docs-prose-audit` |

## Wave 1 — hot subsystems

- [x] routes.md — 5 DRIFT fixed: `web/desktop/src/lib/{fileDocument,useFocusMode,nativeApps}.js` and `web/desktop/src/styles/app.css` → the browser bundle (`web/browser/…`; web/desktop is only the shell bootstrap); user-menu groups were stale (Tools now holds Automations/Snippets/llama.cpp; Integrations sits under PiCode with Browser, Computer, Termset)
- [x] agent-manager.md — 2 DRIFT fixed: workspace menu no longer lists Sessions (removed by `feat/remove-ws-sessions`); the per-CLI session index covers all nine catalogued CLIs, not six
- [x] terminal-bridge.md — clean: backoff 1s→10s/12 bursts, RespawnPaneEnv/PasteText/PaneCommand/PaneSessionID, bornGrace 5s, tmux-guard table (`tmux-guard.log`, `show-environment` probe, `# PiCode intercept` marker), ADR-0138/0139/0180/0112 all verified in code
- [x] security-model.md — clean: guarded() = `/api/`,`/ws/`,`/mcp/communication`; PairingTTL 10min, 5 fails/min + 10min lockout, daily prune after a week; preview ticket DefaultTTL 1h; webapps fetch 4s/256KB/64KB; appCSP `ipc.localhost` desktop-only; `external.rs` allowlist as described

## Wave 2 — Agent CLIs stack

- [ ] cli-terminal-launch.md
- [ ] cli-providers.md
- [ ] cli-session-handoff.md
- [ ] cli-settings.md
- [ ] cli-memory.md
- [ ] packages.md
- [ ] mcp.md
- [ ] integrations.md

## Wave 3 — native surfaces & apps

- [ ] work-browser.md
- [ ] computer-tool.md
- [ ] chrome-extension.md
- [ ] canvas.md
- [ ] docker-app.md
- [ ] docker-maintenance.md
- [ ] tmux-app.md
- [ ] file-preview.md

## Wave 4 — stable

- [ ] automations.md · broker.md · change-feed.md · climetrics.md ·
      compaction-policy.md · credentials.md · data-persistence.md ·
      delivery.md · devservers.md · direct-session-communication.md ·
      gateway.md · llama-manager.md · managed-principals.md · model-roles.md ·
      notices.md · picode-mcp.md · pins.md · rpc-bridge.md · session-reader.md ·
      snippets.md · tui-diff-panel.md · cli-providers.md leftovers

(Per-file verdict lines are appended under each wave as waves land.)
