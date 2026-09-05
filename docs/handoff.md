# Handoff — living project state

> Heartbeat of PiCode. A session that changes state leaves this file matching
> HEAD. Historical detail lives in `docs/handoff-archive.md`.

## Current state (read this first)

**Repository:** bounded captures (ADR-0076), history reconciliation and mobile
composer wrapping are merged into main and locally deployed as `974780ba`.
The capture ADR was renumbered because Integrations took 0075. The opt-in native
emitter and real end-to-end acceptance remain pending: deployment does not
enable browser capture emission. Integrations and the current provider favicons
and tab/sidebar size fixes remain included. Desktop user menu Agent CLIs now
uses the terminal icon so it no longer matches Preferences (sliders).
HEAD also includes File Tree v2 (0074), worktree-aware Git Graph (0073), independent
web apps (0072), Windows task reliability (0071), Agent CLIs v2 and Docker v3.
Managed agents remain Pi-only; coding CLIs are terminals. No push was made.
Preserve the unrelated root `.pi/compact.json`.

**Last application deployment:** `0.1.0+bd9f28b`, health `ok`, boot
`ca34d21d4fc3495d`, desktop bundle `index-UlgaKnlT.js` (served bundle equals
the built one). `make deploy` restarted systemd; the existing terminal
records survived. This deploy carries `pi-diff` (ADR-0077) and refreshed UI
captures; no PiCode core behavior changed.

**Quality:** fmt/vet/Go tests/frontend tests/`make web` passed for the icon
swap. visual-review: PASS on Vite `:5174` and live `:8445` (`overlayAudit` ok).
Gmail `make ci` from the previous deploy still stands. Feature worktree and
branch are removed after merge.

### Product and platform

- One Go binary serves independent `/desktop/` and `/mobile/` apps. Mobile
  owns copied UI and lazy screens; shared contracts/tokens have explicit
  exports. HTTPS defaults to `:8445`.
- Integrations owns Connectors and Webhooks on both apps. Generic signed
  delivery lives in core; service-specific tools stay in external MCP packages
  or services. Native configuration import is reviewed, not implicit execution.
  Installed connector packages are distinct from configured/live services.
  The catalog ships a Gmail card backed by a community MCP server; its Google
  credentials stay outside PiCode in `~/.gmail-mcp/` (recipe in the public guide).
- Webhooks persist cursors, retry deadlines and revision-guarded acknowledgements.
  Only durable events are eligible; retention gaps are recorded. Secrets appear
  only on creation/rotation. No redirects/environment proxy; metadata addresses
  are refused. HTTP/LAN destinations require user trust, not a sandbox.
- Git Graph has one dirty row per worktree, read-only branch/HEAD-addressed
  sibling reads and manual refresh. Unused `git/head` token endpoints remain.
- File Tree keeps files and working diffs inline, protects drafts on replacement,
  close and refresh, and distinguishes Git-ignored entries without hiding them.
- Workspaces support managed Pi agents, free agents and tmux terminals. Private
  sessions follow ADRs 0039/0040/0053; Inbox uses the receiver extension with a
  paste fallback (0060). Store mutations append events transactionally (0048).
- Terminal presence/activity use wrapper leases and exact process fallback,
  never scraped pixels (0062). Agent CLIs v2 owns launch defaults, profiles,
  previews and setup checks; saving configuration never restarts a process.
- Docker v3 and `pi-sysadmin` share inventory, operations, health and reviewed
  maintenance. Compose registration remains proposed.
- Public docs have generated API/llms maps, screenshots and video integrity
  checks. Releases remain tag-driven; ADR-0064's cadence pilot is proposed.
- Windows task policy, diagnosis, scoped repair and normal Quit are locally
  installed. Native lifecycle/race and UAC repair passed; repair is repeatable
  without altering actions/principals/triggers. Device transitions are deferred.

### Compaction policy (`pi-compact`, ADR-0061)

Deployed and dormant without configuration. The root `.pi/compact.json` belongs
to other work and was not evaluated. Real-compaction acceptance must still prove
`fromHook: true`, gemini-3.6-flash pricing and no aborted turns.

### Diff panel (`pi-diff`, ADR-0077)

Deployed. `/diff` in the pi TUI opens a right-hand panel with the changed
files and the focused file's numbered hunks; the footer carries the total.
In pi's fullscreen TUI mode (`--tui-mode fullscreen` or `/settings`) the
panel is a real layout column: full height, fixed while the transcript
scrolls, chat and editor wrapping left, mouse wheel scrolls the hunks. In
regular mode it is a full-height overlay that scrolls with the terminal and
the first `/diff` says so once (owner refinement 2026-09-05, ADR amendment).
Dogfooded in tmux in both modes, including a real grok write turn that
refreshed and refocused the column. 20 logic tests. Listed in the root
`.pi/settings.json`, so project agents load it on their next start.

## In flight

- `pi-diff` (ADR-0077): whether PiCode should spawn terminal TUIs with
  `--tui-mode fullscreen` so the panel is always a fixed column is an owner
  decision not yet taken; in regular mode the panel scrolls with the terminal.

- OAuth-provider acceptance, model-driven connector usage and first-class
  non-Pi agents are not certified by the public/no-auth DeepWiki protocol check.
  Integrations itself is merged and deployed.
- Physical iOS/Android PWA upgrade and push need device acceptance. The browser
  migration, shared worker matrix and mobile review fixes passed locally.
- Windows next-logon, battery and sleep/resume acceptance is owner-deferred;
  do not interrupt the current Windows session for these tests.
- Compose registration/deployment remains a separate ADR-0065 extension;
  existing-project operations and Docker v3 are merged and deployed.
- Autonomous model-driven Sysadmin dogfood and Docker Desktop/rootless acceptance
  remain open. Real Linux Engine QA passed without a model turn.
- Configured real-compaction and model-driven CLI/Inbox dogfood remain pending.
  Reconcile historical Inbox `[Teste 3]` and `mobile-6bf740` before a live reply.
- ADR-0054 `picode-act` needs real model-emitted dogfood before merge; Browser
  preview needs the native opt-in emitter and real partial/final RPC acceptance
  before panel work; the bounded host increment is implemented.
- Second-account, container, public-OIDC and other remote-mode acceptance require
  owner-controlled infrastructure. Official release dates remain unset.

## Next up

1. Scope provider OAuth/marketplace acceptance only if the owner requests it;
   generic Integrations and external connector setup are already deployed.
2. Validate deployed PWA upgrades and push on iOS/Android. Mobile UI increments
   belong only in `web/mobile`.
3. Run the version-specific CLI working/approval/settled acceptance matrix.
   First-class CLI agents need a protocol/session/package parity ADR.
4. Design Compose registration from `docs/plans/docker-v2.md`: file ownership,
   dependency order, deployment preview and recovery need an ADR.
5. Review ADR-0064 and choose the cadence/pilot window.
6. Re-dogfood configured compaction; inspect historical Inbox rows before replies.
7. Run owner-controlled remote-mode acceptance. Continue Browser preview with
   opt-in native emission and real RPC/cancellation/slow-consumer acceptance,
   not the panel yet; ADR-0054 dogfood remains separate.
8. Decide whether selective docs-video recapture/render should be scheduled.

## Known debts / open questions

- **Capture integration: FAIL/deferred.** No real emitter-to-RPC run or measured
  slow-consumer/cancellation matrix. The hub drops on overflow and caches no
  missed partials. Desktop reconnect and same-agent session replacement during
  a pending session-changing API need dedicated acceptance; mobile reconnect
  and selection switching passed with fixtures. See `docs/plans/browser-preview.md`.

- Webhook delivery is at-least-once within event retention, not an unlimited
  archive. Receivers must handle duplicate IDs; arbitrary receiver response
  bodies are not retained. Public connector QA proves protocol, not agent use.
- Task Scheduler retries are not general crash recovery: an exit-one probe
  stayed stopped for 90 seconds. Runtime supervision needs a new approved design.
  Battery/sleep/sign-in remain owner-controlled; see the desktop reliability plan.
- Existing `TestTerminalBrowse` cleanup can leave tmux shells in deleted temporary
  folders. Review cleanup lifetime; preserve pre-existing unrelated shells.
- CLI lifecycle coverage is version-specific; executable/setup checks alone do
  not prove working/approval/settled behavior for a vendor.
- Docker leaves volume deletion, backup/restore, remote engines, historical
  charts and automatic repair for future decisions. Secret masking is best effort.
- Legacy terminal sessions get exact PID/command fallback; instrumented sessions
  have stronger presence. `/proc` start tokens give Linux stronger identity checks.
- Ephemeral feed events can be missed across reconnects (0048). Cross-platform
  paste-fallback acceptance remains open; receiver ack precedes Pi processing.
- Pi has one active credential slot; per-agent OAuth and proactive quota switching
  need an owner decision. The Shift+Enter shim and some legacy role-state/slash
  assumptions remain until stable replacements are measured.
- Tutorial integrity passes, but all three strict freshness audits remain stale
  after source relocation. Recapture/render is explicit; hashes were not relabeled.
- Branch protection and CODEOWNERS require owner action on GitHub.

## Recent activity

- **2026-09-05 — `pi-diff` fullscreen column (ADR-0077 amendment).** The
  owner saw the overlay scroll away mid-screen in a PiCode terminal tab. In
  fullscreen TUI mode the panel now wraps pi's layout root in an `HStack`
  (chat 55 / panel 45, full height, fixed, wheel scroll); regular mode keeps
  a full-height overlay plus a one-time hint. Widget slot re-set per refresh
  because a mode switch replaces the TUI instance. Dogfooded both modes.
- **2026-09-05 — User menu Agent CLIs icon.** Preferences kept sliders;
  Agent CLIs now uses the terminal glyph. visual-review: PASS (`overlayAudit`
  ok). Merged and deployed `2231919`.
- **2026-09-05 — `pi-diff` package built (ADR-0077).** New MIT package
  `packages/pi-diff` in the pi-checklist mold: pure `src/logic.ts` (numstat,
  porcelain and unified-diff parsing, focus rules, layout) with node:test
  coverage, and `extensions/diff.ts` (git, `tui.showOverlay` through a
  zero-height widget slot, `tool_execution_*`/`turn_end` refresh, `/diff`
  command with completions, `alt+n`/`alt+u`/`alt+pageUp`/`alt+pageDown`).
  `ctx.ui.custom()` was rejected because its `ui_prompt_*` events would read
  as "waiting for user" to the guest-TUI sensors. Guide at
  `www/guide/diff-panel.md`. Merged `32b9c24f` (ADR renumbered 0076→0077 on
  merge), UI captures refreshed after `docs-check`, full `make ci` green,
  deployed `bd9f28b`; worktree and branch removed.

- **2026-09-05 — Gmail connector merged and deployed.** Reconciled the capture
  host work and passed combined `make ci`; deployed `9284cd1`. Live `/api/mcp`
  serves the `gmail` preset and the card renders with `context7` untouched;
  32/32 panes survived. visual-review: PASS. No integration mutations or push.

- **2026-09-05 — Gmail connector across all three add paths.** Catalog gains a
  Gmail card (community `@gongrzhe/server-gmail-autoauth-mcp`, credentials stay
  in `~/.gmail-mcp/`), `connectors/gmail.json` is importable, and optional
  `packages/pi-connector-gmail` documents install/sign-in/revocation. Preset
  completeness test added; guide recipe and changelog updated. Isolated-daemon
  browser check: one catalog click created the `gmail` server entry with the
  exact command; desktop/mobile screenshots read, audits ok. visual-review:
  PASS (`integrations-catalog-before/gmail-added.png`). `make ci` passed.

Older activity lives in `docs/handoff-archive.md`.
