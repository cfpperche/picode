# Handoff — living project state

> Heartbeat of PiCode. A session that changes state leaves this file matching
> HEAD. Historical detail lives in `docs/handoff-archive.md`.

## Current state (read this first)

**Repository:** HEAD (deployed as `0.1.0+07cc806`) carries today's
worktree-scoped asset-preview fix (ADR-0073 amendment), the Integrations
rollout (ADR-0075) with connector catalog tabs and the Gmail recipe, the
desktop Inspector rail (ADR-0078, proposed until the owner accepts), bounded
captures (ADR-0076), the pi-diff TUI panel (ADR-0077 with the fullscreen
amendment) and the managed-stop process-group fix, plus File Tree v2 (0074),
worktree-aware Git Graph (0073), independent web apps (0072), Windows task
reliability (0071), Agent CLIs v2 and Docker v3. Managed agents remain
Pi-only; coding CLIs are terminals. Sessions moved under Agent CLIs
(ADR-0079, this branch): `#/clis/sessions(/<wsId>)` replaced `#/sessions*`.
No push was made. Preserve the
unrelated root `.pi/compact.json`. The capture ADR was renumbered because
Integrations took 0075; the opt-in native emitter and real end-to-end
capture acceptance remain pending, so deployment does not enable browser
capture emission.

Managed-stop fix debts (living):

- `Runtime.Stop` has an unrelated start-lease race: a stop arriving while a
  managed start is in flight waits for the start, then returns true without
  stopping the just-started agent. Not exercised by tests; unfixed.
- Windows `Close` kills only the direct child (no posix process groups;
  Job Objects would be the faithful equivalent). The managed server does
  not run on Windows today.

HEAD also includes File Tree v2 (0074), worktree-aware Git Graph (0073), independent
web apps (0072), Windows task reliability (0071), Agent CLIs v2 and Docker v3.
Managed agents remain Pi-only; coding CLIs are terminals. No push was made.
Preserve the unrelated root `.pi/compact.json`.

**Last application deployment:** `0.1.0+07cc806`, health `ok`. `make
deploy` restarted systemd; the terminal records survived. Live verification
of the worktree-blob fix: the previously-404 request answers `200 image/png`
and the uncommitted panel renders the sibling's PNG previews (screenshots
read). Earlier deploy `0.1.0+075f6cc` had carried the connector-tabs
rollout with health `ok`, boot `6020b8eed79d6e22`, 44/44 panes survived.
Live Integrations: the catalog tabs render and the
**Claude Code** tab shows the real `context7` server as "Added from Claude
Code" while the existing service row stayed untouched — read-only checks, no
production mutations. Read evidence: `var/connector-tabs-deploy/`; private
SQLite and previous-binary backups are in its `recovery/` folder.
**Quality:** `make ci` passed three times — on the feature tree, after the
first main merge (with regenerated captures) and on the final merged tree —
including Go tests, 700 frontend/package tests, both UI builds, the embedded
binary, docs parity (`app-inspector` added) and Vale. Browser acceptance:
`scripts/qa-inspector.mjs` 11/11 groups on an isolated fixture
(`docs/screenshots/inspector-qa.json`), 11 screenshots plus the live shell
read, overlay/row audits ok. visual-review: PASS. The docs fixture and QA
browser sessions are closed.

### Product and platform

- The desktop Inspector rail (ADR-0078) follows the selected tab's owner
  (agent, terminal, workspace; apps keep the last anchor) and shows **Changes**
  — a folder-grouped working tree with `+N −M` per file and folder, an
  `Uncommitted` total, branch and worktree, and an `All | This agent` scope
  beside an agent — and **Files**, the lazy project tree with a filter over
  loaded rows. It opens files and diffs as center tabs (the file tab gained a
  Diff view), pins the owner's root and turns a background 409 into "This
  terminal moved to … Follow". Width/open/tab are per-viewer localStorage;
  open by default at ≥1440px; shrinks before it hides and never leaves the
  conversation under 640px. `gitstatus` now carries `add`/`del`/`binary`,
  `totals`, `branch` and `worktree`.
- One Go binary serves independent `/desktop/` and `/mobile/` apps. Mobile
  owns copied UI and lazy screens; shared contracts/tokens have explicit
  exports. HTTPS defaults to `:8445`.
- Integrations owns Connectors and Webhooks on both apps. Generic signed
  delivery lives in core; service-specific tools stay in external MCP packages
  or services. Native configuration import is reviewed, not implicit execution.
  Installed connector packages are distinct from configured/live services.
  The catalog ships a Gmail card backed by a community MCP server; its Google
  credentials stay outside PiCode in `~/.gmail-mcp/` (recipe in the public guide).
  The Add-connector card has catalog tabs: a fixed Catalog tab (Custom card
  opens the server form) plus one tab per agent CLI with found MCP servers;
  host imports use a reviewed confirmation with an Added state. The Use-from
  dialog and inline form are retired.
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

- `refactor/sessions-into-clis` (ADR-0079, accepted): the desktop sessions
  surface moved under Agent CLIs — `#/clis/sessions` (machine-wide) and
  `#/clis/sessions/<workspaceId>` (one folder) render as the third tab;
  `#/sessions*` deep links redirect; sidebar icon, dashboard top sessions
  and the user menu emit the new hashes. UI-only: no API change. Awaits
  merge to main. Phase 2 (other CLIs' sessions) is next, see below.
- `fix/worktree-blob-preview` awaits merge to main and `make deploy`; the
  running instance predates the fix (see Current state). Mobile keeps no
  worktree concept in its Changes screen, so nothing to ship there.
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

1. **Sessions phase 2 (ADR-0079)**: per-CLI session sources —
   `internal/clisession` with a `Source` per CLI (pi wraps the existing
   package; claude-code reads `~/.claude/projects`; codex reads
   `~/.codex/sessions`; grok needs verified evidence first),
   `GET /api/clis/{id}/sessions(+transcript)`, per-CLI filter and search on
   `#/clis/sessions`, and "Open in terminal" resume actions for non-Pi
   sessions. Chat replay stays Pi-only. Decide then whether
   `/api/sessions/all` and `/api/pi-sessions` become deprecated aliases.
2. Scope provider OAuth/marketplace acceptance only if the owner requests it;
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
9. After a few days of Inspector dogfood, decide its later phases (ADR-0078
   designs both, neither approved): a read-only **PR** tab through the host's
   `gh`, and **Commit / Commit & Push** — pre-typed into the owner's terminal,
   or server-side behind an interlock. Accept or amend ADR-0078 then.

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
- Inspector debts (ADR-0078): no complete filename search yet — the Files
  filter covers loaded rows only (`/files?q=` is agent-only and stops at 200
  files; `git ls-files` for all three owner kinds is the planned fix); free
  terminals outside a watched folder refresh on focus/visibility and their row's
  live facts, not on a watcher (a per-anchor watch lease is the fallback);
  the This-agent scope chips show only while the agent's own tab is selected;
  the sizer idiom is still copied in Sidebar and FileTreeSurface; the per-turn
  `+N −M` footer beside the conversation is not drawn. ADR-0078 may be
  renumbered at merge — `feat/pi-diff` also holds an unmerged 0076.

## Recent activity

- **2026-09-05 — Sessions moved under Agent CLIs (ADR-0079).** Desktop
  `#/clis/sessions(/<wsId>)` replaces the top-level `#/sessions*` routes as
  the third tab of the Agent CLIs surface; sidebar icon, dashboard top
  sessions and user menu emit the new hashes and old deep links redirect.
  UI-only phase 1: no API change. `make ci` green; docs captures
  regenerated. visual-review: PASS (empty, loaded, workspace-scoped, error
  and Open-with overlay states read; overlayAudit ok).
- **2026-09-05 — Worktree-scoped asset previews fixed and deployed
  (ADR-0073 amendment).** Terminal `/blob` ignored `?worktree=` and
  workspace git reads skipped it, so uncommitted panels showed "Can't load
  this image." for a sibling worktree's untracked screenshots. All four
  scoped reads (`gitstatus`, `gitdiff`, `git/blob`, `blob`) now narrow by
  ref for all three owner kinds; decision-table test
  (`TestWorktreeScopedEndpoints`) added, no UI change needed. Reproduced
  live pre-fix (404 on `claude-d8c849`'s terminal graph), verified live
  post-deploy: the same request returns the PNG and the panel renders the
  previews (screenshots read: desktop-live and mobile-replay both paint).
  visual-review: PASS. Merged and deployed `07cc806`.
- **2026-09-05 — Connector catalog tabs merged and deployed.** Reconciled the
  Inspector rail work and passed combined `make ci`; deployed `075f6cce`. Live
  catalog tabs render with the real Claude Code host; `context7` shows "Added
  from Claude Code" and its service row stayed untouched; 44/44 panes
  survived. visual-review: PASS. No integration mutations or push.

- **2026-09-05 — Connector catalog tabs (branch).** Add-connector card gains
  a fixed Catalog tab (Custom card opens the former inline form as a dialog)
  and one tab per scanned agent CLI with found servers; host imports confirm
  destination and show an Added state; Use-from dialog and inline form retired
  on desktop and mobile. Domain helper `connectorTabs` unit-tested; `make ci`
  passed. Isolated-daemon browser E2E: custom add, host import and Added state
  on both apps; screenshots read, settled audits ok. visual-review: PASS
  (`connector-tabs-*.png`).

- **2026-09-05 — Inspector rail (ADR-0078).** Study
  `docs/benchmarks/2026-09-05-inspector-rail.md` (Paseo, Orca, t3code), ADR
  and `docs/plans/inspector.md`; `gitgraph.StatusWithStats` with Go tests;
  `lib/inspector.js` + `lib/resizeEdge.js` (17 node tests); `Inspector.jsx`
  mounted after `<main>`; file-tab Diff view; fixture seeds a dirty repository;
  `app-inspector` docs capture. Browser QA on an isolated fixture:
  `scripts/qa-inspector.mjs` passed 11/11 groups (`docs/screenshots/inspector-qa.json`);
  11 screenshots read, overlay/row audits ok. visual-review: PASS. Two defects
  found and fixed in review: the blocked line named the stale cwd (now read live
  from `…/cwd`), and the Diff view's header row was misaligned outside the
  folder tab. Merged `9a8e15a7` after three catch-up merges of main
  (pi-diff, Gmail, managed-stop) and deployed; served bundle
  `index-DYuuKYrY.js`, seven terminals survived. No push.

Older activity lives in `docs/handoff-archive.md`.
