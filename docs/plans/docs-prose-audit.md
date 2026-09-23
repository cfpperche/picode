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
| 1 | routes, agent-manager, terminal-bridge, security-model | — | — | — | `feat/docs-prose-audit` |

## Wave 1 — hot subsystems

- [ ] routes.md
- [ ] agent-manager.md
- [ ] terminal-bridge.md
- [ ] security-model.md

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
