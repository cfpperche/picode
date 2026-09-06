# Handoff — living project state

> Heartbeat of PiCode. A session that changes state leaves this file matching
> HEAD. Historical detail lives in `docs/handoff-archive.md`.

## Current state (read this first)

**llama integration:** delivery 1 (ADR-0080) is integrated with main `66aa4b9c`
on `feat/llama-manager`. Models/Server routes, connection diagnosis and
operation fixes are implemented; combined make ci passed (902 JS/package tests, Go tests, builds,
docs parity/build and Vale). All 16 desktop/mobile captures were read and
overlay audits passed; visual-review: PASS. Deployment is next.
The owner authorized merge and deploy. [Four-delivery plan](plans/llama-manager.md).

**Repository:** HEAD (`50df07f7`, deployed) carries the Inspector
run-when-idle stage (ADR-0078 stage 2, merged as `2da0ba15`), terminal faces
styled like agent faces, the tab-strip phase 1
(no scrollbar under the editor tabs, active tab revealed by code; study
`docs/benchmarks/2026-09-06-tab-strip-overflow.md`, phases 2–4 under Next
up), today's
worktree-scoped asset-preview fix (ADR-0073 amendment), the Integrations
rollout (ADR-0075) with connector catalog tabs and the Gmail recipe, the
desktop Inspector rail (ADR-0078, accepted by the owner) with its PR tab,
the Git actions stage (deployed as `aa9beea4`) and the **run-when-idle**
stage (PiCode presses Enter behind an advisory interlock, deployed as
`0.1.0+2da0ba1` — see Recent activity), bounded
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

**Last application deployment:** HEAD `50df07f7` (terminal faces in
strips match agent faces, on top of the tab-strip phase 1 tree) via
`make deploy` from the root checkout; systemd restarted. Verified on
the live instance: terminal favicons in collapsed strips render in
exactly the agent-face style — computed parity (white plate, 1px ring,
1px padding, 18px) across agent and terminal imgs; COGNIXSE wears
claude + openai, PiCode's strip is uniform with `+5`; screenshot read
(visual-review: PASS), console clean, overlay audit ok. Nothing was
typed into a production terminal. Previous deploy `0.1.0+2da0ba1`
(Inspector run-when-idle, ADR-0078 stage 2) via `make deploy` from the root
checkout at 11:54:36: health `200`, served bundle `index-W52Xj5hY.js` equal
to the built index, 10 terminals restored, the new
`POST /api/terminals/{id}/run` answers 404 for an unknown id, journal clean.
Live smoke in a Playwright Chromium with `ignoreHTTPSErrors` (agent-browser's
Chromium does not trust the mkcert CA) on the terminal anchored at
`~/picode`: the Git menu lists Fetch, Pull, Push, Commit…, Commit and push…
and the unchecked "Run when no agent is working here"; toggling stores
`picode-inspector-run=1`, reopening shows it checked, toggled back off;
overlay audit ok; no Git action clicked, nothing typed into a production
terminal (visual-review: PASS, both menu states read). Incident: at 11:49
the service was found `inactive` — terminated at 11:48:46 by another
session, two seconds before a sidecar instance on `:18097` started from a
production terminal — and a diagnostics probe `picode --version` from the
Inspector session fell through to server mode and served the production
data dir on 8445 for about four minutes until killed (clean shutdown); the
stage 2 deploy brought the service back. Before that `dcbaa316` (tab strip
phase 1), then `0.1.0+db12b097` — the sessions-endpoints
fold (which includes the terminal-faces tree, `fa97e8cf`, and the
menu-item removal).
Live: `GET /api/clis/pi/sessions` answers 387 real sessions with
`cleanupDays`, and the removed `/api/sessions/all` answers 404. Previous
deploy `0.1.0+fa97e8c` (terminal faces): collapsed **COGNIXSE**
(terminals-only) wore its agent-CLI favicons instead of "— empty";
**PiCode** showed a capped strip (`+5`); expand/collapse cycles and
reload persistence verified in-browser; console clean, overlay audit ok.
Nothing was typed into a production terminal.
**Quality:** `make ci` passed on the terminal-faces tree and on the stage 2
tree merged with main `dcbaa316` (Go tests including `TestTerminalRunRefusals`
and `TestTerminalRunTypesAndSubmits`, frontend suites including
`collapseFaces.test.js`, both UI builds, embedded binary, docs parity with
regenerated captures, Vale); the handoff-only merge of `0d346348` re-ran
`docs-check` and Vale before the fast-forward. Browser acceptance:
`scripts/qa-inspector.mjs` 16/16 groups on the scripted-gh fixture
(`docs/screenshots/inspector-qa.json`), commands proven typed and never run
through `tmux capture-pane -J`, and g16 proving run mode: the busy fallback
toast names the other terminal, a real commit lands through "Run in
terminal", then the fixture repo is reset. `inspector-git-*`,
`inspector-commit-*` and `inspector-run-*` screenshots plus the live menu
read, overlay/row audits ok. visual-review: PASS. Fixtures and QA browser
sessions are closed; the feature worktrees and branches are removed.

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
  `gitstatus`'s new `upstream`/`ahead`/`behind`/`detached` fields. With the
  menu's per-viewer checkbox **Run when no agent is working here**, the `run`
  route presses Enter itself only when its interlock finds the repository
  idle (no agent mid-turn, no TUI working, no automation, no other terminal
  working or holding a program, target pane at a shell); otherwise the command
  is prepared and a note names who is busy. The `type` route now refuses a
  moved terminal or a pane not at a shell; the rail takes a fresh terminal.
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

- `feat/tab-strip-phase2` (phases 2 and 3 of the tab-strip study,
  `docs/benchmarks/2026-09-06-tab-strip-overflow.md`) awaits merge to main
  and `make deploy`. Phase 2: `lib/useTabStrip.js` (ResizeObserver over
  strip + tabs, scroll and non-passive wheel listeners) drives `‹ ›`
  arrows that exist only while tabs overflow, edge fades via `mask-image`,
  a 3 px non-interactive indicator and vertical-wheel-to-horizontal.
  Phase 3: "All tabs" Radix menu (out-of-view first), needs-you dot on
  the arrow hiding such a tab, `Alt+[` / `Alt+]` in the app-keys catalog,
  `wireTermKeys` passthrough for Global chords, tablist roles with manual
  activation. Phase 1 is deployed as `dcbaa316`. Phase 4 (label cap)
  remains under Next up.
- llama delivery 1: combined validation passed; authorized merge/deploy in progress.
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

1. llama delivery 2: capability detection, durable jobs, progress and reconnect.
   Deliveries 3–4 remain planned; see the approved llama manager plan.

1. **Tab strip phase 4** (study `2026-09-06-tab-strip-overflow.md`,
   owner-approved): label `max-width` with ellipsis so one long name
   cannot swallow the strip. Debts from phases 2–3: the indicator covers
   the active tab's accent underline while shown (VS Code does the same);
   `scrollend` is the only exact "settled" signal, Safari falls back to a
   400 ms timer; the needs-you dot on an arrow was verified for CSS
   placement with an injected dot and by `hiddenTabs` tests, not on a
   live needs-you tab (none existed during QA); `.mtab-close` inside a
   `role=tab` is reachable by mouse only (tabIndex −1), close from the
   keyboard is not offered yet.
   Separate follow-up: the global `* { scrollbar-width: thin }` makes
   Chromium ignore every `::-webkit-scrollbar` rule in the app
   (`agent-clis.css` and `.dlg-sheet` already carry per-view overrides).
2. **Sessions phase 2 debts**: codex machine-wide scan reads every rollout
   (~6 s on 907 files; per-source mtime cache if it bothers anyone); codex
   preview skips injected instruction blocks by a narrow heuristic
   (`# `/`<` prefixes). Grok sessions are prompt-history summaries — no
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
9. Inspector Git actions, stage 3 (ADR-0078): "ask the agent" variants beside
   each action for folders with a running agent, through the channel that
   already carries prompts. Merge/rebase/branch switch wait for a picker.

## Known debts / open questions

- llama: browser/HTTP-fixture acceptance is synthetic; real installed-build,
  GPU and model inference acceptance remains pending. Synchronous operations
  remain in delivery 1; recovery/concurrency/cancellation belongs to delivery 2.
  Model guidance/readiness is delivery 3; service ownership and cache deletion
  require delivery 4's concrete follow-up ADR. No llama service lifecycle or
  model-file deletion is introduced in delivery 1.

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
- The desktop requests `/desktop/favicon.svg` at runtime and gets 404 while
  the static `<link rel=icon href="/favicon.svg">` answers 200 — seen in the
  live console during the stage 2 smoke; probably the dynamic tab-favicon
  code resolving a relative path. Not touched here; the `runtime-favicon`
  worktree may own it.
- QA fixtures for the Inspector died twice mid-run (wrapper exit 144, no
  panic, data dir left behind) when started as harness background tasks; a
  `setsid` fixture survived a full 16-group run, and one detached fixture still
  died 15 s after a browser opened it while another survived the same step.
  Working hypothesis: a concurrent session's process cleanup by name. Start
  QA fixtures detached, under a unique binary name if it recurs, and never
  `pkill -f` a pattern that matches the calling shell.
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

- **2026-09-06 — Tab strip phase 3: All tabs list, arrow dot, Alt+[ / Alt+],
  tablist.** `describeTab` now feeds both the strip and a Radix
  DropdownMenu listing every tab (out-of-view first via `hiddenTabs`,
  current one marked); `app.tab.prev` / `app.tab.next` join the app-keys
  catalog (Hotkeys dialog and Settings → Keys pick them up); `wireTermKeys`
  gained a `passthrough` predicate and both terminals pass
  `matchGlobalAction`, so Global chords no longer reach the shell
  (`termKeys.test.js`, `appKeys.test.js`; 37 JS tests across the three
  files). Manual activation replaced automatic after QA showed the
  terminal stealing focus on select. Verified on the worktree's Vite
  build at 1000 px with six tabs: roles and tabindex, Alt chords cycle and
  wrap with `defaultPrevented`, arrows / Home / End move focus without
  selecting, Enter selects and reveals, the menu lists 6 items with the
  "Out of view" group and separator inside the viewport (overlay audit
  ok), picking an out-of-view item selects and reveals it, the active tab
  stays visible when the overflow chrome appears. visual-review: PASS
  (p3-list.png, p3-arrowdot.png read; card 5/5).
- **2026-09-06 — Tab strip phase 2: arrows, edge fades, indicator, wheel.**
  `stripState` / `wheelToScroll` / `arrowStep` in `lib/tabStrip.js`
  (12 tests now), `useTabStrip` hook, `.tab-scroller` wrapper around
  `#tab-strip` (QA scripts keep matching `.main-tabs .mtab`). Verified on
  the worktree's Vite build against the live server at 1000 px with six
  tabs: left arrow disabled at start and right at end, `mask-image`
  switches side and shows both in the middle, indicator 3 px with opacity
  0 → 1 on hover and after scrolling, real wheel and dispatched wheel
  both scrolled and were `defaultPrevented`, a `deltaX` gesture was not
  consumed, nothing rendered at 1280 px where the tabs fit. Screenshots
  read for start / middle+hover / end. visual-review: PASS (p2-start.png,
  p2-middle.png, p2-end.png; no overlay; card 5/5).
- **2026-09-06 — llama manager integration.** Owner authorized merge/deploy
  of delivery 1; main incorporated into the isolated feature branch. Code
  merged automatically; living docs reconciled and captures regenerated.
  make ci passed; 16 browser captures read and audits ok; visual-review: PASS.
  Original implementation is `467cd556`; deployment follows the merge.

- **2026-09-06 — Inspector run-when-idle (ADR-0078 stage 2).**
  `internal/server/git_run.go`: `POST /api/terminals/{id}/run {text, root}`
  types and submits in the user's shell behind `repoBusy` (agents mid-turn via
  the runtime snapshot, TUIs via `LooksWorking`, automation runs, other
  terminals by CLI state or foreground program via `PaneCommand`, the target
  pane at a shell; repository identity by git common dir); 409 `moved` /
  `foreground` / `busy` naming who; the `type` route shares the root and
  foreground guards. Tests: `TestTerminalRunRefusals` with injected probes,
  `TestTerminalRunTypesAndSubmits` on a real tmux shell (a created file is the
  proof). Client: Git-menu checkbox `picode-inspector-run`, `run` delivery
  with fallback note and a fresh-terminal retry, dialog reads "Run in
  terminal". ADR-0078 now states the amendment to the write refusals of
  0022/0032/0038/0073, and the ADR index says so on each of them. Merged as
  `2da0ba15`, deployed as `0.1.0+2da0ba1` and contained in `50df07f`
  minutes later. visual-review: PASS (`inspector-run-busy-fallback-dark.png`,
  `inspector-run-commit-dialog-dark.png`, `inspector-run-commit-dark.png`,
  live Git menu read in both checkbox states).

Older activity lives in `docs/handoff-archive.md`.
