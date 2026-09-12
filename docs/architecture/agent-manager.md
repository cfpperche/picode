# AgentManager

> Part of [PiCode's architecture](../architecture.md) (ADR-0105: one file per subsystem). Edit here; the index only links.

Owns agent lifecycle via the SQLite store (`internal/store`): workspaces
workspaces start empty (ADR-0027) and own zero or more agents; tmux
session names derive from the agent id.
An agent runs as `pi` (ADR-0003, user-installed) in a named tmux session.
Per-agent provider/model/thinking is stored on `agents` and passed as
`pi --provider/--model/--thinking` on start (ADR-0009). Auth stays in
`~/.pi/agent/auth.json`; PiCode never collects keys.
`GET /api/providers/{id}/usage` (ADR-0031) reads the active slot.
`GET /api/providers/{id}/accounts/{aid}/usage` reads that vault row
without swapping `auth.json` (refresh writes the row; `auth.json` only
if it is active). Catalog `quotaKind` on each account tells `#/clis/pi/providers`
when to show Usage (`oauth` or `api_key`). Banked resets (Codex, Grok)
ride `resets[]`; `POST …/usage/reset` redeems one after the UI confirms.
Grok resets also try `~/.grok/auth.json` then `GROK_COOKIE`.
`GET /api/providers/usage` (ADR-0058) answers the roster from the process
cache only — never a vendor — with each row's age, so a page load costs
nothing; `StartUsageRefresh` warms the **active, non-paused** slot of each
meterable provider every 5 minutes, sequentially. A fetch writes back any
identity the vendor volunteered (email, normalised plan) onto the vault row.
`POST /api/providers/{id}/verify` runs `pi auth check --provider <id> --json
--no-refresh`; `POST …/accounts/{aid}/pause` keeps a credential but takes the
row out of play (the active one promotes another; the last live one is
refused). A provider with no `auth.json` entry whose API-key env var is set
is signed in with `source: "environment"` and `envVar` — pi reads it, so the
page says so and offers no Sign out. `/api/catalog` also carries how many
agents and automations name each provider, for the Sign out confirm.

HTTP API (Go 1.22 method patterns):
- `GET /api/health` and `/api/version` — liveness and build identity. The
  version response includes `semver`, display `version`, and `release`, which
  tells the shells whether the bundled What’s New notes may auto-open
  (ADR-0063).
- `GET/POST /api/workspaces` — list (with live `running` flag) / add.
  Add registers the folder only (ADR-0027): the 201 carries `agents: []`
  and no `agent` key; an idempotent re-add answers with the real agents
  and never resurrects a deleted one. `agent` is omitted whenever a
  workspace is empty.
- `POST /api/workspaces/clone` — `{url, name, path}` clones a remote
  repository into a fresh/empty destination and registers it as a
  workspace (ADR-0034). Blocking, 10-minute cap; host credentials, all
  interactive prompts disabled. A destination already cloned from the
  same origin is adopted (`200 {adopted:true}`); occupied by anything
  else → 409. The one git write reachable from the GUI.
- `GET /api/github/repos` — repository metadata for the clone form's
  picker, listed with the machine's own `gh` CLI (ADR-0034's credential
  model — the user's gh, never a PiCode-held token): `gh repo list
  --limit 100 --json …` after an `gh auth status` check, 20 s cap. The
  answer is `{available, reason?, repos?}` — unavailable setups carry a
  visible reason (`gh` missing / not logged in / list failed) instead of
  an error, and successes are cached in memory for 5 minutes; failures
  stay uncached and `?refresh=1` bypasses the cache for Retry. Read-only
  metadata; the clone itself still runs git with host credentials.
- `GET /api/apps` — apps host (ADR-0036): manifests `{id, name, icon,
  apiVersion, surface?}` plus a live `badge` (`count` = actionable, `dot` =
  activity) per app; the poll target for the Apps tab. `surface` is
  omitted for a primitives app and `"native"` for a body compiled into a
  shell (ADR-0109); a native app's view route answers one detail line and
  its action route 400. A badge failure degrades to no badge, never a
  failed list. First-party apps only, assembled in `cmd/picode`; the
  registry seeds the **Inbox** (ADR-0037) and `PICODE_DEMO_APP=1` adds the
  hidden QA apps — Demo (primitives) and Native demo.
- `GET /api/apps/{id}/view?path=…` — one screen of an app as a tree of
  UI primitives (list / detail-markdown / form / actions) the SPA
  renders with host components; `apiVersion` gates rendering on both
  sides. `POST /api/apps/{id}/action` — `{action, path, args}` →
  `{toast?, view?, path?}`. A view may hint `layout:"split"` and tag
  blocks with `pane:"list"|"detail"` (list left, detail right, host-side
  drag-to-resize with the width persisted in `localStorage`; stacked
  under 880px, where the resize handle doesn't render), name its own
  blankslate line in `empty`, and decorate
  rows with `meta`, `at` (RFC3339 — the host formats it, relative in the
  row and absolute on hover), `tone` (`info|ok|warn|danger`), `unread`
  and per-action `icon`. A view may also carry `tabs` (`id`/`label`/
  `path`/`badge?`) for a segmented top-level control (e.g. the Inbox's
  Active/Done/All). Block types stay the frozen four (ADR-0036
  amendments).
- `POST /api/inbox` — file an inbox item (ADR-0037): `{kind:
  fyi|question|approval|result, sourceKind: agent|terminal|system,
  sourceId?, workspaceId?, reason, title, body?, blocking?,
  allowedResponses?, sessionPath?}`. `ask_human` supplies its exact Pi
  session file; generic callers may omit it. Localhost trust model
  (ADR-0007); provenance is mandatory and bodies render as markdown,
  never HTML. Questions and approvals block by nature. `GET
  /api/inbox?state=&blocking=` lists (snoozed hidden until due). `POST
  /api/inbox/{id}/respond` `{verb: accept|edit|respond|ignore, text}`
  answers and marks done. A stopped or managed agent receives the existing
  durable `follow_up` task. A TUI agent instead receives the reply inside its
  terminal (ADR-0060): receiver extension or tmux paste, gated on the item's
  exact captured session, refused before mutation when identity is absent or
  unsafe, another send is pending, or a pane/session mutation holds the guard.
  The reply counts only when the session JSONL gains the full-payload user
  row; failure reopens the same Inbox item with the prior response retained
  for prefill. A deleted agent yields 409 and the item stays
  open. A terminal-sourced item (pi in an Agent CLI terminal, ADR-0037's
  2026-09-09 amendment) is delivered through that terminal's receiver the
  same way — never through the task queue. A blocking question from a source
  with no channel at all is refused (409, `ErrNoReplyChannel`) and stays
  open: replying never closes an item while nothing was sent. `POST /api/inbox/{id}/state` triages (`unread|read|done`,
  `snoozedUntil`).
  PiCode itself files items from the RPC pump: a run that settles with
  no `/ws/agent` subscriber becomes a `result` carrying the agent's
  final message (an unread result per agent is superseded, not piled);
  an unexpected process exit becomes an `fyi` (a requested Stop files
  nothing). Agents file via `packages/pi-inbox` (`notify_human`,
  `ask_human`), identified by `PICODE_AGENT_ID` set on every spawn.
  Items are never deleted by any of the above; two more routes give the
  mailbox manual cleanup: `DELETE /api/inbox/{id}` removes one item in
  any state (204, 404 if absent), `DELETE /api/inbox?state=done`
  bulk-removes every done item (`{deleted: N}`) — the bare form without
  `?state=done` is refused (400) so nothing can wipe the mailbox by
  accident.
- `GET /api/workspaces/{id}/favicon` — the project's favicon (root, then
  public/static/app/src/app/www/docs and nested frontends, including Next.js
  App Router `icon.svg` under `app/` or `apps/<name>/app/`; svg > png > ico),
  read-only and confined to the folder; the workspace card wears it.
- `DELETE /api/workspaces/{id}` — remove (stops **all** agents first, then
  kills the workspace's terminals — sessions best-effort, records and
  settings overrides in one transaction; ADR-0026). With
  `?files=1&confirm=<folder name>` it also deletes the project folder
  from disk (ADR-0035): the confirm must match the folder's basename,
  root/home are refused, and validation runs before anything is removed.
  Optional `?sessions=1` deletes the pi session dir when this workspace is
  the last occupant of that cwd. Project folders are never deleted. The
  cleanup preview (`GET /api/workspaces/{id}/cleanup`) counts the terminals
  so the dialog can warn.
- `GET /api/workspaces/{id}/cleanup` / `GET /api/agents/{id}/cleanup` —
  preview for the delete dialog (session count, last occupant, owned work folder).
- `DELETE /api/agents/{id}` — unregister. Optional `?sessions=1&work=1`
  (work only if cwd is under `~/.picode/work/` and nobody else uses it).
- `GET /api/clis/{cli}/sessions` — the per-CLI session index (ADR-0079
  phase 2): pi plus Claude Code, Codex, Grok, Hermes Agent and OpenCode, read-only from disk
  (`internal/clisession`), each row with size/age/messages and
  server-verified resume arguments, tagged with the PiCode workspace that
  owns its folder. For pi the row also carries `inUseBy` (the agent whose
  current session it is) and the response adds `cleanupDays`; scoping by
  `?workspace=<id>` unions the shared cwd bucket with each of that
  workspace's agents' private dirs (`workspaceSessionDirs`, ADR-0040),
  unfiltered by ownership (unlike the per-agent picker, ADR-0039 — this
  view's job is to show everything). `POST /api/clis/pi/sessions/delete`
  `{path}` removes one orphan (in-use → 409, outside the pi root → 400);
  `GET/PUT
  /api/clis/pi/sessions/cleanup` is the orphan auto-clean preference in
  days (0 = off, default) — the sweep runs at boot, daily, and after each
  change, and never deletes a session any agent is bound to *or has ever
  been* (the `agent_sessions` history, ADR-0040: an older but still
  chat-picker-resumable session is not swept just because it isn't the
  current one). Together these power the `#/clis/<cli>/sessions`
  panes: Open with… reuses the resume
  endpoint, Compact reuses the agent compact. Session adoption
  (ADR-0021, "From a Pi session") was removed by ADR-0126 — agents are
  born only from new sessions. The legacy routes
  (`/api/sessions/all`, `/api/pi-sessions*`, `/api/session-cleanup`,
  `/api/workspaces/{id}/sessions/manage`) were removed — one namespace
  per CLI.
- `POST /api/workspaces/{id}/open|close` — start/stop the pi agent
  (idempotent); 409 on a workspace with no agents, like every
  workspace-scoped call that needs one (sessions, status)
- `GET /api/system` — pi/tmux detection + setup warnings (ADR-0003 UX).
  `pi.latest`/`pi.updateAvailable` ride along (registry check, 6 h cache);
  `POST /api/system/pi-update` runs `pi update --self`.
- `GET /ws/term?session=<name>` — xterm.js bridge (Pi TUI or project shell).
  The bridge sets `status off`, `allow-passthrough on` and extended keys on the
  session, then applies the session's **resolved** tmux options (ADR-0024) — on
  every attach, so a setting changed while the terminal was closed takes hold
  when it is reopened and an older session heals itself.
  `web/shared/domain/termClipboard.js` handles OSC 52 (write only; the read form is
  refused) so a copy made in the pane reaches the system clipboard — which is
  what keeps copying possible now that `mouse on` gives the drag to tmux.
  Study: `docs/benchmarks/2026-08-30-web-terminal-clipboard.md`.
- `GET/POST /api/terminals` · `POST /api/terminals/{id}/open` · `DELETE /api/terminals/{id}` · `GET /api/terminals/{id}/cwd` — first-class shells (ADR-0017). Each row carries `workspaceId` (`ws_free` = free; ADR-0026) — the wire stays flat and the sidebar groups client-side. `POST` accepts `workspaceId`; a workspace terminal with no cwd starts in the workspace folder. The list's `cwd` and its git facts both come from the live tmux pane path (record as fallback) — the sidebar says where the terminal is, not where it was born. `POST /api/terminals/{id}/runtime` accepts wrapper `start`/`end` leases for the canonical CLI, PID, and run id (ADR-0062); the ephemeral view carries `tui`, `cli`, and activity `state`. Workspace agent views carry per-agent git from the agent's effective directory (workPath, else the workspace path).
- `GET/PATCH /api/terminals/settings` · `GET/PATCH /api/terminals/{id}/settings`
  — terminal **behaviour** (ADR-0024). One global row plus a row per terminal
  holding only the fields that differ; `internal/termopts` is the registry of
  offered tmux options and the layering rule (defaults ← global ← overrides).
  A PATCH stores and applies live: a terminal patch touches that session, a
  global patch re-resolves **every** owned session individually, so an override
  is never overwritten by the pass that updates everyone else. "Owned" comes
  from the store — the terminals and agents this instance has records for —
  never from `tmux list-sessions`, which answers for the whole machine and
  would let one instance write into another's sessions. `null` clears a
  field — storing the inherited value instead would pin it. The curated
  registry (`mouse`, `status`, `allow-passthrough`, `extended-keys`,
  `extended-keys-format`) carries help and warnings and doubles as the
  defaults layer — the options PiCode once forced in code live here now, so a
  user override wins with no hardcoded exception. Beyond it,
  `GET /api/terminals/settings/catalog` serves the ENTIRE option space of the
  running tmux (ADR-0025), read live from `show-options -sg/-g/-wg`; any of
  it can be stored and applied, validated by tmux itself (scratch session for
  session/window values; server values apply for real) with its refusal
  surfaced in its own words. Server-scoped keys are refused on a per-terminal
  PATCH — tmux keeps them per machine and the UI labels them so. Array
  options travel as ONE string, entries joined by newlines: line *n* is
  `name[n]`, and applying rewrites the list per index, unsetting whatever the
  layer held past the new length (tmux leaves stale indexes in place
  otherwise). An empty block is refused — tmux keeps no empty array layer, so
  it could only behave as inherit while claiming to be a pin. The page is
  `#/termset` (global) and `#/termset/<id>` (one terminal).
  Appearance (font, colours, cursor) stays in `localStorage`, per browser.
- `GET /api/agents/{id}/cwd` — Pi TUI pane path (fallback: agent work dir)
- `GET /api/{agents|terminals|workspaces}/{id}/git` — the commit DAG,
  refs and worktrees of whatever repository that owner's cwd belongs to, plus
  the agents living in each worktree (`?limit=`, default 250). Each worktree
  also carries its own dirty count, `self` (the checkout the graph was read
  through) and `bare`/`detached`/`prunable` health flags, so the browser can
  draw one uncommitted row per dirty worktree (ADR-0073). Each commit
  carries `add`/`del` — its own diff totalled into one +/- pair for the
  listing column — read by one extra `--shortstat` walk over the same
  window with `-m --first-parent`, the very diff the commit detail shows
  per file, so row and detail can never disagree (measured ~0.8 s on this
  repository's 250-commit window; a failed or timed-out walk leaves the
  rows without numbers instead of stalling the graph). One graph per
  repository: the identity is `git rev-parse --git-common-dir`, canonicalized
  through filesystem symlinks, so every worktree (including macOS
  `/var`/`/private/var` aliases) answers with the same key and collapses onto
  one tab. The route
  carries the *owner* because the owner is what authorises the read — the
  server never resolves a repository from a path in the URL (ADR-0022). The
  workspace is an owner like the other two: a project with no agents and no
  terminals still reads its own history, through its folder (ADR-0027).
  Since ADR-0096 each local-branch ref also carries `upstream`, `ahead`,
  `behind`, `gone`, the `worktree` that holds it and `merged` (reachable from
  the HEAD the graph was read through), and the payload carries `remotes`.
  The tracking and checkout fields ride the `for-each-ref` call the graph
  already makes (measured: 9-22 ms either way); `merged` and `remotes` are one
  cheap exec each (7 ms). They exist so a menu can decide *before* the click:
  a branch checked out in a sibling worktree cannot be checked out or deleted,
  and the graph offers the way into that checkout instead of a command git
  would refuse. Phase 1 of ADR-0096 is read-only — a right-click on any row or
  pill opens the one PiCode context menu (`components/ContextMenu.jsx`) with
  clipboard and open-a-tab rows composed by `@picode/shared/domain/graphActions.js`,
  plus the line naming any agent mid-turn in this repository. The command
  composer those later phases deliver through lives in
  `@picode/shared/domain/gitCommands.js`, shared by desktop, mobile and the
  graph (it was duplicated between the first two until ADR-0096). Phase 1's
  read-only menus grew into the full write vocabulary: see the compose route
  below.
- `POST /api/{agents|terminals|workspaces}/{id}/git/compose` — the exact git
  command an action means, its risk tier, its plain-words verb and the
  sentence an agent would be asked (`internal/gitcmd`, ADR-0096). Composing is
  not delivering: what comes back travels to ADR-0078's `type` / `run` / `ask`
  routes, which keep their own guards, and the response is also the preview
  the form shows — so the promise and the command are the same string. Every
  ref that reaches argv is *validated*, never quoted: a leading dash, `..`,
  `@{`, a traversal or a shell metacharacter is refused. The branch a push
  publishes is read from the checkout, not taken from the caller.
  `GET /api/git/actions` serves the catalog (id, tier, required fields) the
  browser gates on; with none loaded the graph offers no write action at all.
  `GET …/git/head` (ADR-0038, now answered for workspaces too) is polled only
  while a delivered action is pending and the tab is visible; its token covers
  the worktree list as well, since adding or removing a checkout changes no
  ref. Risk tiers: **A** additive and reflog-recoverable, **B** moves HEAD or
  history, **C** publishes or destroys — C through the run door, the one place
  a human does not press Enter, asks for a typed phrase first. Before typing
  anything, the door clears the prompt line, so a command an earlier Prepare
  left unsubmitted cannot take the next one onto its end.
- `POST /api/terminals/{id}/ask` and `…/tui-hello`, `…/tui-ack` — the ask
  door for a pi running as an Agent CLI terminal (ADR-0089, amended
  2026-09-08). The receiver extension is injected into pi terminals by the
  wrapper and identifies itself by `PICODE_TERM_ID`; its hello names the
  session it shows. An ask is delivered only through that receiver
  (`tui-inbox/term-<id>/`), never pasted; refusals are `no-receiver`,
  `no-session`, `moved`, `busy`, `stopped`, and `cli` for any other CLI.
  Provenance is an event, since the task queue belongs to agents. The graph
  lists such terminals as worktree occupants (`kind: "terminal"`, `live`).
- `GET /api/{agents|terminals|workspaces}/{id}/git/commit?hash=`
  — one commit with its message body and its patch, already split per file.
  `hash` must be a full object name (40/64 hex): it is the only user-supplied
  part of the git command line, so a ref or a leading dash is refused rather
  than passed through. The patch is read with `-m --first-parent`, which is
  what keeps a merge from arriving as a combined diff (`diff --cc`, `@@@`)
  that a unified-diff reader misreads without ever failing.
- `GET /api/{agents|terminals|workspaces}/{id}/browse?dir=` — one directory
  level under the owner's folder (terminals read the live pane cwd; a
  workspace reads its registered folder, `ws_free` refused — ADR-0030). The
  answer carries `root`, the canonical folder, which is the tree tab's
  identity. Workspaces also mirror `text` (GET/PUT), `blob` and `file`, so
  an empty workspace (ADR-0027) can open files with nobody in it.
- `GET /api/{agents|terminals|workspaces}/{id}/gitstatus` — the working-tree
  changes of the owner's repository, `git status --porcelain -z -uall`
  re-anchored from the repo toplevel to the owner's cwd (what falls outside
  is dropped). All three owner kinds accept `?worktree=<branch|head hash>` to
  read a sibling worktree of the same repository instead: the value is a ref
  resolved through `git worktree list`, never a path from the URL, and an
  unresolvable ref is 404 (ADR-0073). No repository is a state, not an error:
  `200 {"git": false}`. Since ADR-0078 every change carries `add`, `del` and
  `binary` (one `git diff HEAD --numstat -z -M` for tracked files; untracked
  files are counted in Go with a 4 MiB cap and a `truncated` flag), and the
  page carries `branch`, `worktree` (linked worktrees only), `totals`
  (`add`, `del`, `files`) summed over the re-anchored slice, and the branch's
  distance to its upstream: `upstream` (absent when none), `ahead`, `behind`
  (`git rev-list --left-right --count @{upstream}...HEAD`) and `detached`.
- `GET /api/{agents|terminals|workspaces}/{id}/gitdiff?path=` — one file's
  working-tree-vs-HEAD patch (ADR-0032), confined by the same cwd rules;
  untracked files arrive as whole-file additions, binary and truncation
  flagged like the commit route. 404 when there is no difference. The same
  `?worktree=` narrowing covers the working-tree `blob` and revision
  `git/blob` reads for every owner kind, so any owner previews a sibling
  worktree's assets, committed or not (ADR-0073 as amended).
- `POST /api/{agents|terminals|workspaces}/{id}/reveal` — opens the owner's
  folder (optional confined `{"path"}` body) in the host file manager via
  `internal/osopen` (WSL → explorer.exe, darwin → open, else xdg-open).
  Host-local by design: a remote browser opens it on the server's desktop.

- `GET /api/{agents|terminals|workspaces}/{id}/pr` — the pull request of the
  owner's branch through the host's `gh` (ADR-0078). Always 200 with a
  `status`: `ok` (number, title, url, state, draft, reviewDecision, head,
  base, additions, deletions, changedFiles, author, updatedAt, and `checks`
  folded into passed/failed/pending/skipped plus the failing names), `none`
  (no pull request for `branch`), or `blocked` with a `reason` (`gh-missing`,
  `gh-unauth`, `no-remote`, `no-git`, `gh-timeout`, `gh-error`) and a
  message. Cached a minute per folder and branch; `?refresh=1` asks gh again;
  no background poller. PiCode never holds a GitHub token (ADR-0034).
- `POST /api/terminals/{id}/type` `{"text"}` — types a command into the
  terminal's pane as literal keystrokes (`tmux send-keys -l`), never Enter:
  the Inspector pre-fills `gh pr create --fill` or `gh auth login` for the
  human to submit. Control characters, newlines, a leading dash and texts
  over 2000 characters are refused (400); a terminal without a live pane is
  409, and so are a terminal whose live cwd differs from an optional `root`
  (`reason: moved`) or whose pane is not at a shell (`reason: foreground`).
- `POST /api/terminals/{id}/run` `{"text","root"}` — the same keystrokes plus
  Enter, behind the Inspector's interlock (ADR-0078 stage 2): 409 `busy`
  naming every PiCode-known writer of that repository (agents mid-turn, TUIs
  working, automation runs, other terminals reported working or holding a
  foreground program), 409 `moved` / `foreground` / `closed` for the target
  terminal itself; otherwise `{"ran": true}`. Git runs in the user's shell,
  never in the service process; the interlock is advisory and momentary.

ADR-0074 adds optional `?root=<canonical folder>` to browse, text (GET/PUT),
blob, gitstatus, gitdiff, git/blob, pr and reveal. The resolved owner cwd remains
the authority: the parameter only asserts equality, and a mismatch returns
409 before reading or writing. The tree pins these requests to its open root;
only explicit Refresh may adopt a new terminal cwd, after its document guard.
