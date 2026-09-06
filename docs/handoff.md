# Handoff — living project state

> Heartbeat of PiCode. A session that changes state leaves this file matching
> HEAD. Historical detail lives in `docs/handoff-archive.md`.

## Current state (read this first)

**This branch (`feat/llama-manager`):** delivery 1 of the owner-approved
[llama.cpp plan](plans/llama-manager.md) is implemented (ADR-0080). Models and
Server have dedicated desktop/mobile routes; Providers links to them and
labels credentials configured. Connection failures are typed and model
operations no longer hide unload failures or accept failed downloads.
Validation passed: make ci (893 JS/package tests, Go tests, both UI builds,
embedded binary, docs parity/build and Vale); 16 desktop/mobile captures read
with overlay audits ok. No merge or deployment.
The production/base-project record below is preserved from the branch point.

**Repository:** HEAD (deployed as `0.1.0+9a41241`) carries today's
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

**Last application deployment:** `0.1.0+aa9beea`, health `200`, desktop bundle
`index-DAL01ZST.js` (served and built hashes match). `make deploy` restarted
systemd; the seven terminal records survived. Live: the `glm5` agent's rail
reads `main ↑89` (upstream `origin/main`, nothing pushed — as this file has
recorded all along), the Git menu opens with its six actions and the overlay
audit passes; nothing was typed into a production terminal.
**Quality:** `make ci` passed on the Git-actions tree (Go tests including the
bare-remote ahead/behind case, 690 frontend/package tests, both UI builds,
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

- llama.cpp manager: owner-approved four-delivery [plan](plans/llama-manager.md),
  ADR-0080. Delivery 1 validated on `feat/llama-manager`; integration pending.
  Deliveries 2–4 remain planned, including managed-service ownership design.

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

1. Integrate the validated llama delivery 1 branch after reconciling current main;
   then delivery 2: capability detection, durable jobs, feed/SSE progress and
   reconnect/cancellation. Deliveries 3 and 4 remain in the approved plan.

1. **Sessions phase 2 debts**: codex machine-wide scan reads every rollout
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

- llama: browser acceptance uses synthetic router responses and Go HTTP
  fixtures. No installed-build/GPU or real model download/inference acceptance
  yet. Existing synchronous operations remain; navigation/restart recovery,
  server-side concurrency and cancellation belong to delivery 2. Catalog
  guidance and agent readiness belong to delivery 3. Service ownership and
  destructive cache operations belong to delivery 4's follow-up ADR.

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

- **2026-09-06 — llama.cpp manager, delivery 1 (branch).** Recorded all four
  deliveries before implementation. Added Models/Server navigation and typed
  connection results; corrected cancel/load, unload failure and download
  completion. Go HTTP fixtures and desktop/mobile browser matrix added.
  make ci passed (893 JS/package tests); docs gates rerun after guide copy.
  Browser matrix covers cancel, keep/unload others, unload, connection form
  validation and Providers entry on both apps. All 16 captures read, audits
  ok; visual-review: PASS. No merge/deploy; fixture/browser sessions closed.

Older activity lives in `docs/handoff-archive.md`.
