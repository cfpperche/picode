# Handoff archive

Moved off `docs/handoff.md` when it exceeded ~150 lines. Newest living
state is always `docs/handoff.md`. Do not treat this file as current.

## Recent activity (archived 2026-09-01)

- **2026-08-31** — sidebar version reflects the build (`fix/version-truth`):
  `version.Build()` appends the Go-embedded `vcs.revision` (+ dirty `*`)
  unless the release workflow stamped `version.Stamped`; wired into
  `/api/version` (new `semver` field keeps comparisons pure), `picode
  update`, and the sidebar (ellipsis + title for narrow widths).
  install.Newer/backup keep plain `Version`. Tests in `version_test.go`.
  Follow-up in the same session: the dirty `*` marker was dropped — Go
  stamps `vcs.modified` against the primary checkout (not the linked
  worktree being built), and with parallel agents that checkout is
  routinely dirty, so the flag was pure noise; the revision alone is the
  signal.

- **2026-08-31** — roles follow-up (`fix/roles-gate`): the `role-state`
  endpoint is gated on pi-roles being in the agent's effective package
  list (`agentHasRolesPackage` over `loadPackageReport`; honors
  `PackagesIsolated`, fails open on listing errors) — an uninstall no
  longer leaves an orphaned composer chip. `/roles auto` accepted as an
  alias for `/auto` (pi-roles 0.5.1). Gate covered in
  `roles_state_test.go` (valid v1 file + no package → null; back → state).

- **2026-08-31** — **/roles: active-role chip, restored locks, rich
  pickers, scoped remove** (`feat/roles-active`, planned + approved;
  ADR-0033 amendment #2). pi-roles 0.5.0.
  - **State contract v1**: extension writes
    `~/.pi/agent/roles-state/<agent>.json` on mode/roles changes
    (`stateJson`/`parseState` in logic.ts, tested); `session_start`
    restores a lock whose role still resolves (model applies on next
    input, never at startup). Server: `GET /api/agents/{id}/role-state`
    (`roles_state.go`, null on missing/broken/future-version, tested).
  - **Composer chip** (`RoleChip.jsx`, SearchCombo like the other chips;
    presence-driven, no settings flag): `auto` quiet, lock in accent with
    a Lock icon; dropdown = roles with definitions + Edit roles…;
    picking sends `/role <name>`/`/auto` through sendTask. Refetch on
    select/snapshot/settled/notify.
  - **Rich select labels** (`roleOption`/`roleFromChoice`) in
    `/roles` picker, edit and remove; web trims the decoration off pills
    and definition lines.
  - **Scoped remove** with smart-skip (`removeScopes`): one layer →
    no question; both → `Remove from` select; without the env →
    workspace, as always. `Removed x (scope)` renders as its own line.
  - Bugs caught while verifying: missing `BACK` import in roles.ts
    (remove crashed); RoleChip's `triggerClassName` replaced
    `cockpit-chip` instead of extending it (squished chip). Also learned:
    a follow_up carrying an extension command is rejected by pi — only
    reachable via direct API posts (the web queues follow-ups locally).
  - Decision table 1–14 verified on the scratch rig (state file content,
    restore across restart, fallback-to-auto when the role vanishes,
    chip present/absent/lock, dropdown pick, rich pickers, remove both
    scopes, dark, reload); rows 13–14 by unit tests + design (TUI).
    Gates green (286 js + 69 pkg + Go incl. new endpoint test).
  - **Note (env)**: agent-browser now runs in a named session
    (`AGENT_BROWSER_SESSION`) — the default shared browser was hijacked
    mid-QA by another agent's session; use named sessions in dogfoods.

- **2026-08-31** — **`/roles` empty-state copy + notify-as-thread-line**
  (`fix/roles-empty-copy`). Owner asked why `/vision` is listed with no
  roles configured — it is (commands register at package load; config is
  read at run time, ADR-0028 dormant contract) — and approved fixing what
  it *answers* instead: lock commands with no config now say
  `No roles yet — /roles edit vision creates the first one.` (logic.ts),
  and a notify that answers a still-quiet slash segment becomes a thread
  **note line** (new item kind `note`: mark + command badge + text, the
  `/roles …` fragment as a prefill chip) instead of a fading toast —
  `slashNoteTarget()` in askForm.js decides; ask-memory persists notes.
  **Bug found while verifying**: `groupTurns` (turns.js:10) silently
  swallowed the new item kind — the note rendered nowhere despite the
  handler running; fixed by adding `note` to the loose kinds.
  Verified on the scratch rig (line renders, chip prefills, reload keeps
  2 notes); gates green (276 js + 60 pkg).

- **2026-08-31** — **`/roles` chat UX pass** (`feat/roles-chat-ux`; owner
  verdict on the shipped surfaces: "que porcaria de UIUX … texto seco,
  zero identificação"). P0+P1 of the approved plan:
  - Ask cards remember the slash command that opened them (`cmd` on the
    card, `cmdOf()` in `askForm.js`) — the open stepper now has a
    `ROLES <args>` header with Cancel in the corner; pills connect with
    `›`; the combo trigger names the field ("Choose provider…").
  - Confirms are a block (question + file chip + verbs); titles starting
    Delete/Remove get **Delete** (danger-at-rest, scoped to `.ask-confirm`)
    / **Keep** instead of Yes/No.
  - `summaryLine` refactored over a typed `summaryParts()` (definition /
    role / cleared / kept / empty / text); `AskOutcome` renders finished
    flows as one-liners with mark + `ROLES` badge + chips (provider icon
    via `ProviderFace`, thinking, scope, file). The nothing-to-clear line
    carries a "Set one up → /roles add" chip that prefills the composer
    (`onPrefill` threaded App → ChatSurface → Conversation).
  - Extension 0.4.1: confirm-No notifies `Kept <rel>` so the line stops
    degrading to "this agent · No". TUI otherwise untouched.
  - Verified on the scratch rig, light+dark, reload persistence (cmd and
    note serialize through ask-memory; cancelled still drops by design),
    overlayAudit ok on the dropdown. Gates green (274 js + 60 + 60 pkg).
    P2 (slash-bubble compaction, spacing rhythm) not started — follow-up.

- **2026-08-31** — MCP adapter detection from a terminal tab. `#/mcps`
  passed `selectedId` (`t:…`) as `agent`, GET 404'd, and the catch painted
  "Install the MCP adapter" even with `npm:pi-mcp-adapter` on the machine.
  Same bug Packages already fixed (3b3713e3). Server ignores a non-agent
  *and* a non-workspace id; a workspace terminal still carries that
  folder as MCP/Packages context so the machine list stays; the pane
  passes `agent.id`. A load error is Retry. Table-tested.
  **visual-review: PASS** (empty+adapter, blocked Open packages, error
  Retry; overlayAudit ok; card 5/5). First deploy was overwritten by
  `main`'s binary — user still saw the bug on :8445 until redeploy.
  That repeated on 08-31 evening: three more deploys from `main` (git
  graph work) shipped without this branch until it was merged.

- **2026-08-31** — Remove workspace can delete local data (ADR-0035):
  opt-in checkbox + GitHub-style typed folder-name confirmation; server
  re-verifies the name and refuses root/home (guard sabotage-tested);
  remote repo never touched. Clone-form segmented now full-width with
  folder/git icons. QA on 8448: wrong name keeps Remove disabled, right
  name deletes the folder from disk, plain remove keeps it; light+dark
  read, overlayAudit ok. **visual-review: PASS** (shots 10-13; card 5/5).

- **2026-08-31** — **pi-roles: choose the save target; `/roles clear`**
  (`feat/roles-scope`, ADR-0033 amendment). Owner asked how the chat
  session picker scopes (folder, confirmed by reading `session.List` /
  `AgentCwd` — sessions and roles are both per-cwd, not per-agent; only
  `agent.SessionPath` differs) and then approved this instead of only
  documenting the workaround (`cp` the overlay to the workspace file, or
  edit from a plain terminal).
  - `pickStart`/`pickAnswer` (`packages/pi-roles/src/logic.ts`) gained a
    `scope` stage: under `PI_ROLES_AGENT`, thinking is followed by a
    **Save to** select (*this agent* / *workspace* / `‹ back`); without
    the env the question is skipped, as before. `editFlow`/`addFlow` in
    `extensions/roles.ts` resolve a `layerFor()` (agent overlay vs.
    `.pi/roles.json`) from the answer.
  - New command **`/roles clear [agent|workspace]`**: confirm, then
    delete the whole file. No arg under the env asks which; a lock whose
    role stops resolving falls back to `/auto`.
  - Chat: `fieldLabel`/`summaryLine` (`web/src/lib/askForm.js`) learned
    the `Save`/`Clear` labels and the `(workspace)` suffix on the
    definition line.
  - **Bug caught in the same dogfood pass, fixed before shipping**: the
    existing role-picker regex (`/\broles?\b/`) also matched "Delete
    this roles file?", mislabeling the confirm's "Yes" as a role name —
    `/roles clear agent` rendered `Yes — .pi/roles/qa-213680.json`
    instead of `Cleared …`. Narrowed to `/^roles\b/` (only the role
    *picker* titles) and made the delete-confirm title itself carry the
    `Clear` label too, since the arg form (`clear agent`) skips the
    select step and needs the confirm alone to still gate the
    note-vs-fallback branch in `summaryLine`. Regression tests added
    (`askForm.test.js`, `logic.test.ts`).
  - Verified live on the roles-adversarial scratch rig (`PICODE_DATA`
    + port 8471, scratch `HOME` pointed at this worktree's package):
    Save-to appears/steps back correctly, workspace vs. agent writes
    land in the right file, `/roles clear` (both arg and no-arg forms,
    both scopes) deletes and reports correctly, "nothing to clear" warns
    without a stuck card, the ordinary `/roles` role picker still labels
    "Role". Gates: fmt/vet/test/test-js green, `npm test` 60/60,
    `make build` ok. **Merged to main and deployed to :8445.**

- **2026-08-31** — New workspace can clone a remote repo (ADR-0034):
  "Local folder | Clone repository" switch in the same dialog, URL →
  derived editable name/destination, blocking `POST /api/workspaces/clone`
  with host git credentials and prompts disabled, same-origin destination
  adopted. New `internal/gitclone` package; argv-injection defenses
  sabotage-tested. QA on an isolated 8447 server: real clone of
  octocat/Hello-World, `/tree/<branch>` honored, adopt, 409, classified
  not-found error, dark + 480px drawer, overlayAudit ok.
  **visual-review: PASS** (shots 01–09; card 5/5).

- **2026-08-31** — **`/roles` chat stepper: adversarial review + redo**
  (`fix/roles-adversarial`, **merged to main and deployed to :8445**;
  running agents must be restarted to reload the path package). The
  owner called the shipped stepper
  broken; the review confirmed the cancel-as-back design was the root
  cause and replaced it. What changed:
  - **Extension (ADR-0028 amendment):** cancel aborts the whole flow;
    going back is an explicit `‹ back` option on selects with a prior
    field. `pickAssignment` is now a pure state machine in
    `packages/pi-roles/src/logic.ts` (`pickStart`/`pickAnswer`, tested).
    A lock that lands on the already-running model still notifies, so the
    UI always gets a definition. pi-roles → 0.3.0.
  - **Web:** optimistic Working ends at the first real signal (dialog,
    notify, answer, `task_delivered` + 3s fallback) — never sticks;
    the completion notify folds into the card as its definition line
    (`default — xai/grok-4.6 · high`); back-walk answers `‹ back`
    instead of auto-cancelling; ask-memory gained a per-agent live slot
    (fresh agents persist across reload) and no longer cross-writes
    slots on tab switch; process exit / agent stop closes open cards;
    Stop shows while waiting, not only streaming; a queued follow-up is
    marked sent at flush time (was delivered twice).
  - **Go:** `deliver`/`SendTurn` no longer fail a task at the 60s
    deadline while a dialog is pending (a human thinking in a picker is
    delivery, not failure) — this was the red `context deadline
    exceeded` bubble. Regression test `TestSlowDialogIsDeliveredNotFailed`.
  - Decision table rows 1–9 verified in a scratch instance by browser
    (screenshots read; overlayAudit ok). Row 10 (TUI) verified by the
    package state machine tests + design (still one select at a time;
    Esc now aborts instead of stepping back — documented in README/ADR).
  - **Debt:** the TUI flow was not exercised interactively this session
    (no tmux dogfood) — behavior change is Esc=abort + visible `‹ back`
    rows. If the owner dislikes `‹ back` in the TUI list, the option can
    be dropped there only at the cost of reintroducing cancel ambiguity.
  - **Not done (refused per owner):** no `#/roles` page, no modal wizard;
    the flow stays in the conversation (C1).

- **2026-08-31** — Ask back-step: reopen the clicked pill, skip mismatched
  dialogs. Merged and deployed. Reload the agent. **superseded by the
  adversarial redo above** (auto-cancel walk removed).

- **2026-08-31** — Ask form UX: definition line persists, back on pills,
  compact stepper. Merged and deployed. Reload the agent for `/roles` back.
  **visual-review: UNVERIFIED** (owner dogfood).

- **2026-08-31** — File tree header shows the full folder path (mono,
  ellipsized, `title` tooltip) instead of just the basename — the owner
  noted the tab strip already carries the name, the header is where you
  confirm which folder. `.ft-title` gained `min-width:0` so a long path
  can no longer push Reveal/Refresh/Close off the header.

- **2026-08-31** — Extension select in chat is one growing form (pills +
  filter dropdown), then a pill line. Merged and deployed.
  **visual-review: UNVERIFIED** (owner dogfood on 8445).

- **2026-08-31** — Z.AI Usage parse: GLM Coding Plan now sends
  `CREDIT_LIMIT` (unit 3/5 = 5h, unit 6/1 = week) instead of
  `TOKENS_LIMIT`. Live Pro payload (0% / 100%) is the test fixture.
  Merged `fix/zai-credit-limit`.

- **2026-08-31** — pi-roles v2 (ADR-0033): per-agent overlay via `PI_ROLES_AGENT`.
  Workspace `.pi/roles.json` stays the default; `/roles` in a PiCode agent
  writes `.pi/roles/<id>.json`. Merged and deployed. Reload the agent.
  **visual-review: n/a** (no UI chrome).

- **2026-08-31** — **Provider Usage V3** (ADR-0031). Usage is per vault
  account (`GET/POST /api/providers/{id}/accounts/{aid}/usage[/reset]`)
  without swapping `auth.json`. OpenRouter / MiniMax / MiniMax CN / Kimi
  API keys get meters. Grok resets try PiCode OAuth, then Grok CLI
  `~/.grok/auth.json`, then `GROK_COOKIE`. Qwen Token Plan stays hidden
  (no API-key quota JSON). Chrome cookie dump refused. QA on :8451 —
  Usage on each account row, OpenRouter credits, empty/error/auth
  overlays. **visual-review: PASS**. Merged `feat/provider-usage-v3`.

- **2026-08-31** — Stop during `/roles` cancels the dialog and clears Working.
  Merged `fix/abort-clears-extension-wait`.

- **2026-08-31** — **File tree V2** (ADR-0032): Changes rows expand into
  working-tree diffs (`gitgraph.WorkingDiff`; `gitLoose` keeps stdout on
  --no-index's exit 1, /dev/null fallback only for ls-files-empty paths —
  both sabotage-proven); `POST …/reveal` opens the folder in the host
  file manager (`internal/osopen`, extracted from backup, WSL dedup);
  focus/visibility refresh instead of polling; branch pill badges
  `gitinfo.Dirty` (porcelain -uall count; cost: +1 subprocess per row per
  list, recorded in the ADR with the ?dirty=1 fallback).

- **2026-08-31** — `/roles edit|add` picks provider, then model, then thinking.
  Merged and deployed. Reload the agent to load the path package. **visual-review: n/a**.

- **2026-08-31** — **Provider Usage V2** (ADR-0031). ZAI and OpenCode Go
  API keys get Usage. Codex/Grok banked resets show in the dialog; Redeem
  confirms then `POST /usage/reset`. Grok reset row is omitted if the
  grok.com call needs cookies — weekly windows still load.

- **2026-08-31** — Sidebar row2 refined into pills (owner feedback after
  testing the file tree): `.ws-pill` — [folder + dir] opens the tree,
  [git + branch] opens the graph; repoLine/termLine expose `dir` alone.
  QA on :8501 — repo/non-repo rows, dark, 180px narrow. **visual-review:
  PASS**.

- **2026-08-31** — Packages Installed row is back when opened from a
  terminal tab (machine packages were never deleted). Merged
  `fix/packages-list-when-terminal`.

- **2026-08-31** — **Provider Usage dialog** (ADR-0031). `#/providers` shows
  **Usage** only when `quotaKind` matches the active oauth slot (Claude,
  Codex, Copilot, Kimi, xAI). `GET /api/providers/{id}/usage` fetches vendor
  windows in-process; tokens stay on the server. Dialog: skeleton → bars /
  empty / error / Sign in. Statusbar still does not invent quotas.
  visual-review: PASS (usage-windows/empty/error/loading/auth, overlayAudit ok).

- **2026-08-31** — **File tree per workspace/terminal/agent** (ADR-0030):
  `#/tree/<w|t|a>/<id>` opens a read-only tree of the owner's folder, tab
  deduped by canonical root (`d:<root>`, owners in `picode-tree-owners`).
  A **Changes** section on top lists `…/gitstatus` (porcelain `-z -uall`,
  re-anchored to the owner's cwd; rename records consume two NUL fields —
  sabotage-proven); changed files and ancestor folders carry kind dots.
  Workspaces became file-reading owners (`browse|text|blob|file|gitstatus`,
  `ws_free` refused) so empty workspaces (ADR-0027) browse too; terminals
  gained `/browse` at the live pane cwd; `browseAgentDir` answers `root`.
  Entry points: row2 folder icon (agents/terminals), Files on the
  workspace card + palette. Terminal trees pin; manual Refresh after a
  `cd` renames/merges the tab. Known limit inherited from the git graph:
  navigating by URL to an owner created seconds ago can flash "gone"
  until the app's list refreshes. QA: dedupe, cd+rename, empty/non-repo/
  deleted-folder states, dark/light, overlayAudit ok.

- **2026-08-31** — User menu fit: popover 236→268px so the Theme/Layout
  segments ("Desktop · Auto · Mobile" + icons, mono 12px) stop clipping;
  segments are equal-width and centered; Install app spans the row
  (`.um-install`) and carries IconDownload (mobile drawer inherits the
  icon). Isolated visual on :8507. **visual-review: PASS**.

- **2026-08-31** — Merged `feat/model-roles` as ADR-0028/0029 (numbers
  shifted: main already had 0025 tmux catalog and 0026 sidebar). Isolated
  visual on :8477; installed :8445 untouched. **visual-review: PASS**.

- **2026-08-31** — **Workspaces start empty** (ADR-0027): POST /api/workspaces
  registers the folder only; the New-workspace form is name + folder
  (Provider/Model/Thinking and the session shortcut stay on agent forms);
  `workspaceView.Agent` is a pointer with omitempty (the zero-object read
  as a truthy agent everywhere); empty-workspace open/close/sessions/status
  answer 409. connectPanel now takes the agent id (fixed: with two agents
  it connected to the workspace's first). Also shipped: workspace cards
  wear the project favicon (`GET /api/workspaces/{id}/favicon`, confined,
  sandbox CSP; fallback IconFolder — debt: a favicon added mid-session
  shows only after reload, and faviconRels is a fixed list), the
  Workspaces tab icon is lucide Folders, and the folder picker's address
  bar filters/navigates as you type (lib/pathFilter.js + useDebounced).


- **2026-08-31** — **ADR-0036 + ADR-0037 accepted** (docs only, no code
  yet): extensions host — apps as manifest + schema-driven primitives
  (no in-process JS ever, iframe deferred to v2, WASM deferred), fifth
  sidebar tab with an app grid seeded first-party; and the Inbox — core
  data plane (SQLite mailbox, `POST /api/inbox`, blocking-count badge)
  with the view as the first app, `packages/pi-inbox` giving agents
  async `notify_human`/`ask_human`. Web-benchmark research with sources
  in both ADRs. Next step: implementation plan for the 0036 pipeline.

- **2026-08-31** — **Apps host pipeline (ADR-0036)** on branch
  `worktree-feat-apps-host`: `internal/apps` (Manifest/Badge/Host,
  primitives View/Block/ListItem/Form/Field/Action + Validate, explicit
  Registry, hidden demo app), routes `GET /api/apps` (badges inline,
  failure-proof), `GET /api/apps/{id}/view?path=`, `POST
  /api/apps/{id}/action`; `Deps.Apps` nil-safe; env read in cmd only.
  Web: `x:<id>` tabs + `#/app/<id>` (all six dispatch points), fifth
  sidebar tab with grid/badges (tabs tighten to 26px under 240px so
  PiCode fits at 180px), AppSurface renderer (Field methods mirror
  rpc.UIDialog; Confirm→ConfirmDialog, toast/view/path results),
  palette entries, `lib/appPrimitives.js` normalizers. QA on isolated
  :8611 with PICODE_DEMO_APP=1: grid+badge, list→detail→danger
  confirm→toast+replaced view, path navigation, form (4 methods),
  gone card, no-env placeholder + empty `apps: []`. `make ci` green.
  Note: first screenshot at 1.5s settle caught boot mid-flight —
  use `--wait-ms 4000` for boot-dependent shots.

- **2026-08-31** — **ADR-0036 amended** (owner decision after the host
  shipped): in the marketplace era the sandboxed iframe (separate
  origin + bridge + published tokens/component package) is the
  first-class body surface for third-party apps; primitives stay as the
  cheap default, the host-chrome tissue, and the ONLY surface for
  sensitive actions (agent approvals, destructive confirms — tokens
  don't stop phishing, host-rendered controls do); the primitive
  vocabulary is frozen at the four blocks. v1 refusals unchanged.

- **2026-08-31** — **Inbox (ADR-0037)** on branch `worktree-feat-inbox`:
  migration 014 `inbox_items` (no FKs — items outlive sources), store
  CRUD + `RespondAndForward` (verb/state validated BEFORE forwarding —
  a rejected verb must never enqueue; caught by test), runtime filing
  in `pumpEvents` (`Hub.Len()` unobserved gate, `lastFinal` from
  agent_end, `stopRequested` so manual Stop files nothing), routes
  `POST/GET /api/inbox` + respond (409 + annotation on dead agent) +
  state/snooze, `internal/apps/inbox.go` seeded in `BuiltIns` (grid no
  longer empty), `packages/pi-inbox` (MIT; notify_human/ask_human with
  terminate:true, node:https loopback, soft failure when PiCode is
  down), `PICODE_AGENT_ID` in SpawnEnv. Fixed in passing: AppSurface
  load race (busyRef ate a navigation racing a focus refetch —
  latest-wins seq now). QA on :8612: 4 kinds via curl, badge 2+dot,
  needs-me/feed, question detail form → reply → toast → follow_up task
  `queued/source=inbox` verified in SQLite, dead agent 409 + body
  annotation + item open. `make ci` green. Live pi smoke of pi-inbox
  (`pi -e packages/pi-inbox/extensions/inbox.ts` + ask_human) left for
  the owner — spends provider credits.

## Recent activity (archived 2026-08-30)

- **2026-08-30** — **Sidebar restructured into four flat tabs** (ADR-0026):
  Agents (free, flat, name-sorted), Workspaces (one collapsible card per
  workspace — section collapses are gone), Terminals (free only), Pins.
  Workspaces now own terminals: migration 013 adds
  `terminals.workspace_id` (default `ws_free`, no FK — SQLite refuses ADD
  COLUMN with REFERENCES + non-NULL default; cascade is app-driven),
  `POST /api/terminals` takes `workspaceId` and a workspace terminal is
  born in the workspace folder. Removing a workspace kills its terminals
  (tmux best-effort, records + settings in one tx) and the cleanup dialog
  warns with the preview's count. Wire stays flat; grouping is client-side
  (`web/src/lib/termGroups.js`). Group hover actions became an absolute
  overlay — four buttons reserving grid space squeezed the workspace name
  to nothing at 180px. The brand version yields below 254px. Known
  behaviors, by decision: a stored `picode-side-tab:"agents"` now shows
  only free agents (no migration — indetectable; empty states carry the
  action); V1 has no move-terminal-between-workspaces; a tmux kill that
  fails after the DELETE leaves an orphan session recoverable via the tmux
  catalog (ADR-0025).

- **2026-08-30** — The last ADR-0025 debt is paid: tmux **array options are
  editable** in `#/termset` (`command-alias`, `terminal-features`,
  `terminal-overrides`, `status-format`, `update-environment`). They are
  edited as text, one entry per line; **Start from inherited** copies the
  inherited entries in; Apply rewrites the list per index and unsets whatever
  the layer held past the new length — measured first: tmux leaves a stale
  `name[2]` in place forever otherwise, and a whole-option unset before the
  rewrite resurfaces the layer below. An empty block is refused (tmux keeps no
  empty array layer, so it would be a pin that behaves as inherit).
  Browser QA found a bug that predates arrays: the **global** panel dropped a
  cleared non-curated key from the store but never unset it on the live
  sessions, so it kept applying. Fixed with `unsetClearedEverywhere` and a
  test that fails without it. Also fixed: `.dlg-input` pins height to one
  control row, which collapsed every list editor to a single line.
- **2026-08-30** — Agent cards got the terminal treatment: the name is the
  rename control (hover accent + dotted underline, click opens "Rename
  agent", `PATCH /api/agents/{id}`), prefilled with the shown name so a
  workspace `default` agent never opens a blank field. **No gear was added**:
  measured in the browser, the hover action row already spans x=120–230 of a
  244px sidebar, so a fifth icon would have cut the name's clickable run from
  49px to 23px. Debt (pre-existing, not from this change): with four actions
  the overlay covers the tail of a long agent name on hover — "Claude Code"
  reads "Claude". The full name is still in the `title`.

- **2026-08-30** — Terminal rows in the sidebar lost the pencil: the name
  itself is the rename control (hover paints it accent with a dotted
  underline, click opens the rename dialog), and the hover action row is
  now remove then settings, so the gear is the last icon on the line. The
  rest of the row still selects the terminal; the name button claims only
  its own text (`flex: 0 1 auto`), not the whole line. Owner's request from
  a screenshot.

- **2026-08-30** — Terminal appearance moved into `#/termset` ("Appearance —
  this browser" section, global page only); Preferences lost its Terminal
  tab and `#/preferences/terminal` degrades to Appearance. Storage homes
  unchanged (ADR-0024 amended). Shared pieces extracted: `ThemeCard.jsx`,
  `TermAppearance.jsx`.

- **2026-08-30** — Follow-up caught by the owner's screenshot: the SELECTED
  terminal showed `~ / main` — an impossible pair. `POST /open` still
  answered with the record cwd and no git; the app merges that response into
  its list, so the stale path overwrote the live one while the old git
  survived. All four terminal-returning handlers now share `liveTermView`;
  a test opens a terminal after a `cd` and asserts the live path comes back.
- **2026-08-30** — Sidebar cards unified (terminals ↔ agents): second line is
  icon + path, or git icon + `path / branch` in a repo. `GET /api/terminals`
  now reports the live pane cwd (it printed the creation dir forever while
  the git facts beside it were live — the two disagreed after any `cd`).
  Workspace agent views carry per-agent git from the agent's effective dir;
  `repoLine` returns the git object (fixing the tooltip that read `.branch`
  off a boolean) and never pairs one directory's path with another's branch.

- **2026-08-30** — **Compact status moved into the chat** (merged from
  `feat/compact-chat-line`, Claude Code-inspired, PiCode tokens). The
  “Compacting” segment is gone from the composer statusbar; in-flight
  compaction is now a live line at the end of the conversation — pulsing
  accent dot (`.work-dot`), “Compacting session…”, elapsed in the chat's
  `turns.js` `1m 05s` format — and the finished compact folds into the
  existing one-line collapsible `compaction-card`, which `compaction_end`
  now fills from `ev.result.summary` so auto-compacts land live instead of
  on next reload (dedup by summary text; user-initiated flow still replays
  via `loadSessions`). “Nothing left to compact.” and failures are chat
  alerts; `picode-compacting` localStorage survives reloads/rebuilds.
  `make ci` green. **Visually verified** on an isolated scratch server
  (8468) with a crafted session, screenshots read: light collapsed card +
  live line, dark expanded markdown body, dark live line, collapse cycle
  back to one line, `overlayAudit ok`, composer carries no compact
  segment in any state.

- **2026-08-30** — Two guards, both closing holes opened earlier today. `picode install`/`deploy` now refuse a binary with no embedded UI: one was deployed by a plain `go build` and the browser got the ADR-0023 "not built yet" page. The check sits in the command layer, not in `install.Deploy`, because `picode update` deploys a *downloaded* release where "does this binary embed the UI" is the wrong question. And the `node_modules` make guard now stamps on `node_modules/.package-lock.json` rather than the directory — an empty directory with a fresh mtime satisfied make and the build then died on `vite: not found`, which is what it did. Both verified in both directions.
- **2026-08-30** — Pi item reappeared in the user menu: **a parallel agent deployed a stale `bin/picode`** (built before `4913a3a5`), not a source regression — main was clean. Fixed by `make build` + deploy from current main. **Rule for every session: before `bin/picode deploy`, run `make build` on a tree at current main** — deploying an old binary silently reverts UI changes that are already merged.

- **2026-08-30** — Removed the **Pi** item from the user menu (owner call): the update surface is the System card only. Also restored the pi-update CHANGELOG entry — it was lost in a conflicted merge earlier.
- **2026-08-30** — **Pi update alert shipped** and proven on a real release: System card with installed → latest, Copy command, and **Update now** (`POST /api/system/pi-update` → `pi update --self`, background ctx so a client disconnect cannot kill the install). Live run updated pi 0.84.3 → 0.84.4 end to end. **Ops note:** deploying with a plain `go build` binary (no `-tags embedui`) installs a disk-mode server that serves "UI has not been built" — always `make build` before `bin/picode deploy` (this bit us once today; fixed by rebuilding embedded).

- **2026-08-30** — ADR-0024: terminal settings. Written after removing tmux's forced `mouse on` broke scrolling in Pi's TUI while leaving Claude Code's alone — Claude Code takes the mouse itself, Pi does not, and one constant cannot serve both. The shape is Windows Terminal's (`profiles.defaults` plus profiles declaring only what they change), extended with user presets. Two storage homes on purpose: tmux behaviour is per session and shared across devices, xterm appearance is per browser and should differ. Decision only; no code.
- **2026-08-30** — **Pi update alert shipped** and proven on a real release: dot on the user-menu **Pi** item (registry check on /api/system, 6 h cache with stale-fallback for hiccups), System card with installed → latest, Copy command, and **Update now** (`POST /api/system/pi-update` → `pi update --self`, background ctx so a client disconnect cannot kill the install). Live run updated pi 0.84.3 → 0.84.4 end to end; dot cleared after. **Ops note:** deploying with a plain `go build` binary (no `-tags embedui`) installs a disk-mode server that serves "UI has not been built" — always `make build` before `bin/picode deploy` (this bit us once today; fixed by rebuilding embedded).

- **2026-08-30** — Memoised the repository lookup in the occupant scan: 200 agents sharing a subfolder went from 4.6s to 22ms, and the cost stopped growing with the agent count. Implementing it uncovered a real bug shipped in ADR-0022 G1 — `gitgraph.Key` resolved git's relative answer (`../.git` one level down) against `--show-toplevel` instead of against the directory asked about, so any cwd below the repo root got a key one level too high. Effect: an agent in a subfolder was silently dropped from the graph. `TestNestedRepoIsNotAnOccupant` had been passing for the wrong reason and now passes for the right one.
- **2026-08-30** — ADR-0022's two unmeasured costs, measured. **Commit ceiling: there isn't one** within what the product allows — layout is 14ms for 10,000 commits, the server answers `?limit=2000` in 0.12s (408KB), and the browser holds 2,000 rows / 17k DOM nodes with a row click at 0.4ms and scrolling at 0.1ms. The 250 default is conservative, not a limit. **Occupant scan has a cliff**: free when agents sit at worktree roots, ~23ms per agent whose cwd is below one — see debts. Also worth recording: mid-measurement I nearly filed 'Load earlier is broken' as a bug. Instrumenting the button showed the clicks were never reaching it — `agent-browser click "text=…"` does not hit it, while dispatching on the element does. The feature works.
- **2026-08-30** — `picode install` / `deploy` survive a non-login shell. `systemctl --user` needs `XDG_RUNTIME_DIR` and `DBUS_SESSION_BUS_ADDRESS`, which a script, cron job or agent shell does not have; both commands copied the binary *before* calling it, so the failure left the new binary on disk with the old one running — hit exactly that during today's deploy, and only a hash comparison showed it. `install.Run` now fills the two variables from `/run/user/<own uid>` when the socket is there, and `EnsureUserSession` refuses before copying when it is not, naming `loginctl enable-linger`. Verified the injected values turn `systemctl --user is-system-running` from a bus error into `running`, and that the guard is what prevents the half-update: without it the test finds the installed binary replaced anyway.
- **2026-08-30** — Frontend tests run in CI. 197 of them were passing where nothing watched. The blocker was ordering — `npm test` needed `node_modules`, which only `make build` installed — so installing moved into its own target gated on `web/package-lock.json`, and `web` and the new `test-js` both depend on it. That also removes a second full `npm ci` per `make ci`. Verified the gate can actually fail: a deliberately broken test takes `make ci` to exit 2, and the guard skips the install on a second run (10s → 2.7s) but reruns it when the lockfile is touched.


- **2026-08-30** — `POST /api/workspaces/{id}/agents` accepts `workPath`; it was hardcoded empty, so ADR-0022's centrepiece — two agents in sibling worktrees of one repo — could only be built from free agents. Reuses `resolveAgentWorkDir`, the same resolver free agents use, so the two creation paths cannot drift; blank stays blank and keeps the agent on the workspace folder. Verified the test has teeth: with the old hardcode both agents pile onto `main`. **API only — no UI was added**, since nothing asked for one.
- **2026-08-30** — Clipboard validated in a browser, closing the ADR-0023-era debt. Text emitted as OSC 52 from inside a tmux pane came back out of the *system* clipboard via a real Ctrl+V — the whole chain, not an inference. Chrome refuses the write without a recent user gesture and accepts it right after a click; the handler's toast covers the refusal. That refusal path needed a synthetic probe to reach at all, so it is not being designed around. Firefox is still unchecked: the automation here only drives Chrome.
- **2026-08-30** — `make fmt` and `make fmt-check` stop reaching into `.worktrees/`. Both walked the tree with `.`, so a sibling agent's uncommitted code failed this gate — and `fmt` was worse than reported: `gofmt -w .` would have *rewritten* their files. Both now walk the directories `go list ./...` reports, the same module boundary `vet` and `test` always respected. The `git ls-files` fix I had written in this file was wrong and is dropped: tested, it misses a new file not yet `git add`ed, so it would pass locally and fail in CI. `go list` catches it, and covers 204 of 204 `.go` files in the module including `//go:build ignore` ones. CI's inline gofmt step now calls the same target instead of repeating the command.
- **2026-08-30** — ADR-0023 implemented. `internal/web` splits into a disk loader (default) and an embedded one (`-tags embedui`), which is what `make build`, `ci.yml` and `release.yml` now use; `internal/web/public/` is untracked and gitignored. The ADR's gating question is answered: over https the service worker registers, activates and fills `picode-assets-v1` with the hashed assets in disk mode, and both modes serve byte-identical asset URLs with identical Cache-Control (`immutable` for `/assets/`, `no-cache` elsewhere). The earlier `swRegs: 0` was a red herring — `main.jsx:16` only registers over https, so it was 0 in both modes. A disk build with no UI answers 503 with `run make web` instead of 404s, and starts serving the moment the files appear.
- **2026-08-30** — ADR-0023: the built UI stops being committed. It had been tracked since the bootstrap commit with no decision behind it — 330 files, 33 MB, 77% of the repo's commits, 133 files rewritten per UI change, and 335/334 `rename/rename` conflicts in two merges that day, every one resolved by rebuilding. Checked six peers (Grafana, Prometheus, Vault, Coder, Gitea, Syncthing): none commits it. Prometheus also answers the constraint that kept us — `//go:build !builtinassets` serves from disk and the default build does not embed at all. Decision recorded; the code change is not made yet.
- **2026-08-30** — User menu gained a **Sessions** item (between System and Providers) that opens the machine-wide `#/sessions` view; `go("sessions")` short-circuits the `/sessions/:id` template. Live click-through verified: menu → 36 folders · 99 sessions · audit ok.
- **2026-08-30** — Terminals stopped wearing tmux's skin. PiCode forced tmux's own `mouse on` since before `termWheel.js` grew its SGR fallback; its only remaining effect was tmux copy-mode on every drag, which is why text could not be selected. Owner tested `mouse off` on the real machine — the wheel still scrolls — so both call sites drop it. Paired with `allow-passthrough on` and a write-only OSC 52 handler so a copy made inside the pane reaches the system clipboard. Proved A/B in the browser: with passthrough on the handler fires, with it off the sequence never arrives. The read form (`52;c;?`) is refused on purpose — `@xterm/addon-clipboard` implements it, which would let any agent in a pane read the user's clipboard. Benchmark: `docs/benchmarks/2026-08-30-web-terminal-clipboard.md`.
- **2026-08-30** — **All-folders Sessions view** (`#/sessions`): every Pi session on the machine grouped by folder (workspaces first; "not a workspace" badges; 36 folders / 99 sessions / ~554 MB on the QA machine). Same actions per row — Open with… only where a workspace owns the folder (disabled + reason otherwise), Delete validated against the sessions root, Compact for in-use. New endpoints `GET/DELETE /api/sessions/all`; `ListAll` summaries now carry size. QA live: fixture in a non-workspace folder deleted end-to-end, scope link both ways, audit ok after height fix (19→36 px on the scope link).
- **2026-08-30** — **Sessions view shipped (A+B)**: `#/sessions/<id>` lists every Pi session under the folder (size/age/msgs/cost/provider, `inUseBy`) with Open with… (resume, no copy), Compact (in-use), Delete (orphans, confirm; in-use 409 with reason) — plus **auto-clean orphans** (Off/30/60/90 d; boot+daily+on-change sweep, default Off). New: `session.ListDir` (+Size in Summary, missing dir = empty), manage endpoints, `StartSessionSweep`. QA live on the 629 MB workspace: fixture deleted end-to-end, in-use blocked with tooltip, dialog lists workspace agents, empty state after the missing-dir fix, audit ok. Known debt: list rescans all JSONLs (same cost as the adopt picker) — needs a stat-only mode if it ever feels slow.
- **2026-08-30** — `web/node_modules` was tracked as a symlink to an absolute local path (added by accident in `971cc632` via `git add -A`); on any other clone it dangles. Untracked it and dropped the trailing slash from both `.gitignore` files — verified in a scratch repo that `node_modules/` leaves a symlink of that name untracked while `node_modules` ignores it, which is exactly how it got in.
- **2026-08-30** — Git graph G3 (ADR-0022): clicking a commit opens its diff. `git show -m --first-parent` is a correctness fix, not a preference — without it a merge arrives as a combined diff (`diff --cc`, `@@@`) that the unified-diff reader misreads silently; proved by removing the flags and watching the test go red. The hash is the only user-controlled part of the git command line, so anything but 40/64 hex is refused before it reaches git. `DiffLine` moved out of Conversation.jsx so the chat and the graph render diffs the same way. Screenshot caught the selected row scrolling out of sight when the pane opened.
- **2026-08-30** — Git graph G1+G2 (ADR-0022). `internal/gitgraph` reads the DAG, refs and worktrees; the column allocator is ported from mhutchie's Git Graph (MIT, attributed in the file header) minus its uncommitted-changes row. Two parser bugs caught by tests with teeth: git hands back a literal 0x1f typed into a message, which a plain Split turned into a *dropped commit*, and a 0x1e split a record into a phantom commit whose hash was someone's subject. Verified on the real repo: 250 rows, `overlayAudit ok`, no h-scroll, 26px rows, dark and light both read, and the occupant chips show `default` on main beside `graph-impl` on its worktree.
- **2026-08-30** — Green bar under the agent TUI killed: agent tmux sessions were the only ones without `status off` (tmux's default status line renders green). Set at `NewSession` and on every bridge attach — old sessions heal on next view (verified live: picode2 pane went 45→46 rows). Scrollbar note: each attach is a `tmux attach` in a PTY, so tmux owns the scrollback — the native xterm scrollbar never fills in either surface; the wheel scrolls tmux history (mouse on at attach). That is inherent to attaching any terminal to tmux, not a PiCode bug.
- **2026-08-30** — Agent TUI and PiCode terminals share **one surface**: the desktop agent terminal view now renders through TermSurface/ShellTerm (same xterm options, wheel/keys/links wiring, padding 0, custom scrollbar, screen margin) instead of the TerminalDock wrapper — computed-style diff between the two is now identical. Managed mode shows a one-line hint + Open TUI action. `closeTerm` moved to `lib/terms.js` (mobile still uses the dock component). Also recovered `web/node_modules` in the primary checkout after a parallel session replaced it with a self-referential symlink.
- **2026-08-30** — Compaction residue fixed at the root: the transcript window now cuts at the **last compaction boundary** (what pi replays), so reload no longer resurrects pre-compaction history. The summary renders as a collapsible **Session compacted** card (39 K chars confined to 253 px + scroll) instead of a giant assistant message. `compacted` in the API response separates "needs /compact" from "already compacted, file stays large" (cold boots stay slow — pi#8843).
- **2026-08-30** — /compact progress moved out of the conversation into the **composer statusbar**: "⠋ Compacting 1:23" segment with spinner + 1 s ticker, persisted per agent in localStorage (survives the TUI→managed panel rebuild and page reloads), cleared by the HTTP answer or `compaction_end`. Verified live on the 140 MB adopted session (78 events post-boundary vs 9 673 raw).
- **2026-08-30** — Sidebar spinner covers TUI work too: GET /api/tui-working polls tmux capture-pane for pi's Working state (3 s).
- **2026-08-30** — /compact is visible end to end: "Compacting session…" line in the thread, closes on HTTP answer or pi's compaction_end, "too small" maps to "Nothing left to compact."
- **2026-08-30** — Terminal/chat view persists per agent across reload (localStorage `picode-term-view`).
- **2026-08-30** — Transcript API paginated (`?tail=&skip=`): opening the 129.5 MB session loads a 200-event slice (~1 s server, small payload); Load earlier fetches older turns from the server with scroll anchoring.
- **2026-08-30** — Huge sessions render a window (~60 turns) with Load earlier on scroll; composer height tracks local text. Proven on a 122 MB session (226 turns → 60 mounted).
- **2026-08-29** — ADR-0022: the git graph belongs to a repository, not to an agent. Studied VS Code's Git Graph first (it is mhutchie's extension, not VS Code): one runtime dependency, `iconv-lite`, no framework and no charting library — 913 lines of DOM + SVG, of which 73 place the columns. Measured that worktrees share refs and differ only in HEAD, which is why one graph with N marked heads beats N near-identical graphs. Read-only in this ADR: no write path exists that could hit a worktree while an agent is mid-turn.
- **2026-08-29** — ADR-0021 adopt Pi session by copy. GET/POST `/api/pi-sessions`. New agent → From a Pi session.
- **2026-08-29** — Agents work in an isolated git worktree (AGENTS.md #5). After merge, remove the tree and the branch.
- **2026-08-29** — Keepalive debt paid: the tray's `wsl.exe` child now lives in a job object with `KILL_ON_JOB_CLOSE`, so the kernel tears it down however the tray dies — `taskkill /F` and crashes included, where no deferred Go code runs. Established first that killing the Windows-side `wsl.exe` does end the Linux process (launched a `sleep 99999`, killed its wsl.exe, watched it go), because the whole fix rests on that. Then proved it end to end on the machine: before, a force-killed tray left `sleep infinity` behind; after, `taskkill /F` leaves nothing. `startDetached` became `startSupervised`, and a failure to supervise is non-fatal — a keepalive that can be stranded beats no keepalive.
- **2026-08-29** — Tray icon is now the browser favicon (`web/public/favicon.svg`): the blocky Pi in white on `#09090b`, transcribed as rectangles in `mkicon.go` and drawn at 4x then boxed down so 16px stays legible. `icon_test.go` reads the SVG's fills and asserts the committed ICO carries both — verified it actually fails by regenerating with the old indigo. Restarted the tray on the owner's machine with the new mark (PID 29532); doctor still reports everything ok on both halves.
- **2026-08-29** — **PiCode Desktop is live on the owner's machine.** Ran `picode-desktop.exe` through WSL interop: `doctor` showed the mkcert CA already trusted from an earlier `setup-cert.sh`, leaving only the logon task; `install` elevated (one UAC) and registered `PiCodeDesktop` — `At logon time`, `Run As User: cfpp`, `"…\picode-desktop.exe" --tray`. The tray runs (PID 30320) with its `sleep infinity` keepalive holding the distro open. Every step now reports ok on both halves. Fixed a fourth bug the run exposed: `doctor` summarised only the distro, so a fully set-up machine was still told to run install; `printWindowsSteps` now reports whether its half is finished and the summary weighs both.
- **2026-08-29** — Ran the real provision on the owner's machine (Linux half). One change: `Linger=no` → `yes`. `/etc/wsl.conf` came back with the **same md5** and no `.picode.bak`; PID 447165 and all three tmux sessions untouched, because the unit was already enabled and running so nothing restarted. The root pass then exposed a third bug: `systemctl --user` always answers for the *calling* account, so from root it reported goat's running service as "present but not enabled" — confident and wrong. `Env.Acting` now records who the process actually is, `OnBehalf()` compares it to the target, and the service check returns **blocked** instead of guessing. The summary line also stopped counting a skip as "would change". Windows half (logon task, CA trust, keepalive) still not run — it needs the `.exe` on Windows plus one UAC click.
- **2026-08-29** — First `picode-desktop doctor` run found two real bugs no test had. (1) The root pass invoked a bare `picode`, which is on **no** PATH but the owner's — ADR-0018 installs to `~/.local/bin`, reachable only through that account's profile. `PicodePath` now resolves it once via the owner's **login** shell (`sh -lc`) and hands both passes the absolute path. (2) Far worse: `cmd/picode` fell through to `serve()` for any unrecognised argument, so the *old* installed picode, asked to `provision`, silently started a second server **as root** in `/root/.picode` on port 8446. Killed it and removed the directory; the owner's picode on 8445 was untouched. `dispatch` now owns the command list and a bare word it does not know exits 2.
- **2026-08-29** — Desktop **M5**: release plumbing (ADR-0020, plan complete). Tag-triggered `release.yml` builds picode for 5 platforms plus `picode-desktop-windows-amd64.exe` into one release; `version.Version` became a var so `-X` can stamp it (verified: a stamped build reports 9.9.9). `install.LatestReleaseFor` extracted so both binaries share one update check. A test reads the workflow and fails if the asset names drift from what `update` looks for — the kind of break that is silent forever. Tray icon: a pi generated by `mkicon.go` (`//go:build ignore`, `go:generate`) at six sizes, BMP payloads; the first draft merged its legs into a "T" at 16px, so the geometry was fixed against a rendered preview. Elevation is runtime `ShellExecuteW`/`runas`, **not** a manifest — the same binary is the tray, and `requireAdministrator` would elevate that too.
- **2026-08-29** — Desktop **M4**: the clean-machine path (ADR-0020). A stage machine derived from observed state, not saved progress, so an interrupted install resumes and a finished one is a no-op: install WSL `--no-distribution` → restart → install the distro `--no-launch` → create the account → provision. Both flags dodge the interactive account setup plain `wsl --install` opens. Exit **3010** is "succeeded, restart pending", not a failure; `RunOnce` resumes at the next logon and deletes itself. The Linux account is named after the Windows one (accent-folded, sanitised) with the password left **locked** — PiCode reaches root via `wsl -u root` and never needs sudo. Writing the 3010 test found that POSIX shells truncate exit status to 8 bits (3010 arrives as 194) while Windows does not, so the rule is tested through an interface rather than a real process. Live-WSL test confirms this machine is classified as ready and skips every stage.
- **2026-08-29** — Desktop **M3**: `cmd/picode-desktop` + `internal/desktop` (ADR-0020). Drives the distro with two `picode provision --json` passes (root, then owner) and merges them so whichever pass resolved a step wins — the rule that keeps "skipped for lack of privilege" from masking "fixed". Windows side: `onlogon` task at `/rl limited` (an elevated tray cannot reach the notification area), mkcert CA import gated on a count so logon does not re-import, `sleep infinity` keepalive against the idle timeout, `CREATE_NO_WINDOW` on every child. `wsl.exe` output is UTF-16LE **with no BOM** — decoding is decided by inspecting bytes, and the real 136-byte output is a base64 fixture. `make desktop` cross-compiles from WSL: 6.5 MB PE32+ GUI, `CGO_ENABLED=0`, no C compiler. Live-WSL tests (skip in CI) confirm it picks `Ubuntu` (WSL 2) and reads the account `goat`. New dep: `fyne.io/systray` (pure Go on Windows). **Not executed against the machine** — no `install` run.
- **2026-08-29** — Desktop **M2**: `picode provision` (ADR-0020) converges six steps — wsl.conf, systemd, linger, cert, unit, health — with `--dry-run` and `--json`, and root vs user scopes so the Windows side can drive it in two calls. `EnsureKey` merges `/etc/wsl.conf` by line: the owner's real file (comment, key order, `generateResolvConf = false` spacing) is a test fixture asserted byte-identical, and the fix writes no backup when nothing changed. Writing the `Run` decision table caught a real bug: blocked steps were reported as "planned" in a dry run, promising a fix no run could deliver. Extracted `tlsutil.LocalNames` so the self-signed and mkcert paths issue for the same hosts. Dry run on the owner's machine: 4 ok, 2 to fix (linger, unit) — matching the plan, with `/etc/wsl.conf` verified unchanged (same md5, no `.picode.bak`).
- **2026-08-29** — Track 3 live cwd: Ctrl+click asks tmux `#{pane_current_path}`. File-preview roadmap closed. Tests: PaneCwd + GET `/api/terminals/{id}/cwd` after `cd`.
- **2026-08-29** — Chat file cards sit under the turn file names (one per click). visual-review: PASS (file-chat-cards-inline.png).
- **2026-08-29** — Terminal: Shift+drag select, Ctrl+C copy if selected, Ctrl+V paste. visual-review: PASS (pref-term-copy.png).
- **2026-08-29** — Sidebar Terminals list flush like Pins. visual-review: PASS (terms-flush.png).
- **2026-08-29** — **ADR-0020** accepted: PiCode Desktop provisions the distro from Windows (tray `.exe` at the WSL boundary, `picode provision` inside). Supersedes ADR-0018, whose "Alternatives considered" had rejected both the logon task and linger; the linger objection is answered by enabling it on install and never disabling it. Carries a preservation contract (`~/.picode` untouched, tmux survives, `wsl.conf` line-merged after backup, cert reissued only near expiry). Docs only — no code, no user-visible change, so no CHANGELOG entry.
- **2026-08-29** — Chat file card: click a path → closable card in the thread; Open in tab → same `#/file/a/…` as the terminal. Split FilePane removed. visual-review: PASS (file-chat-card.png, file-chat-tab.png).
- **2026-08-29** — File preview track 1: png, pdf, md, audio, video, glb/gltf (model-viewer). visual-review: PASS (file-png-preview.png, file-md-preview.png, file-pdf-preview.png, file-audio-preview.png, file-video-preview.png, file-glb-preview.png, file-bin-raw.png, file-bin-gone.png).
- **2026-08-29** — Ctrl+click on a terminal path eats mousedown/mouseup so tmux SGR (`<16;NaN;NaNm`) does not land in the Pi composer.
- **2026-08-29** — Terminal Ctrl+click also matches bare `App.jsx` / `foo.js` (not only paths with `/`).

## Recent activity (archived 2026-08-29)

- **2026-08-29** — File V1.1: `.svg` / `.mmd` open Preview \| Raw \| Save (one chip group). Empty: “Nothing to preview.” Bad mermaid: “Can't draw this diagram.” visual-review: PASS (file-svg-preview.png, file-svg-raw.png, file-mmd-preview.png, file-mmd-empty.png, file-mmd-error.png).
- **2026-08-29** — ADR-0019: Ctrl/Cmd+click a path in the terminal opens `#/file/…` on the tab strip (text only). http(s) → browser. visual-review: PASS (file-tab.png, file-tab-gone.png, file-tab-outside.png). Chat FilePane unchanged.
- **2026-08-29** — Reconnect covers fast restarts: health `bootId` + WS-close kick → tab reloads itself (proved: restart → page age reset). Shift+Enter trusted-key E2E: bytes `[27;2;13~` only, multiline composer confirmed; earlier "submit" reading was a bad pane-tail interpretation. keypress guard added for browsers that fire it after a canceled keydown.
- **2026-08-29** — Shift+Enter three-layer fix: ground truth via session JSONL proved the user's Windows Chrome submits (a stray `\r` after the canceled keydown). `termDataFilter` on `term.onData` swaps/drops that `\r` (120ms window). Keydown tracker lives on the xterm textarea (capture).
- **2026-08-29** — Terminal Ctrl+C copies if text is selected (Warp / Windows Terminal); else interrupt. Keys list in Preferences → Terminal; `/hotkeys` shows them.
- **2026-08-29** — Settings **Keys**: search / Add (press a key) / Reset writes `~/.pi/agent/keybindings.json`. visual-review: PASS (settings-keys.png, settings-keys-empty.png, settings-keys-listen.png).
- **2026-08-29** — Terminal scroll: tmux `mouse on` so xterm.js enables mouse tracking (#426); screen `margin-right` so the scrollbar is clickable (#1751); stop capturing wheel (that blocked SGR). visual-review: PASS (term-scrollbar.png, `enable-mouse-events` true).
- **2026-08-29** — Tabs flush (no left pad); drag to reorder. Terminal: debounce resize 150ms; wheel sends SGR to TUI transcript; Shift+Enter newline (Preferences → Terminal). visual-review: PASS (tabs-reorder.png, first tab gap 0).
- **2026-08-29** — Terminal: ResizeObserver fits the pane as soon as the sidebar/split changes (not only window.resize). Wheel capture: scroll xterm, else PageUp/PageDown to the TUI. visual-review: PASS (term-resize.png; screen 1000→1168px when sidebar 244→80).
- **2026-08-29** — Terminal wheel on a TUI (Pi) pages/scrolls the view instead of the composer cursor. Stopped stretching `.xterm-screen` to 100% (broke hit-testing + scrollbar). visual-review: PASS (term-wheel-scrolled.png).
- **2026-08-29** — Composer chips joined into button groups (session+New; provider/model/thinking/mode/kind). visual-review: PASS (composer-chip-groups.png, composer-chip-groups-run.png).
- **2026-08-29** — Machine menu: Theme/Layout are one left-aligned capsule (radio semantics); entries have icons. Fixed segment svg shrinking to 0 (`flex:none` + `flex:auto`). visual-review: PASS (user-menu.png).
- **2026-08-29** — Composer Expand + More menu; sketch Cancel/Insert 28px. visual-review: PASS (composer-pages.png, composer-more.png, composer-sketch.png).
- **2026-08-29** — Preferences → Terminal is two columns (controls + live xterm preview). visual-review: PASS (pref-terminal.png).
- **2026-08-29** — `make deploy` / `picode deploy` = repo → systemd. `picode update` = GitHub release for a normal user.
- **2026-08-29** — `picode install` / `uninstall` (systemd --user, ADR-0018). No Windows task.
- **2026-08-28** — Rename terminals (pencil / double-click). visual-review: PASS (term-rename.png, term-renamed.png). Terminal Light/Dark in Preferences (pref-term-theme.png), independent of the app theme.
- **2026-08-28** — File pane header compact (24px); Save, Close, Expand last. visual-review: PASS (file-header.png, file-expanded.png, overlayAudit ok).
- **2026-08-28** — File pane left edge resizes (persists). visual-review: PASS (file-resize.png).
- **2026-08-28** — File pane syntax colors follow GUI light/dark. visual-review: PASS (file-theme-light.png, file-theme-dark.png, overlayAudit ok).
- **2026-08-28** — Track E4 turn file names. visual-review: PASS (turn-files.png, turn-files-open.png, overlayAudit ok).
- **2026-08-28** — Track E3 Keep/Undo on edit diffs. visual-review: PASS (hunk-keep.png, hunk-kept.png, hunk-undo.png, overlayAudit ok).
- **2026-08-28** — Track E2 CodeMirror Save. visual-review: PASS (file-edit.png, file-discard.png, overlayAudit ok).
- **2026-08-27** — Track E1 open file beside chat. visual-review: PASS (file-open.png, file-gone.png, file-binary.png, overlayAudit ok).
- **2026-08-27** — ADR-0015 + Track E plan (file pane, Save, hunk Keep/Undo). Herdr study: runtime peer, not the editor bar.
- **2026-08-27** — Voice V1 owner dogfood: Chrome Windows mic works. No V1.1 unless a quality gap shows up.
- **2026-08-27** — Track D5 package updates (badge + Update). npm only. visual-review: PASS (pkg-update.png, pkg-update-menu.png, overlayAudit ok).
- **2026-08-27** — Track D4 Prompts timeline (Now + continue in a new session). visual-review: PASS (prompts-tree.png, prompts-continue.png, overlayAudit ok).
- **2026-08-27** — Track D3 `@` mentions (agent / skill / file). visual-review: PASS (at-mention.png, at-mention-empty.png, overlayAudit ok).
- **2026-08-27** — Track D2 cost on the session chip. visual-review: PASS (session-cost.png, overlayAudit ok).
- **2026-08-27** — D1 loop: hash apply no longer depends on selectedId (tabs vs URL freeze). visual-review: PASS (agent-url-switch.png, overlayAudit ok).
- **2026-08-27** — Track D1 `#/agent/<id>`. visual-review: PASS (agent-url.png, agent-url-gone.png, overlayAudit ok).
- **2026-08-27** — Plan: `@agent` / `@skill` are mentions (context in this prompt). Agents talking to each other is the broker item, later.
- **2026-08-27** — Track C2 draft persistence (text + kind per agent). visual-review: PASS (chat-draft-reload.png, overlayAudit ok).
- **2026-08-27** — Follow-up queue: Edit / Remove on the bubble (held until idle). visual-review: PASS (chat-queue-edit.png, overlayAudit ok).
- **2026-08-27** — Track C3 visible queue: Send while busy/waiting; prompt→follow-up; abort drops Steer. visual-review: PASS (chat-queued.png, overlayAudit ok).
- **2026-08-27** — Track C1 waiting: extension dialogs in the conversation (`POST /api/agents/{id}/ui`). Notify is a toast. visual-review: PASS (chat-waiting.png, chat-ask-cancel.png, overlayAudit ok).
- **2026-08-27** — Track C roadmap: waiting → queue → draft (`docs/design/conversation-control-roadmap.md`). Next-roadmap gaps recorded there (rewind, cost, `#/agent/<id>`, extra `@`, hunk accept, broker, ACP, IDE). Backlog unchanged.
- **2026-08-27** — Cleared MCP dogfood: `picode-dogfood-*` out of Claude/Codex/Grok; Notion/Linear tokens out of the keyring; `~/.picode/mcp-auth` job files gone. Claude keeps `context7`.
- **2026-08-27** — MCP layout: Servers (list or “No servers yet.”) then Add with **Use from…** as first chip. Dropped “Using Claude Code”. visual-review: PASS (mcp-named.png, mcp-use-from.png, overlayAudit ok).
- **2026-08-27** — Signed-in rows do not also say “Signed in”; **Sign out** is the state. visual-review: PASS (mcp-signout.png).
- **2026-08-27** — MCP **Sign out** forgets the keyring login (Off does not). Linux secret-tool. visual-review: PASS (mcp-signout.png, mcp-signout-confirm.png, overlayAudit ok).
- **2026-08-27** — After Sign in the toast fired but the row still said Sign in: overlay ends on callback before keyring write, and Idle list does not poll without a running agent. Optimistic **Signed in** + retry load. visual-review: PASS (mcp-linear-signed.png).
- **2026-08-27** — Linear Sign in was sending `redirect_uri=https://mcp.linear.app/callback` (Linear homepage, overlay hung). Now we register a localhost callback. Authorize URL verified `127.0.0.1` + refuse non-loopback.
- **2026-08-27** — Add GitHub still saves the row; Copilot DCR failure is a toast, not a failed Add. Dogfood Linear in Claude (`picode-dogfood-linear`). visual-review: PASS (mcp-linear-list.png).
- **2026-08-27** — MCP overlay ends on callback (not after ~40s keyring write). Signed-in OAuth rows say **Signed in** and hide Sign in. visual-review: PASS (mcp-signed-in.png). Signed-in detection is Linux secret-tool (WSL); macOS/Windows later.
- **2026-08-27** — MCP Sign in: GUI opens the authorize tab (Pi/PowerShell does not) so the PiCode callback can `window.close()` like Claude/Codex. visual-review: PASS (mcp-signin-auto.png, overlayAudit ok).
- **2026-08-27** — MCP callback page is the same PiCode success HTML as providers (close + return to `#/mcps`), not the adapter's "return to pi".
- **2026-08-27** — MCP Sign in no longer uses `/mcp-auth` paste UI. Headless `authenticate()` (callback only) writes a result file; overlay ends when that file is ok. visual-review: PASS (mcp-signin-auto.png).
- **2026-08-27** — MCP Sign in opened two Notion tabs (GUI window.open + Pi open on WSL). GUI no longer opens a tab. visual-review: PASS (mcp-signin-auto.png, overlay unchanged).

## Recent activity (archived 2026-08-27)

- **2026-08-26** — MCP Sign in is automatic (no paste): GUI opens the tab, adapter callback auto-closes it, overlay ends on success notify. visual-review: PASS (mcp-signin-auto.png).
- **2026-08-26** — MCP Sign in overlay stayed up after Notion Authorization Successful (callback did not unblock `/mcp-auth` UI). Now notify success finishes the wait; Paste is always there.
- **2026-08-26** — MCP Sign in opened two Notion tabs (GUI window.open + Pi open()). GUI no longer opens a second. visual-review: PASS (mcp-signin-wait.png, overlay unchanged).
- **2026-08-26** — MCP Sign in waits for the browser callback (no paste by default). Paste address is fallback. visual-review: PASS (mcp-signin-wait.png).
- **2026-08-26** — MCP Sign in uses a short pi when no agent is running. Add/On on OAuth starts Sign in. visual-review: PASS (mcp-signin-short.png). Dogfood: Notion login opened with no agent.
- **2026-08-26** — Sign in is a button next to On (not a SIGN-IN tag). Off has no Sign in. visual-review: PASS (mcp-signin-btn.png).
- **2026-08-26** — MCP Sign in starts `/mcp-auth` (RPC + paste dialog). visual-review: PASS (mcp-signin.png).
- **2026-08-26** — MCP GET redacts env/header values. Dogfood in Codex/Grok left for later.
- **2026-08-26** — Diff cards in conversation use JetBrains Mono + Fira Code (same as source fences). visual-review: PASS (chat-diff-font.png).
- **2026-08-26** — Remove on a Use-from overlay no longer unmasks the import (stays Off). Dogfood Claude servers deleted. List is A–Z. visual-review: PASS (mcp-list-az.png).
- **2026-08-26** — MCP list is A–Z by name (live poll no longer reshuffles).
- **2026-08-26** — Conversation source uses JetBrains Mono + Fira Code ligatures. visual-review: PASS (chat-code-font.png).
- **2026-08-26** — MCP live status (Idle / Live / Failed / Sign in). visual-review: PASS (mcp-live-idle.png).
- **2026-08-26** — MCP Add More: env / headers / Sign in / Token. visual-review: PASS (mcp-add-more-url.png, mcp-add-more-env.png, mcp-add-more-error.png).
- **2026-08-26** — MCP card: agent icon + name at top; scope pill is This agent again. visual-review: PASS (mcp-this-agent.png).
- **2026-08-26** — Use from is a tree (app → servers). Pick per server; Off the rest. visual-review: PASS (mcp-use-from-tree.png).
- **2026-08-26** — Dogfood MCP in Claude/Codex/Grok globals (`picode-dogfood-*`). Use from lists counts. visual-review: PASS (mcp-use-from-dogfood.png).
- **2026-08-26** — Import renamed **Use from…** (mirror, not copy). Empty hosts hidden. visual-review: PASS (mcp-use-from.png).
- **2026-08-26** — MCP Import is a picker, not import-all. visual-review: PASS (mcp-import-pick.png).
- **2026-08-26** — MCP B3 Import (adapter `imports` only). visual-review: PASS (mcp-import.png).
- **2026-08-26** — Agent context is the first line in the MCP/Packages card, not under the title. visual-review: PASS (mcp-card-ctx.png).
- **2026-08-26** — MCP/Packages name the agent (title + pills). Sidebar click from a pane opens that agent. visual-review: PASS (mcp-named.png).
- **2026-08-26** — MCP empty redesigned (one line + Open packages). UI skills now load-before-JSX; visual skip = quality-gate FAIL. visual-review: PASS (mcp-blocked.png).
- **2026-08-26** — MCP manager: list/add/toggle/remove on adapter files (machine / folder / this agent). B3 import next.
- **2026-08-26** — Composer `!cmd`: RPC bash in the agent folder, inline block + Stop. Track A done.
- **2026-08-26** — Toolbar clip attaches workspace files (image → chip, else `@path`); reads stay inside the folder.
- **2026-08-26** — Click a composer/chat thumbnail to preview the image.
- **2026-08-26** — Composer image chip 64px; `@` list has a filter and hides dotfiles until typed.
- **2026-08-26** — Composer paste/drop images (RPC `images[]`). Next: `!`.
- **2026-08-26** — Composer `@` file picker (agent cwd). Next: images, then `!`.
- **2026-08-26** — Roadmap: composer files then MCP (`docs/design/composer-mcp-roadmap.md`). Auth/llama parked.
- **2026-08-26** — Restore walks the same job overlay (stop agents → db → pins → sessions) and asks to reload.
- **2026-08-26** — Reveal uses host Explorer on WSL. Backup job steps animate. Motion + optimistic UI is a gate.
- **2026-08-26** — Backup schedule is explicit (off until Schedule). Preferences split into tabs.
- **2026-08-26** — Folder picker on WSL: Home / C: / E: chips; accepts `C:\\` paths.
- **2026-08-26** — Backup V1: Preferences folder + interval/retention. `VACUUM INTO` + hardlink snapshots. Restore refuses newer schema.
- **2026-08-26** — Decision table is a quality gate when conditions change the outcome (AGENTS.md).
- **2026-08-26** — Delete agent/workspace: confirm may offer session + work-folder purge (last occupant only). All workspace agents stopped first.
- **2026-08-26** — Pin V3 sketches (Excalidraw, lazy). Blank or annotate image.
- **2026-08-26** — Pin V2.1 TipTap editor (markdown on disk).
- **2026-08-26** — Pin attachments V2 (image + file). Sketch/Excalidraw is V3.
- **2026-08-26** — Pin studio is a route (`#/pins/new` / `#/pins/:id`). List stays in the sidebar.
- **2026-08-26** — `npm:pi-agent-browser-native` + skill shrunk. Fix: IconPin crash (blank app).
- **2026-08-26** — Pins V1: title, tags, markdown body. Flat list, `+` on title bar.
- **2026-08-26** — Sidebar tabs Agents / Pins. QR → user menu.
- **2026-08-26** — Conversation polish: blockquotes + ```diff``` hunks. Images + Mermaid + KaTeX + tables.
- **2026-08-26** — Source **Run** (bash/python/js/go) in the agent cwd. Not a browser sandbox.
- **2026-08-26** — Conversation source renderer (fenced code: lang + copy + highlight).
- **2026-08-26** — Codex DID reply; chat ignored `message_end` (no `text_delta`). Free-agent Sessions listed the wrong folder (0).
- **2026-08-25** — Chose `npm:pi-web-search` (this machine). Chat search cards from tool sources. Full packages-cycle dogfood deferred.
- **2026-08-25** — Packages **This agent** + optional isolate (skip machine/folder). `pi -e` every start / every session.
- **2026-08-25** — llama GUI installer reverted. Setup stays in `www/guide/llama.md`; dialog is URL + link. Continuity → backlog.
- **2026-08-25** — `/llama` dialog on the agent (not Providers redirect). HF download, wait-for-load, default `127.0.0.1:8080`.
- **2026-08-25** — Slash TUI 24 all **ui**. Skills/templates picker. `/export` `/import` `/share` (gist) `/hotkeys` `/changelog`.
- **2026-08-25** — ADR-0013 multi-account vault. OAuth re-login updates same account; click name to rename.
- **2026-08-25** — Device-code OAuth: Copilot / Kimi / xAI. Claude + Codex stay loopback.
- **2026-08-25** — SW never caches `index.html`. Sidebar tree: 12px indent, shared chevron|icon|label grid.
- **2026-08-25** — Providers GUI: no docs copy (guide is public). Voice V1 shipped; owner dogfooding.
- **2026-08-25** — Relicensed PolyForm Noncommercial. Public docs VitePress on Pages (no in-app iframe).

## Recent activity (archived 2026-08-25)

- **2026-08-25** — Public docs: VitePress Markdown, new-tab slash hints
  (`/commands#{id}`). No in-app docs/iframe.
- **2026-08-25** — Public docs (`www/` → GitHub Pages). `#/docs/{cmd}`
  iframes them (later removed). `/tree` click remains fork (pi#8645).
- **2026-08-25** — ADR-0011: sidebar **Free** vs **Workspaces**, many agents
  per folder (own model). Selected entity is the agent id.
- **2026-08-25** — Packages: This machine vs This workspace (`-l`).
  Session/`pi -e` still deferred (This run). ADR-0010 amended.
- **2026-08-25** — Optimistic UI is a bar (`docs/philosophy.md` §7).
  Packages gallery uses layout skeletons on first load; refetch keeps
  last hits. Blank wells while fetching are FAIL.
- **2026-08-25** — Voice V1: dictation + Grok-style voice composer
  (`docs/design/voice-mode.md`). Web Speech API, no Realtime fork.
- **2026-08-24** — Desktop/mobile shells in one Vite app (`web/src/desktop`,
  `web/src/mobile`). Boot picker by viewport or `?desktop=1`/`?mobile=1`.
- **2026-08-24** — Phone QR: prefer current LAN IP. Drawer lists lan/tailnet
  targets; QR only for addresses on the cert.
- **2026-08-24** — Adopted AgentDeck's product-benchmark set: Cursor +
  t3code + paseo. Studies in `docs/benchmarks/`.
- **2026-08-24** — Route split: Settings = PiCode system; `#/providers`
  and `#/mcps` are first pi-facing routes.
- **2026-08-24** — Agent provider/model/thinking moved onto the agent tab
  bar (auto-save).
- **2026-08-24** — **ADR-0009 + M3 v1**: catalog from pi, auth via `/login`
  in the TUI, MCP status-only, agent config flags on start, exclusive lock.
- **2026-08-24** — M2 closed: inline diffs and Ctrl+K palette. Accept/reject
  hunks deferred.
- **2026-08-24** — **ADR-0008**: UI React + Vite + Tailwind. Source in `web/`.
- **2026-08-24** — Dock: `[hidden]{display:none !important}`; single pane
  owned by the active agent tab (no inner tab strip).
- **2026-08-24** — IDE-style agent tabs; dock opens only by explicit action.
- **2026-08-24** — Exploratory QA (agent-browser): real-pi prompt stream,
  port rebind 8445→8446→8445, theme sweep.
- **2026-08-24** — `agent-browser` skill added (agentdeck port).
- **2026-08-24** — User-menu popover SyntaxError; Cache-Control no-cache +
  `cmd/uicheck`. JS-syntax gate mandatory after app.js edits.
- **2026-08-24** — ADR-0007 shipped: HTTPS default, port rebind, server.json.
- **2026-08-24** — UI redesign after owner feedback (conversation-hero,
  tool pills, rounded composer, terminal dock).
- **2026-08-24** — M2 core shipped (ADR-0006): rpc + runtime + delivery,
  mode-switch, /ws/agent, agent panel. Verified against real pi.
- **2026-08-24** — ADR-0005 shipped: SQLite store, schema v1, migrations,
  legacy JSON import.
- **2026-08-23** — UI copy de-documentarized, Vercel-style user menu,
  settings route, live statusbar. M1 visually validated by owner (PASS).
- **2026-08-23** — CI `-race` term-bridge shutdown race; single-owner pty.
- **2026-08-23** — M1 complete: screenshot tooling, tmux, WS↔PTY, terminal
  grid, ADR-0004.
- **2026-08-23** — Language policy: English is the repository language.
