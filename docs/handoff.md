# Handoff — living project state

> Heartbeat of PiCode. A session that changes state leaves this file matching
> HEAD. Newest activity comes first; historical detail lives in
> `docs/handoff-archive.md`.

## Current state (read this first)

**Repository:** HEAD includes independent desktop/mobile web applications
(ADR-0072), desktop task reliability (ADR-0071), Agent CLIs v2 and Docker v3.
Managed agents remain Pi-only; coding CLIs are terminals. This decoupling
increment is local-only: no push or application deployment. Read Git for
current upstream status. Preserve the unrelated root `.pi/compact.json`.
Compose registration/deployment remains proposed.

**Last application deployment:** `0.1.0+7964d4d`, health `ok`, boot
`70c3e75648358c41`; served HTML references the freshly built `index-_7SLKl4D.js`
bundle. Ships the Rename menu icon on top of the row alignment fix. All ten
sidebar rows kept their menu triggers after the restart. This concurrent
deployment restarted the service at 14:52 UTC; the tray increment did not.

**Windows desktop (ADR-0071):** explicit resident-task installer policy,
read-only `startup-check`, scoped/backup-first `startup-repair`, and successful
normal Quit are implemented. Native Windows launch-retry/duplicate/Quit and
policy-preservation tests passed, plus focused race tests and `make ci`.
The installed tray was replaced through `make desktop-restart`; normal UAC
repair removed `PT72H` and battery/idle/network gates. A final task launch at
11:55:13 local time is running with one instance; all 18 baseline pane IDs/PIDs
survived. Action/principal/triggers were preserved; repeat repair is a no-op.

**Quality:** decoupling passed `make ci` (603 frontend tests, Go/packages,
build, docs and Vale), embedded UI/server tests, independent-build artifact
checks, and synthetic-browser review. Empty/blocked/error, light/dark,
narrow desktop and wide mobile sheets were read; overlay audits passed
(`docs/screenshots/split-*.png`). Legacy links, worker upgrade, chunk retry,
root preservation on resize and both Vite dev entries were checked. No model
turns were started. Physical PWA/push acceptance is still open.

### Product and platform

- One Go binary serves independent `/desktop/` and `/mobile/` apps. Desktop
  stays responsive; mobile owns copied UI and lazy screens. Shared contracts
  and tokens have explicit exports; HTTPS defaults to `:8445`.
- Workspaces support multiple agents, free agents and tmux terminals. Private
  agent sessions follow ADRs 0039/0040/0053. Pi has interactive TUI and managed
  RPC modes; Inbox uses the receiver extension with a paste fallback (ADR-0060).
- Store mutations append feed events transactionally. Terminal presence and
  activity are distinct, with wrapper leases and exact process fallback;
  pixels are never scraped (ADRs 0048/0062).
- Agent CLIs v2 owns launch defaults, copied profiles, previews, setup checks
  and lifecycle actions. Saving configuration never restarts a process.
- Docker v3 and `pi-sysadmin` share resource inventory, project operations,
  health sampling and reviewed maintenance. Compose registration stays proposed.
- Public docs have generated API/llms maps, screenshots and video integrity
  checks. Releases remain tag-driven; ADR-0064's cadence pilot is proposed.

### Compaction policy (`pi-compact`, ADR-0061)

Deployed and dormant without configuration. The root `.pi/compact.json`
belongs to other work and was not evaluated. A configured real-compaction
run must still prove `fromHook: true` and no aborted turns. Prior policy
and fallback detail is archived; the implementation is unchanged here.

## In flight

- Physical iOS/Android PWA upgrade and notification delivery need device
  acceptance. The browser migration and shared worker decision table passed.

- Windows next-logon, battery and sleep/resume acceptance is explicitly
  deferred by the owner. Installer and local task repair are complete;
  do not interrupt the current Windows session for these tests.
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

1. Validate the existing PWA on iOS/Android, then promote the decoupled build.
   Further mobile UI increments belong only in `web/mobile`.

2. Run the version-specific CLI working/approval/settled acceptance matrix.
   Any first-class CLI agent proposal needs a separate ADR covering
   protocol/session/package parity.
3. Design the separate Compose registration/deployment increment from
   `docs/plans/docker-v2.md`, with an ADR for file ownership, dependency order,
   deployment preview and recovery. Existing-project operations are implemented.
4. Review ADR-0064 and choose the official cadence/pilot window; no release
   date is committed yet.
5. Verify the local `pi-compact` configuration and re-dogfood its policy.
6. Inspect the exact historical Inbox rows before any real TUI reply test.
7. Run the owner-controlled remote-mode acceptance matrix.
8. Continue the Browser preview panel and ADR-0054 dogfood.
9. Decide whether selective docs-video capture/render should be scheduled;
   current explicit capture and integrity gates already pass.

## Known debts / open questions

- Task Scheduler launch retries are not general crash recovery: a native
  exit-one probe stayed stopped for 90 seconds. Separate tray/WSL keepalive
  lifetime and runtime supervision need a new owner-approved design.
- Windows native lifecycle acceptance is opt-in in ordinary CI; it was run
  and passed here. Battery/sleep/sign-in transitions remain owner-controlled;
  see `docs/plans/desktop-task-reliability.md` for the decision table/evidence.
- Existing `TestTerminalBrowse` cleanup can leave tmux shells with deleted
  temporary folders. Seven were observed without a worktree ownership marker;
  retained them and recorded the cleanup context/lifetime debt.
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
- Tutorial integrity passes. The strict freshness audit reports all three
  tutorials stale after source relocation; recapture/render remains explicit
  maintenance. Their existing hashes were not relabeled as fresh.
- Branch protection and CODEOWNERS still require owner action on GitHub.

## Recent activity

- **2026-09-05 — Independent web applications (ADR-0072).** Own npm/Vite
  entries, copied mobile UI, shared contracts/tokens, explicit app paths and
  stable PWA identity. Initial mobile JS: 562 kB vs 2,557 kB at `522844a2`;
  optional previews remain lazy. Incorporated concurrent Rename/task work
  and the Windows/macOS test portability fixes.
  visual-review: PASS; public captures refreshed. No application deployment.

Older activity and retired implementation detail are in
`docs/handoff-archive.md`.
