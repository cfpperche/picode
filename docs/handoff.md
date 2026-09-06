# Handoff — living project state

> Heartbeat of PiCode. A session that changes state leaves this file matching
> HEAD. Historical detail lives in `docs/handoff-archive.md`.

## Current state (read this first)

**Repository:** `main` includes llama.cpp delivery 2 (ADR-0083), merged as
`70edb214` and deployed as `0.1.0+70edb21`, and the checklist sidebar
refinements + expand-on-click disclosure (ADR-0082, merged `e2cdc61f`,
deployed `0.1.0+1af83f4`). Models, Server and
Activity work on desktop/mobile. Load, unload and download now create durable
jobs, show per-file progress and recover observation after reconnect/restart
without replaying a mutation. Cancellation is enabled only for verified b10809.
Delivery 1 was deployed as `ac4ff1dd`; delivery 2 is live. Deliveries 3 and 4 remain planned in the
[four-delivery plan](plans/llama-manager.md). Other sessions' root work is
preserved; production deployment information below is inherited history.

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

**Last application deployment:** `1af83f4d`, version `0.1.0+1af83f4`, via
serialized `make deploy` from the root checkout (checklist refinements +
expand-on-click disclosure merged, ADR-0082). Health `ok`, systemd active,
8 terminals survived, `POST /api/terminals/nope/checklist` 404s live. The
sidebar disclosure is browser-QA'd on an identical bundle (isolated daemon:
expand, live flip, keyboard); on the live instance a card shows the
expandable line once its pi publishes (pi-checklist 0.2.0 on the process's
next start).

Previous deployment: `70edb214` (llama delivery 2 merged
with the checklists tree) via `make deploy` from the root checkout; systemd
active, `GET /api/terminals` 200 after restart. Verified live at 1000 px with
six terminal tabs: strip gutter 0 and `scrollbar-width: none`, three
overflow buttons (two arrows + All tabs), `mask-image` fades, indicator
mounted, `role=tablist`, `Alt+]` cycled the selection, label `max-width`
200 px, tabs `flex-shrink: 0`; the All tabs menu listed 6 items with the
"Out of view" group inside the viewport, overlay audit ok, screenshot read
(visual-review: PASS). The `/api/version` label was again not read from the
browser session — the bundle is proven by behaviour the previous one lacked.
Previous deploy `d1f1e9f9`, version `0.1.0+d1f1e9f`, via
serialized `make deploy` from the validated worktree after fast-forwarding
main (terminal checklists, ADR-0081). Health `ok`, systemd active. All 9
pre-deploy terminal IDs survived the restart; `GET /api/terminals` answers
and `POST /api/terminals/nope/checklist` 404s on the live instance. Terminal
cards show a checklist line once a terminal pi publishes (pi-checklist 0.2.0
picks up on the process's next start; existing pi processes keep the old
extension until restarted).

Previous deployment: `ac4ff1dd`, version `0.1.0+ac4ff1d` (llama.cpp
delivery 1) — live desktop/mobile old-link navigation, Models/Server pages
and Test connection passed; four production screenshots read, overlay/
alignment audits ok and no JavaScript page errors. The configured llama endpoint
`http://127.0.0.1:8080` times out, as it was unavailable before deployment;
no llama service was started and no model operation was executed in production.
Evidence and previous-binary recovery copy: `var/llama-deploy/`.

**Quality:** delivery 2 visual-review PASS: 16 Models/Server captures,
12 Activity captures, three real-router UI captures and four regenerated
public screenshots were read; overlay/alignment audits passed. Real b10809
CPU acceptance used four threads, 8192 context and one model at a time:
Qwen3-4B load/restart/reconnect produced exactly one load request; unload
completed; Qwen3-0.6B download recorded 5,297,108/428,970,080 bytes and confirmed
cancellation. Evidence: `docs/plans/llama-jobs-runtime-qa.json` and
`docs/screenshots/llama-jobs-{qa,live-ui}.json`. Full `make ci` passed with `GOMAXPROCS=4 GOFLAGS='-p=2 -count=1'`
(919 JS/package tests, Go tests, formatting/vet, both apps, embedded binary,
docs parity/build and Vale). Race tests passed for llama, llamajob and store. Test result caching was disabled after a local
Go cache-path lookup stalled; see the validation plan. Ordinary CI skips the opt-in real-router
`TestLiveLlamaJobs`; its explicit prerequisites and successful separate run
are documented in [the validation plan](plans/llama-manager.md#delivery-2-validation-and-limits).

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

- Checklist refinements + disclosure are **merged and deployed**
  (`e2cdc61f`, `0.1.0+1af83f4`; worktree and branch removed). Shipped: the
  counter `(x/n)` muted, an **absent** checklist rendering as silence
  everywhere (ADR-0082, owner call; data plane unchanged), and the
  **expand-on-click disclosure** (`checklistRows` pure + node-tested;
  `ChecklistDisclosure` on agent and terminal cards — ☑ dimmed, braille
  `PiSpinner` on the step being executed, ☐ pending; real button with
  `aria-expanded`, Enter/Escape, never navigates; 150ms grid-rows motion
  behind reduced-motion; live over the feed, no fetch). Open acceptance:
  see a live terminal card gain its expandable line once a terminal pi
  publishes with pi-checklist 0.2.0 (running pi processes pick it up on
  their next start); light-theme screenshot of the expanded list; mobile
  expansion (server already embeds `checklist` in terminal views —
  desktop-first scope).

- llama delivery 2 is complete in `feat/llama-jobs`; integration/deployment
  remains separate. Keep real validation sequential with four CPU threads
  and unload models afterwards. Production llama configuration was not changed. Owned QA servers are stopped.

- Tab strip phases 2–4 are merged and deployed (`139ab1ba`); worktree and
  branch removed. The study's adoption list is complete; its debts sit
  under Next up.
- Terminal checklists (ADR-0081) are merged and deployed (`d1f1e9f9`,
  `0.1.0+d1f1e9f`). Full stack: publish target fallback
  (`PICODE_AGENT_ID` wins, else `PICODE_TERM_ID`), `terminal_checklists`
  store + `terminal.checklist` events + `/api/terminals/{id}/checklist`,
  view fold in `liveTermView`, sidebar card line and terminal-pane strip,
  pi-checklist 0.2.0. Verified on an isolated QA daemon: present/
  live-update/absent/reset states and reload persistence. Open acceptance:
  a real pi process publishing through the extension in a live terminal (the
  POST contract is covered by Go tests; a terminal pi linked to this repo
  picks the new extension up on its next start — running pi processes keep
  the old extension until restarted), light-theme screenshot of the strip,
  mobile TermRow (server embeds `checklist` in terminal views —
  deliberate desktop-first scope).
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

1. Validate the deployed llama delivery 3 guidance dialog on the live service.
   Delivery 4 needs a concrete service-ownership/cache-deletion ADR.

1. **Tab strip debts** (study `2026-09-06-tab-strip-overflow.md`, all
   four phases shipped): the indicator covers
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

- **Sessions (ADR-0079) review debts:** the mobile session picker migrated
  URLs and is field-compatible with the new shape (code-verified) but has
  had no mobile visual pass; the `session_deleted` feed event regained
  workspace attribution in the review pass and is exercised by every delete
  test though no test asserts the payload itself; Conscious broadening, recorded in architecture.md: the unified delete
  accepts any orphan under the pi root — the old workspace-scoped route
  confined deletes to that workspace's dirs; the UI only lists in-scope
  rows, and the in-use/409 and root/400 guards are unchanged.


- llama: CPU real-model and b10809 job acceptance passed; GPU and other
  server-build cancellation remain unverified. An unknown download with an
  absent model retains its reservation because absence cannot prove completion;
  no forced unlock or automatic mutation replay is provided. History retains
  older SQLite records (UI shows latest 50 plus unresolved jobs); pruning is
  deferred. Model guidance/readiness is delivery 3; service ownership and cache
  deletion require delivery 4's follow-up ADR.

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
- **2026-09-06 — Pushed main to origin.** `298de6c9..1033aaf6` (163
  commits: everything since the last push — checklists terminal+disclosure,
  Inspector Git actions, tab strip, sessions under CLIs, llama deliveries,
  deploy guards and docs). origin/main == local main, no divergence; only
  tag remains `v0.1.0`.
- **2026-09-06 — Checklist refinements + disclosure merged and deployed
  (`e2cdc61f`/`1af83f4d`, `0.1.0+1af83f4`).** Reconciled two mid-flight main
  moves (llama delivery 2, drop-sessions cleanup); conflicts kept both
  sides' entries. Deploy verified: health ok, 8 terminals survived,
  unknown-terminal checklist 404 live. Disclosure interaction was
  browser-QA'd pre-deploy on an identical bundle. No push.
- **2026-09-06 — Checklist refinements + disclosure reconciled with llama
  delivery 2 (branch).** CHANGELOG/ADR-index/handoff conflicts resolved
  keeping both sides (llama ADR-0083 vs my ADR-0082); captures taken on
  main's side pending regeneration. Content unchanged from the two prior
  branch commits: muted counter, absent-silence (ADR-0082), and the
  expand-on-click disclosure (☑ / braille spinner / ☐), browser-QA'd on an
  isolated daemon. visual-review: PASS.

- **2026-09-06 — Dead `go("sessions")` branch removed.** The user menu was
  its last caller; the sessions debt note shrinks accordingly (routes.test
  drops the branch's own test with it).
- **2026-09-06 — llama delivery 2 completed in the feature branch.** Durable
  jobs, capability detection, file progress, cancellation and recovery with
  no mutation replay; decision-table tests and isolated real CPU acceptance.
  visual-review: PASS (35 screenshots read; overlay/alignment audits ok).
  ADR-0083, public guide, API schema and four-delivery plan updated together.

Older activity lives in `docs/handoff-archive.md`.
