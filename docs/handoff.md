# Handoff — living project state

> Heartbeat of PiCode. A session that changes state leaves this file matching
> HEAD. Newest activity comes first; historical detail lives in
> `docs/handoff-archive.md`.

## Current state (read this first)

**Repository:** local `main` includes Agent CLIs v2 (`c3265377`, ADR-0070),
v1 and Docker v3. Managed agents remain Pi-only; coding CLIs are terminals,
not agent runtimes. Nothing was pushed. Preserve the unrelated root
`.pi/compact.json`. Compose registration/deployment remains proposed.

**Last application deployment:** `0.1.0+7964d4d`, health `ok`, boot
`70c3e75648358c41`; served HTML references the freshly built `index-_7SLKl4D.js`
bundle. Ships the Rename menu icon on top of the row alignment fix. All ten
sidebar rows kept their menu triggers after the restart.

**Local Windows tray (2026-09-05):** restarted through the existing enabled
`PiCodeDesktop` logon task; one tray process observed. Server boot
`97d7ca348df4a75a` and all six observed tmux pane IDs/PIDs stayed unchanged.
Task settings still need administrator approval; no installer code changed.

**Quality:** `make ci` passed (Go, 495 frontend tests, packages, build, docs
parity and Vale), plus focused CLI/Store/runtime race tests. Real tmux tests cover
argv parity, inheritance/pins, retry, preflight PID preservation and cleanup.
Browser QA covers profiles, workspace/palette context, reset, pending/restart,
dirty navigation and preview/network recovery. Empty/blocked, desktop/mobile
and light/dark screenshots were read; overlay audits passed (`docs/screenshots/cli-v2-*.png`).
Public captures were refreshed with isolated fixtures. No model turns were used;
v2 does not certify every vendor lifecycle event.

### Product and platform

- One Go binary serves the React/Vite desktop and mobile ADE. HTTPS defaults
  to `:8445`; hashed assets may cache, while HTML and APIs do not.
- Workspaces contain multiple agents; free agents and first-class tmux
  terminals are supported. Agent sessions are privately scoped and recorded
  in `agent_sessions` (ADRs 0039/0040/0053).
- Agents have interactive Pi TUI and managed Pi RPC run modes. Inbox replies
  use the injected receiver extension with a tmux paste fallback (ADR-0060).
- Desktop/mobile consume the ADR-0048 change feed. Store mutations append their
  events in the same transaction; ephemeral terminal runtime/state signals are
  deliberately in memory and reconcile through the terminal list.
- Terminal CLI presence and activity remain separate. Wrappers identify
  Claude Code, Codex, Grok, or Pi with a run id; exact tmux command/PID data is
  only a weaker legacy presence fallback. Pixels are never scraped and a guest
  CLI is never promoted to an Agent. CLI favicons fill the same 22px face slot
  as agent rows, with no chip; the compact text mark is only an asset-load
  fallback.
- Agent CLIs is available from the desktop user menu/palette and mobile More.
  It owns visible launch defaults, copied profiles, per-terminal overrides,
  shared previews, persistent setup checks, explicit repair and lifecycle
  actions. The old Terminal status preference redirects to `#/clis`. Saving
  settings never restarts a process; a stopped configured terminal needs Start.
- Apps now provide literal output, progress, empty lists and generic mobile
  navigation. Docker exposes inventory, resources/consumers, health/incident sampling,
  reviewed project operations and maintenance procedures with durable history. Inventory groups exact Compose
  projects inside the app, with saved folds and search (ADR-0066). Optional
  `pi-sysadmin` tools share the same service, existing authentication and
  local Unix socket access.
- Public docs use VitePress, generated OpenAPI, Vale, committed screenshots,
  and integrity-checked tutorial videos.
- Public release mechanics are tag-driven. The cadence study, proposed
  ADR-0064 and maintainer checklist are documented, but no calendar-triggered
  release or Preview lane is active.

### ADR-0061 compaction policy package (`pi-compact`)

Deployed and dormant until configured: no defaults without `.pi/compact.json`
(or a per-agent overlay). The untracked root config belongs to other work and
was not evaluated here. Commands are `/compact-edit|model|on|off`; bare
`/compact` remains Pi's native command. Trigger on `agent_settled` plus idle,
never `turn_end`, which aborts active work. Auto summarizer fallback remains
`gemini-3.6-flash` → `claude-haiku-4-5` → Pi; 54 package tests pass. A configured
real-compaction run must still prove `fromHook: true` and no aborted turns.

## In flight

- Local tray policy repair: `Set-ScheduledTask` returned `0x80070005`
  (access denied). Removing `PT72H` and both battery restrictions awaits
  owner-approved elevation; those settings remain unchanged.
- Compose file registration/deployment remains a separate proposal extending
  ADR-0065; existing-project operations and Docker v3 are merged and deployed.
- Autonomous model-driven Sysadmin dogfood and Docker Desktop/rootless
  acceptance remain open; real Linux Engine operation QA passed. No model
  turn was used to validate the 11-tool extension.
- **ADR-0061's amendment is deployed in the current service.** A configured
  real-compaction re-dogfood remains pending; without a config it stays dormant
  as designed, and the run must record `fromHook: true` plus gemini-3.6-flash
  pricing with no aborted turns.
- Model-driven CLI/Inbox dogfood remains pending; native CLI launch acceptance
  passed without model turns. The historical Inbox `[Teste 3]` and
  `mobile-6bf740` rows still need reconciliation before a new live question.
- ADR-0054 `picode-act` still needs real model-emitted dogfood before merge;
  the Browser preview emitter/panel remains open.
- Second-account, container, public-OIDC, and other remote-mode acceptance
  runs require owner-controlled infrastructure.
- ADR-0064 is proposed: the owner still needs to choose whether to accept the
  three-release, two-week pilot. Official release dates remain unset.

## Next up

1. Run the version-specific CLI working/approval/settled acceptance matrix.
   Any first-class CLI agent proposal needs a separate ADR covering
   protocol/session/package parity.
2. Design the separate Compose registration/deployment increment from
   `docs/plans/docker-v2.md`, with an ADR for file ownership, dependency order,
   deployment preview and recovery. Existing-project operations are implemented.
3. Review current local `main` and decide when to push/promote it.
4. Review ADR-0064 and choose the official cadence/pilot window; no release
   date is committed yet.
5. Verify the local `pi-compact` configuration and re-dogfood its policy.
6. Inspect the exact historical Inbox rows before any real TUI reply test.
7. Run the owner-controlled remote-mode acceptance matrix.
8. Continue the Browser preview panel and ADR-0054 dogfood.
9. Decide whether selective docs-video capture/render should be scheduled;
   current explicit capture and integrity gates already pass.

## Known debts / open questions

- Desktop installer task creation still inherits Windows duration/battery
  defaults. An installer fix and next-logon acceptance are separate work.
- `TestTerminalBrowse` left one disposable tmux session after passing CI;
  identity-checked cleanup removed it. Review test cleanup context/lifetime.
- CLI lifecycle coverage remains version-specific. Run the explicit
  working/approval/settled acceptance matrix before claiming full coverage
  for a vendor; setup checks only prove executable response and prerequisites.
- Docker v3 deliberately leaves volume deletion, backup/restore, remote engines,
  historical metrics charts and automatic repair policies for future decisions.
  Secret masking is best effort; arbitrary unlabeled secrets are not detectable.
- Wrapper presence is strongest for instrumented sessions. Legacy sessions
  get only exact command/PID fallback; Linux process identity has stronger
  `/proc` protection than platforms without start tokens.
- ADR-0048 ephemeral events can be missed across reconnects until a later
  state change or explicit reconciliation.
- Cross-platform acceptance of the ADR-0060 paste fallback has not been
  live-proved. A receiver ack also leaves a small window before Pi processes
  the queued reply.
- Pi still exposes one active credential slot; per-agent OAuth isolation and
  proactive quota switching need an owner decision and measurement.
- The terminal Shift+Enter shim remains until a stable xterm/tmux protocol
  combination replaces it. Some non-terminal screens still carry legacy
  `role-state`/`slash` assumptions.
- Tutorial CI checks committed integrity without rendering every changed
  tutorial; `make docs-videos-fresh` remains the explicit drift audit.
- Branch protection and CODEOWNERS still require owner action on GitHub.

## Recent activity

- **2026-09-05 — Rename menu icon.** `IconPencil` (existing) now leads the
  Rename entry in agent and terminal row menus; previously the only entry
  without one. visual-review: PASS (term-menu / agent-menu captures at
  1280×900 on the deployed service; overlay audit ok; card 5/5). First
  `make ci` failed only at `docs-check` (fleet/mobile capture inputs changed);
  `make docs-shots` regenerated three `www/img` captures and `ci-docs`+`vale`
  then passed; every earlier stage had passed. Process deviation: committed
  on the primary `main` checkout and deployed before this docs commit; no
  worktree was used. The public site captures are not part of the binary.

- **2026-09-05 — Workspace row first-line alignment fix.** Agent and terminal
  rows centered the status chip and overflow button against the two-line
  title block, so the right column rendered on the subtitle line; `.ws-row-*`
  now top-align with the name (title center delta 0.6px, measured). Visual
  review read agent, terminal and menu-overlay captures at desktop and 550px
  widths; overlay audit ok. Deployed after merge.

Older activity and retired implementation detail are in
`docs/handoff-archive.md`.
