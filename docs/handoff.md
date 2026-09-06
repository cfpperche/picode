# Handoff — living project state

> Heartbeat of PiCode. A session that changes state leaves this file matching
> HEAD. Historical detail lives in `docs/handoff-archive.md`.

## Current state (read this first)

**Repository:** HEAD (`dcbaa316`, deployed) carries the tab-strip phase 1
(no scrollbar under the editor tabs, active tab revealed by code; study
`docs/benchmarks/2026-09-06-tab-strip-overflow.md`, phases 2–4 under Next
up), today's
worktree-scoped asset-preview fix (ADR-0073 amendment), the Integrations
rollout (ADR-0075) with connector catalog tabs and the Gmail recipe, the
desktop Inspector rail (ADR-0078, accepted by the owner) with its PR tab and
the **Git actions** stage (branch chip with ahead/behind, a Git menu preparing
fetch/pull/push/commit/PR commands in the owner's terminal), merged and
deployed as `aa9beea4`, bounded
captures (ADR-0076), the pi-diff TUI panel (ADR-0077 with the fullscreen
amendment) and the managed-stop process-group fix, plus File Tree v2 (0074),
worktree-aware Git Graph (0073), independent web apps (0072), Windows task
reliability (0071), Agent CLIs v2 and Docker v3. Managed agents remain
Pi-only; coding CLIs are terminals. Sessions moved under Agent CLIs
(ADR-0079, this branch): `#/clis/sessions(/<wsId>)` replaced `#/sessions*`.
No push was made. The capture ADR was renumbered because
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

**Last application deployment:** HEAD `dcbaa316` (tab strip phase 1) via
`make deploy` from the root checkout; systemd restarted, desktop answered
after reload. Verified on the live instance with seven terminal tabs:
strip gutter 0 px (was 10), tabs 39 px tall, computed `scrollbar-width:
none`, active tab fully inside the strip at either clipped side; screenshot
read (visual-review: PASS). The `/api/version` label was not read from the
browser session — the served bundle is proven by behaviour the previous
bundle lacked. Previous deploy `0.1.0+db12b097`, health `200` — the
sessions-endpoints fold (which includes the terminal-faces tree,
`fa97e8cf`, and the menu-item removal). `make deploy` restarted systemd.
Live: `GET /api/clis/pi/sessions` answers 387 real sessions with
`cleanupDays`, and the removed `/api/sessions/all` answers 404. Previous
deploy `0.1.0+fa97e8c` (terminal faces): collapsed **COGNIXSE**
(terminals-only) wore its agent-CLI favicons instead of "— empty";
**PiCode** showed a capped strip (`+5`); expand/collapse cycles and
reload persistence verified in-browser; console clean, overlay audit ok.
Nothing was typed into a production terminal.
**Quality:** `make ci` passed on the terminal-faces tree (Go tests,
frontend suites including `collapseFaces.test.js`, both UI builds,
embedded binary, docs parity with regenerated captures, Vale). Browser
acceptance: `scripts/qa-inspector.mjs` 15/15 groups on the scripted-gh
fixture (`docs/screenshots/inspector-qa.json`), commands proven typed and
never run through `tmux capture-pane -J`; `inspector-git-*` and
`inspector-commit-*` screenshots plus the live menu read, overlay/row audits
ok. visual-review: PASS. Fixtures and QA browser sessions are closed; the
feature worktree and branch are removed.

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
  `totals`, `branch` and `worktree`. The **PR** tab reads the branch's pull
  request through the host's `gh` (`GET …/pr`, states in 200, a minute cache,
  no background poller, no token in PiCode) and pre-types `gh pr create --fill`
  / `gh auth login` into the owner's terminal for the human to submit. The
  **Git actions** menu (Commit stage 1) prepares Fetch, Pull, Push, Commit,
  Commit and push and Create pull request the same way — an idle shell of the
  folder is reused, a terminal hosting a CLI never receives keystrokes — and
  the branch chip shows `↑ahead ↓behind`, `unpublished` or `detached` from
  `gitstatus`'s new `upstream`/`ahead`/`behind`/`detached` fields.
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

Deployed. The workspace `.pi/compact.json` is now committed (`a2cbad66`):
glm-5.3-flash, half the window, thinking off, no fallback. Real-compaction
acceptance with that config must still prove `fromHook: true`, its pricing and
no aborted turns.

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

- Tab strip phase 1 is merged and deployed (`dcbaa316`); the worktree and
  branch are removed. Phase 1
  of the tab-strip study (`docs/benchmarks/2026-09-06-tab-strip-overflow.md`,
  approved by the owner with the 3 px overlay indicator and `Alt+[` /
  `Alt+]`): the strip hides its layout scrollbar and reveals the active
  tab by code (`lib/tabStrip.js`). Phases 2–4 (wheel, edge fades, arrows,
  indicator, all-tabs list, keys, label cap) are listed under Next up.
- `feat/term-collapse-faces` awaits merge to main and `make deploy`. The
  collapsed workspace header's face strip now includes terminals after
  managed agents (`collapseFaceItems`, `TermFace` in `ProviderFaces.jsx`,
  wired in `Sidebar.jsx`): agent-CLI terminals wear their CLI favicon
  (vendor marks as fallback), shells the `>_` mark — a terminals-only
  workspace no longer shows "— empty", which now means truly no agents
  and no terminals. The docs fixture seeds project terminals and a
  terminals-only "sandbox" workspace (plus an empty "fresh"), so
  `make docs-shots` captures changed and were regenerated. Verified on
  the fixture in-browser (screenshots read, overlay audit ok, reload
  persists); not yet deployed to the live instance.
- `fix/worktree-blob-preview` awaits merge to main and `make deploy`; the
  running instance predates the fix (see Current state). Mobile keeps no
  worktree concept in its Changes screen, so nothing to ship there.
- `pi-diff` (ADR-0077): the owner chose project settings over a core flag —
  the workspace `.pi/settings.json` sets `tuiMode: fullscreen` (`3c447edf`),
  so every pi opened here starts fullscreen with the panel as a fixed column.
  Owner confirmed the panel works in a live PiCode terminal tab. Upstream ask
  filed: earendil-works/pi#9238 (`ctx.ui.setTuiMode` + `getLayoutRoot()`);
  new-contributor issues there are auto-closed until a maintainer reopens.

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

1. **Tab strip phases 2–4** (study `2026-09-06-tab-strip-overflow.md`,
   owner-approved): wheel → horizontal by dominant axis; `data-overflow` /
   `data-at-start` / `data-at-end` from one ResizeObserver + scroll
   listener driving edge fades and `‹ ›` arrows; 3 px non-interactive
   overlay indicator (hover/scroll, fades after 500 ms); "All tabs" list
   in `.main-tabs-end` with status dots, hidden tabs first, and a
   needs-you dot on the arrow of an off-screen tab; `Alt+[` / `Alt+]`
   plus `role=tablist` arrow-key focus; label `max-width` with ellipsis.
   Separate follow-up: the global `* { scrollbar-width: thin }` makes
   Chromium ignore every `::-webkit-scrollbar` rule in the app
   (`agent-clis.css` and `.dlg-sheet` already carry per-view overrides).
2. **Sessions phase 2 debts**: codex machine-wide scan reads every rollout
   (~6 s on 907 files; per-source mtime cache if it bothers anyone); codex
   preview skips injected instruction blocks by a narrow heuristic
   (`# `/`<` prefixes). Decide whether `/api/sessions/all` and
   `/api/pi-sessions` become deprecated aliases of the per-CLI endpoint
   (owner call). Grok sessions are prompt-history summaries — no
   transcripts exist in that format.
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
9. Inspector Git actions, next stages (ADR-0078): an optional "run when no
   agent is working here" mode that presses Enter behind the interlock (the
   step that amends the write refusals), and "ask the agent" variants beside
   each action for folders with a running agent. Merge/rebase/branch switch
   wait for a picker.

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
  `+N −M` footer beside the conversation is not drawn. PR tab: `gh pr view`
  runs with a 15 s timeout and answers are cached a minute per folder and
  branch, so a merge on GitHub shows on the next Refresh or anchor change,
  not live; `gh pr checks` detail beyond the rollup is not read.

## Recent activity

- **2026-09-06 — Tab strip phase 1: no scrollbar, active tab revealed.**
  Measured on the live instance with seven tabs: the strip's classic
  scrollbar took 10 of 39 px (Windows drew arrow buttons because
  `scrollbar-width: thin` disables the webkit rules in Chromium ≥ 121),
  the active tab sat at 1151 px in a 995 px viewport, wheel did nothing.
  Study written with receipts (VS Code, Zed, JetBrains, Sublime, Firefox,
  Chrome, MUI, Ant, Radix, Mantine, NN/g). Shipped: `.tab-strip`
  `scrollbar-width: none` + smooth `scroll-behavior` (reduced-motion
  aware, `overscroll-behavior-x: contain`), `revealLeft` in
  `lib/tabStrip.js` (6 tests) applied from a layout effect in
  `AgentTabs` on selection/open (instant on first paint). Verified on the
  worktree's Vite build against the live server: gutter 0, tab 39 px,
  active tab gap 0 on either clipped side. visual-review: PASS
  (tabs-after.png read; no overlay in this change; card 5/5).
- **2026-09-05 — Session management API folded into the per-CLI namespace
  (ADR-0079).** `/api/sessions/all`, `/api/pi-sessions(+/adopt)`,
  `/api/workspaces/{id}/sessions/manage` and `/api/session-cleanup` are
  removed; desktop and mobile use `GET /api/clis/pi/sessions
  [?workspace=|?cwd=]`, `POST /api/clis/pi/sessions/delete|adopt` and
  `GET/PUT /api/clis/pi/sessions/cleanup`, which now carry `inUseBy` and
  `cleanupDays` on pi rows and — the semantic gap closed — scope by
  workspace through `workspaceSessionDirs` (cwd bucket + each agent's
  private dir, ADR-0040). Delete is a POST action (ServeMux collision with
  the profiles routes), same guards: in-use → 409, outside root → 400.
  Decision tables migrated, not dropped: adopt, manage+sweep, in-use
  naming, machine-wide tagging. visual-review: PASS (pi machine-wide,
  workspace scope, delete overlay + file removed, auto-clean persisted,
  adopt created an agent with the copied session).
- **2026-09-06 — Collapsed workspaces show terminal faces.** A workspace
  with only terminals (shell / agent CLI) collapsed to "— empty"; the
  strip now renders terminals after managed agents — CLI favicons
  (Claude Code, Codex, Grok, Pi) with vendor-mark fallbacks and `>_` for
  shells, `faceSlice`-capped at 5 (`collapseFaces.test.js`; `TermFace` in
  `ProviderFaces.jsx`; `Sidebar.collapsedMark(agents, terms)`). Fixture
  seeds terminals + "sandbox" (terminals-only) and "fresh" (empty)
  workspaces; docs captures regenerated. In-browser review on the
  fixture: all four strip states read, expand/collapse cycle and reload
  persistence ok, console clean, overlay audit ok. visual-review: PASS
  (collapse-zoom2.png, expand-sandbox.png). `make ci` green on the
  branch.
- **2026-09-06 — Inspector Git actions, stage 1 (ADR-0078).** `gitstatus`
  gains `upstream`/`ahead`/`behind`/`detached`
  (`TestStatusWithStatsUpstreamAheadBehind` on a bare remote); the branch
  chip shows `main ↑2 ↓1`, `unpublished` or `detached`; a Git menu in the
  rail's header prepares Fetch, Pull, Push, Commit, Commit and push and
  Create pull request in an idle terminal of the folder through the `type`
  route (`gitActionCommand`, `shellQuote`, `gitActions`, `branchChip`,
  `commitMessageSchema`; `InspectorCommitDialog.jsx` on ResponsiveDialog with
  Zod errors and a command preview). The fixture seeds a bare origin one
  commit behind. `scripts/qa-inspector.mjs` 15/15 groups on the scripted-gh
  fixture (`tmux capture-pane -J` proves the commands are typed, never run;
  `gitstatus` keeps its four changes); `inspector-git-*` and
  `inspector-commit-*` screenshots read, audits ok. visual-review: PASS.
  Fast-forwarded main (no drift) and deployed `aa9beea4`; served bundle
  `index-DAL01ZST.js`, seven terminals survived. No push.

- **2026-09-06 — Deploy serialization + living-docs guards merged and
  deployed.** Fast-forwarded after reconciling three interleaved main moves;
  combined `make ci` passed; deployed `a8a201b` under the new lock (lock
  proven to block a second holder). Service healthy. No push.

- **2026-09-06 — Deploy serialization + living-docs guards (branch).** After
  two rounds of parallel-session interference, `make deploy`/`restart` now
  hold a lock across build+restart (a gate-to-restart race shipped the wrong
  tree once), and the pre-commit hook refuses commits where a staged
  `CHANGELOG.md` or `docs/handoff.md` lost its first-line shape — the
  signature of a session writing into the wrong worktree. Selftest grew three
  cases (15/15); the handoff-update skill starts by verifying the working
  directory. Root cause of the clobbering (a parallel session's stale cwd)
  remains behavioral — the guards catch it at commit time.
