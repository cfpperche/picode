# Handoff — living project state

> Heartbeat of PiCode. Session that changes state **must** leave this file
> matching **HEAD** (ritual: `/skill:handoff-update`, contract: AGENTS.md).
> Stale *Next up* (listing shipped work) is FAIL. Newest *Recent activity*
> first. Archive to `docs/handoff-archive.md` when this file exceeds ~150 lines.

## Current state (read this first)

**Visual gate:** `read` uiux-review **before** JSX. Empty/blocked/error
screenshots must be `read`. overlayAudit + visual-card. Clip = FAIL.
Skip = quality-gate FAIL.

**Phase:** ADE past M3. Composer **A** + MCP **B** + Track **C** (waiting, queue, draft) shipped. Slash TUI **24 ui · 0 missing**. Providers + vault.
Public docs on Pages. llama.cpp manager is **docs + dialog**, not an installer.
Local backup V1 in Preferences (ADR-0014).

What exists:
- Public `cfpperche/picode`, PolyForm Noncommercial + commercial, CI linux/macos/windows.
- HTTPS default `:8445` (ADR-0007). SW caches **only hashed `/assets/`**; HTML/API network.
- Workspaces + many agents per folder (ADR-0011). Settings vs Preferences (ADR-0012). Packages: machine / workspace / **this agent**; optional isolate skips inherit (ADR-0010).
- Providers: API key + OAuth. Claude/Codex loopback `53692`/`1455`. Copilot/Kimi/xAI **device-code**. Radius stays TUI (gateway URL). **Usage** (ADR-0031) on each vault account on `#/providers` (OAuth plans, ZAI / OpenCode Go / OpenRouter / MiniMax / Kimi keys) — live 5h/7d/week/extra/credits; Codex/Grok banked resets with confirm-to-redeem. Grok resets fall back to `~/.grok/auth.json` then `GROK_COOKIE`.
- **ADR-0013** vault `~/.picode/accounts.json`; `auth.json` is the one slot pi reads. Add account / Use / Sign out; OAuth re-login updates the same account.
- Composer `/` opens **PiCode UI** (never the dock). Skills/templates = picker insert; pi RPC expands. `/share` = secret gist (`gh`).
- `/llama` dialog on the **current view** (URL, Save, Retry, load/unload, HF download). Link **Set up llama.cpp** → `www/guide/llama.md`. **No GUI installer** (reverted).
- Voice **V1 shipped** (dictation + Grok composer + browser TTS). Owner dogfood done (Chrome Windows mic).
- Public docs: VitePress `www/` → GitHub Pages. GUI chrome carries **state**, not docs. ADRs 0001–0013.
- `#/mcps` missing adapter: one line + Open packages (`www/guide/mcp.md`). A terminal tab (or gone agent) does not hide an installed adapter — same as Packages. Load error is Retry, not the install prompt. No npm/architecture in the view.
- MCP / Packages / Settings name the selected agent (icon + name) as the first line in the card. Scope pills say **This agent**. Sidebar agent click from a pane goes to `#/`.
- MCP **Use from…** mirrors other apps. User picks; Off hides a server. Empty host files are not offered.
- MCP Add **More**: env on command servers; headers + Sign in / Token on URL servers.
- MCP list live state: Idle / Live / Failed / Sign in when the GUI agent is running. File Off has no word.
- MCP **Sign in** is automatic like providers: short `pi -e` runs headless `authenticate()` (no paste). Pi does not open the browser. GUI `window.open`s the URL once. Overlay ends when the callback HTML is served (keyring write continues). OAuth rows with tokens show **Sign out** (forgets the keyring login on this machine). No extra “Signed in” label.
- Composer `@` / images / `!cmd` shipped. MCP list/add/toggle/remove/Use from/live/auth shipped.
- Track **C1 waiting**: confirm/select/input/editor is a chat card; sequential selects are a labeled stepper (back on prior pills via the extension's explicit `‹ back` option — ADR-0028 amendment; Cancel/Stop aborts); done is one definition line persisted in localStorage (per-agent live slot before the first session file exists); `POST /api/agents/{id}/ui`. Mobile has no waiting card.
- Track **C3** queue: Send while busy/waiting is follow-up (or Steer from the kind chip). Follow-up is held until idle — Edit / Remove. Abort drops Steer.
- Track **C2** draft: composer text + kind persist per agent (`picode-drafts`). Images do not.
- Track **D1** `#/agent/<id>`: URL is the open agent. Missing id: “That agent is gone.”
- Track **D2** session chip shows `$0.05` when the session has spent money.
- Track **D3** `@` lists files, other agents, and skills (`@agent:…` / `@skill:…`). Mentions only.
- Track **D4** `/tree` is Prompts. Now on the current card. Other cards confirm then start a new session (this one stays). In-place jump still pi#8645.
- Track **D5** behind npm packages show **Update**. User menu dots Packages. Nothing updates until you click. Git / path / pinned skipped.
- **ADR-0015** + Track **E**: E1–E4 shipped (open, Save, Keep/Undo, turn file names).
- **ADR-0017** first-class terminals (sidebar + `#/term/<id>`). ADR-0016 editor-tab UI superseded. Pi TUI dock unchanged.
- **ADR-0020** PiCode Desktop: a Windows tray binary provisions the distro; `picode provision` does the Linux half. ADR-0018 superseded (it had ruled out both a logon task and linger). **M2–M5 shipped** (the ADR-0020 plan is complete): `picode provision` (6 steps, `--dry-run` / `--json`) and `picode-desktop.exe` (tray, logon task, keepalive, CA trust, clean-machine bootstrap). Neither has been **run for real** — dry-run only, by the owner's decision.
- Preferences → **Terminal** (colors, font, size, line height, spacing, cursor, blink, scrollback, padding, **Keys**: newline + copy-if-selected). Ligatures omitted: xterm canvas in the browser cannot join glyphs (`@xterm/addon-ligatures` needs Node font-finder).
- **ADR-0028** `packages/pi-roles/`: opt-in MIT pi package (carve-out from PolyForm). Not installed by default. `/roles edit|add|remove` writes `.pi/roles.json`. Composer lists those commands while the agent is running (ADR-0029).
- **ADR-0033** pi-roles v2: `PI_ROLES_AGENT` overlays `.pi/roles/<id>.json` on the workspace file. PiCode sets the env on RPC and TUI start. Amendment: `/roles edit|add` end with a **Save to** select (this agent / workspace) under the env; `/roles clear [agent|workspace]` deletes a whole roles file.
- **ADR-0039** per-agent session ownership: an Agent's **Search sessions** picker now shows only sessions PiCode has recorded as that agent's own (`agent_sessions` table), not every pi JSONL in the shared cwd bucket. Fresh spawns pre-mint a `--session-id`; resume/fork/clone/adopt/import historize the path they point at. Machine-wide "All sessions" / "Manage sessions" stay unfiltered on purpose.
- **ADR-0040** per-agent `--session-dir`: every agent spawn (both run modes) also gets its own private `~/.pi/agent/sessions/<agentID>/`, so pi's **own** native "Resume Session" TUI picker — unreachable by ADR-0039's DB filter — is scoped too. Verified live: two agents sharing a cwd, each attached via tmux, each agent's own in-TUI resume overlay shows only its own session. Terminals unaffected (never call `CLIFlags()`). Orphan sweep also now protects any session in `agent_sessions`, not just an agent's *current* pointer.
- **ADR-0053** a pending session resolves at spawn: before minting a fresh `--session-id`, both spawn chokepoints adopt the newest file matching an earlier run's pending id (`store.ResolvePendingAgentSession`), so chat → TUI → chat stays on **one session** instead of minting a competitor each hop; the chat picker unions the cwd bucket with the private dir (fresh sessions were invisible there — an actively-used agent read as "No sessions yet"); and the desktop status bar fetches `/status?agent=<selected>`, so a new agent no longer opens with the workspace's first agent's context/spend/cache. Tested row by row (spawn table + picker table); gap: `Runtime.Start`'s identical adoption branch has no live managed-spawn test. **Deployed and verified on the owner's machine 2026-09-02** — the owner's "not fixed" report traced to the installed 16:49 binary, which predated these commits. Same-day follow-up: the explicit **+ New** seals pendings before the restart so adoption never overrides it (fix/new-session-fresh-start).
- **ADR-0041** session observability dashboard: the no-tabs-open main pane now shows spend/activity/fleet stat tiles + spend-by-provider (Today/7d/30d/All), replacing both the bare "No agents yet" copy for that case and the branch's own earlier rejected `HomeView` (workspace/agent list) attempt. `GET /api/sessions/stats` bucket at message granularity via `entryTS`, not file mtime. No chart library; hand-rolled SVG mirrors the `GitGraph.jsx`/`lib/gitgraph.js` split.
- **ADR-0047** Web Push: `internal/push` (stdlib VAPID + RFC 8291), `store` migration 018, `/api/push/*`, `PushPrefs.jsx`; presence-aware (host online → no push); real-device dogfood pending (needs the mkcert cert on the phone; iOS needs Home Screen install).
- **ADR-0046** one modal primitive: `components/ResponsiveDialog.jsx` (Radix dialog ≥720px, Vaul sheet below, `Alert` for confirms); `.dlg.dlg-sheet` CSS; `lib/dialogPolicy.test.js` refuses raw dialog/drawer imports outside it (allowlist: `Palette`, `Hotkeys`).
- **ADR-0044** mobile shell v2 — supervision console (amended same day: no header; Work tab = workspaces / free agents / terminals; `#/term/<id>` screen with key bar). `web/src/mobile/` is now screens (`Now`, `Inbox`, `Agents`, `Agent`, `More`) + hooks (`useHashRoute`, `usePoll`, `useFleet`, `useAgentSocket`) + components (`TabBar`, `ScreenHeader`, `StateChip`, `NeedsYouCard`, `AgentRow`, `StatStrip`, `CreateSheet`) over pure libs `lib/mobileRoutes.js`, `lib/agentEvents.js` (reducer of the agent WS stream; desktop `handleEvent` untouched), `lib/needsYou.js`, `lib/createSubmit.js` (lifted from `submitNew`). Server: `agentView.streaming/waiting/dialog` from `Runtime.Get(id).Snapshot()`; `AppSurface` gained `initialPath`; `summarizeArgs` moved to `lib/toolArgs.js` (re-exported). The waiting card exists on mobile now — the two "Mobile has no waiting card" lines below are history. Push is the next ADR (phase 2 of the approved plan: Web Push/VAPID in Go stdlib, presence-aware).
- **ADR-0042** dashboard v2: same scan now yields `byModel`, `byWorkspace` (server labels cwd → workspace via `canonDir`), `tokens` (+ cache hit), `tools` (top 8), `turns` (assistant/user/errors/aborted/compactions), `topSessions` (top 5, name/cwd/lastAt, never preview); `Series[].turns`. Server memoises per range behind `session.Fingerprint` (stat sweep: count/size/newest mtime). UI: 4 tiles (Sessions + Fleet strip from `workingIds`/`waitingId`), `DailyChart` (`lib/barchart.js`), `RankedBars` (folds tail into "N more"), `TokenBar`, Reliability facts, `TopSessions`; 60 s auto-refresh + 30 s tick, paused when hidden. Not built (refused in the ADR): projection/burn rate, LOC/commits, latency, per-agent context % on the home.
- **ADR-0043** Chrome extension Track A (sensor) and Track B (devices + Windows console host): `ext/` MV3; `GET/POST /api/extension/*`; `make desktop` also builds `picode-nmh.exe`; side panel pings `kind=extension`. Isolated Chromium unchanged. Chrome-only. Not an App, not a pi package.
- **ADR-0054** extension actuator (Track C): "Let the agent act on this page" — the agent may end a reply with one ```picode-act JSON block; the server validates it into an `act_batches` row (migration 021), the panel polls via the native host, asks once per origin (grant in `chrome.storage.local`, revocable), executes action-by-action with visible highlights (`chrome.scripting`, injected on demand), and posts outcomes back as one more watched turn; 3 rounds cap, Stop, 10-min expiry, claimed batches resumable. Routes: `GET /api/extension/act/next`, `POST /api/extension/act/{id}/result`.
- **ADR-0045** Automations: `#/automations` (user menu, palette). Scheduler goroutine in the daemon (`internal/automate`, 1-min tick, deterministic jitter, single catch-up, boot reconcile) + webhook `POST /api/automations/{id}/fire` (bearer secret, 64 KB). One agent per automation, fresh session per run; `RunObserver` on the managed agent; cost cap / 2 h watchdog; runs table; Inbox `sourceKind: automation` (migration 017 rebuilt `inbox_items`). Decision table in `internal/server/automations_run.go`, tested row by row. v1 only: no templates, no `/automate`.
- **ADR-0049** authentication: `internal/auth` gate wraps the mux (`Deps.Auth`, nil = ungated in tests); modes off/remote/all (`auth.mode`); `auth_sessions` + `auth_pairings` (migration 019); install token `<data>/token`; routes `/api/auth/*` + `/pair`; `picode pair`, `picode token [rotate]`; Devices section in Preferences → Server; pairing screen on 401; QR carries a pairing link; native host and `pi-inbox` send the bearer; `PICODE_DATA` in spawn env. Roadmap: `docs/design/remote-modes-roadmap.md` (Tracks B–D next).
- **ADR-0048** change feed: `events` is the change log (every store mutation appends in-tx; `Store.OnEvent` after commit; invariant test enumerates mutators), `internal/feed` (replay under the subscribe lock, `ErrReset`, ephemeral id 0), `GET /api/events` (SSE `hello`/`change`/`reset`, `Last-Event-ID` or `?after=`), push consumes the feed (`Notifier.OnEvent`), presence `OnChange` → `device.online`, runtime `OnState` → `agent.state`. Web: `lib/feed.js` (cursor in sessionStorage, bootId kick), `lib/feedReducers.js` (fleet / inbox / automations / runs; `null` = refetch), desktop sidebar + badges + AppSurface + Devices + Automations and mobile `useFleet` / `usePoll` / inbox loop on the feed; timers tick only while the feed is down.
- **ADR-0045 v2** `/automate` + templates: `lib/automateDraft.js` (prompt, fence/object parser, command detection), `lib/automationDraft.js` (read-once `sessionStorage` handoff), `automate.Templates()` + `GET /api/automations/templates`, Suggested cards with category chips, *Start from template…* in the editor, "Drafted by / From template" origin line. Turn correlation is client-side (`automateRef` + `agent_settled` + `lastAssistantText`); no server change for `/automate`.

## In flight

**Feed migration phase 4 — merged to `main` 2026-09-03, deployed.**
`StartPackageUpdatesWatch` re-runs the npm update check every
30 min for the user dir + every registered workspace (serial,
network-bound, fingerprint per scope) and publishes ephemeral
`packages.updates` only on change; the desktop badge/list apply the
event directly (the scan is the data) and the 30-min client interval is
now the feed-down fallback. Publish/diff/fingerprint table-tested with
a stubbed scan; the boot publish seeds the diff state and is by design
unobservable to clients (ephemeral, fires before any listener can
attach). The live client-apply path is exercised only when a result
changes mid-connection — covered by the unit test, not by a browser
e2e (would need a real npm package flip). **Migration complete.**

**Feed migration phase 3 — merged to `main` 2026-09-02, deployed.**
`StartGitWatch` inspects each workspace path and agent
cwd once per 3 s tick and publishes ephemeral `git.updated` (fleet-wide,
one Inspect per changed dir); `applyFleet` patches the sidebar pills in
place (desktop + mobile — both `touches` lists now include "git"), and
FileTreeSurface reloads itself when its root changes. Verified live: a
terminal commit cleared the dirty badge with zero fleet refetches; a
new file re-raised it and appeared in the tree on its own. Terminal
panes are not watched (their cwd is live tmux state — their pills still
refresh with the fleet). Phase 4 (`packages.updates`) is pending.

**Feed migration phase 2 — merged to `main` 2026-09-02, deployed.**
The llama.cpp panel's `runOp` no longer polls `/api/llama` at 1 s while
an operation runs: the endpoints block until done and the busy row is
the motion, so one refresh on completion is the only refetch. Verified
live against a fake router (blocked Load + Unload → exactly one GET).
Phase 4 (`packages.updates`) is pending.

**Feed migration phase 1 — merged to `main` 2026-09-02, deployed.**
`StartMcpWatch` (mirrors `tui_watch.go`) reads each running agent's
live MCP snapshot once per 3 s tick for the whole fleet and publishes
ephemeral `mcp.updated`; the Mcps panel subscribes, reconciles on
`feed.open`/`feed.reset`, and its 2.5 s interval is now the feed-down
fallback only. Verified live end to end: hand-written live file flips
the badge with exactly one `/api/mcp` request (was ~1 per 2.5 s).
(main had independently repaired the stale LooksWorking tests in
485be488; the branch's duplicate fix was dropped in the merge.)
Phases 2–4 of the polling→feed migration plan (llama op overlay, git
events, `packages.updates`) are not started.

**ADR-0045 Automations — merged to `main` and shipping.** Scheduler,
webhook, notify channel, gateway hook route, `/automate` + templates;
editor/detail UI fixes and the message-run amendment landed 2026-09-02.
Owed: acceptance runs listed under *Next up* (owner's sudo).

**ADR-0054 Track C on `feat/ext-track-c`** — coded end to end and gated;
NOT yet live-dogfooded: the loop depends on the model emitting the
```picode-act block, so the first real send-with-act is the acceptance
test. Panel states verified via `?preview=` screenshots only.

**ADR-0025 — the whole tmux catalog is a settings surface. Delivered.**
The owner overruled ADR-0024's "grows from parity gaps" rule: the GUI exposes
all of tmux's options (142 on this machine's 3.6), read live from
`show-options` across the three scopes, validated by tmux itself (scratch
session for session/window values; server values apply for real). The old
hardcoded forces — status off, allow-passthrough on, extended keys — are now
curated *defaults* the user can override, consequence labelled. The dialog
became a page (`#/termset`, `#/termset/<id>`): featured tier, search, scope
sections, danger labels. Arrays are editable as a block since 2026-08-30 —
one entry per line, line *n* is `name[n]` (ADR-0025 amendment).
Caught in browser QA and fixed: the search now filters the featured tier too
— unfiltered, a featured control sat where a result was expected and took the
click meant for it.

**ADR-0024 — terminal settings. Delivered except presets, which are held on
purpose (see *Next up*).** `internal/termopts` is the registry of offered tmux
options and the layering rule; `terminal_settings` holds one global row plus a
row per terminal carrying only what differs. `GET/PATCH /api/terminals/settings`
and `.../{id}/settings` read and write it, `null` clears a field, and unknown
keys or values are refused rather than dropped. Two gears in the Terminals
list: beside **+** for the defaults, on a row for that terminal.

Applied in three places, all resolving the same way: at creation, on every
attach (so a session that predates a setting heals itself), and on PATCH. A
global PATCH re-resolves **each** session on its own rather than pushing the
new global at all of them — a test pins that, and it fails if you take the
shortcut.

The session list comes from the **store**, not from `tmux list-sessions`.
That distinction was learned the hard way on 2026-08-30: `list-sessions`
answers for the whole machine, so a single `go test` with its own database in
`/tmp` flipped `mouse` on the developer's live terminals, which wore the same
prefix. A test now seeds a foreign session and asserts a global change leaves
it alone; a second test asserts the change still reaches an owned one, so the
scoping cannot quietly become applying to nothing.

Measured, not assumed: flipping `mouse` on a session with a client attached
takes effect with no reattach, which is why a PATCH is enough. The chain is in
the ADR. The known cost: while tracking is on the drag belongs to the
application, so native selection needs Shift and a copy-mode drag reaches the
clipboard as `OSC 52 ; ; <base64>`.

Only `mouse` is offered. The ADR says the list grows from real parity gaps,
and that is still the only one anybody has hit.

**ADR-0022 — git graph. G1 (data), G2 (the graph) and G3 (commit diff) are
built and visually verified. The ADR is delivered. ADR-0038 (v2) is also
delivered:** inline resizable commit detail (the ported `expandAt`/`expandY`
hooks finally run, tests first), the Uncommitted Changes row (pseudo-commit
through the ordinary allocator, dashed trail to HEAD, gitstatus/gitdiff on
click), client-side search (dim + walk, never hide), numstat + clickable
parents in the detail, token-polled auto-refresh (5s, visible tab only —
supersedes 0030's manual-refresh precedent for the graph), and remote pills
that read as remote. QA'd on a scratch instance against this repo: inline
panel below the clicked row with lines detouring, parent-link navigation,
uncommitted 1→2 within 7s of an external touch, sizer drag persisting
280→360px, file tree WorkingDiff untouched.

The repository icon beside an agent or terminal in the sidebar opens
`#/git/<t|a>/<id>`; the tab is `g:<git-common-dir>`, so two owners in two
worktrees of one repo land on the same tab. Rows carry ref chips and, on a
branch, the agents living in the worktree checked out on it. Clicking a commit
splits the surface and shows its message and patch, one card per file.

Verified on the real repo at 250 commits, dark and light: `overlayAudit ok`,
no horizontal scroll, 26px rows, the selected row stays in view when the diff
pane opens, and a 59-file commit reports `truncated` rather than trying to
render megabytes.

## Next up

0. **Live browser preview (agent_browser) — ADR owed** from the 2026-09-02
   benchmark study (`docs/benchmarks/2026-09-02-live-browser-preview.md`):
   proposal is a tool-agnostic `details.preview` contract emitted by the
   package (or a thin `pi-agent-browser-view` companion) + generic rendering
   in the conversation (reducer consumes `tool_execution_update`; tool pills
   render preview images) + a Browser surface on `#/agent/<id>`; v2 proxies
   `agent-browser stream enable` behind `internal/auth`. No new package
   system — pi's exists (ADR-0010); only the tool→GUI rendering contract is
   missing. Decision for the owner: propose the contract upstream to pi
   first vs. PiCode-side convention.
1. **Remote modes — acceptance runs owed** (owner's sudo): Track C on a real second account (`sudo picode gateway install`, `sudo picode provision --user demo --shared`, `sudo picode users add <login> demo`); Track D.2 (`sudo apt install systemd-container debootstrap`, `sudo picode provision --user demo --shared --container`); Track D.1 with a real GitHub OAuth app behind Caddy on a public name. Then decide on Track E (SaaS: signup, quotas, billing, VM per client) — the roadmap sketches it.
2. **Feed follow-ups (ADR-0048)** — git events shipped 2026-09-03
   (phase 3; the create / clone / config-PATCH refetches it would retire
   were already feed-first via `refreshFleetFallback`). Still open:
   outbound webhooks fed by the log; a fake pi fixture that emits
   `usage.totalTokens` so the context bar can be tested end to end.
2. **ADR-0053 leftovers** — a live managed-spawn test for
   `Runtime.Start`'s adoption branch (needs a fake pi RPC fixture; the
   spawnFlags twin is table-tested); decide whether a chat send that
   would kill a *working* TUI should confirm first (refused for now —
   with adoption the loss is one in-flight turn; see the ADR's
   alternatives).
2. **Automations** — `pi-automate` only if `/automate`'s fence parsing
   proves flaky. Notify URL, webhook recipes and the gateway hook route
   shipped 2026-09-02.
**ADR-0043 Track C** — built on `feat/ext-track-c` (ADR-0054). After merge
and live dogfood: Track D (`packages/pi-tab`) and E stay parked.

**Watch: retiring the Shift+Enter shim (`web/src/lib/termKeys.js`).** Researched
2026-08-31 at the owner's request: today it cannot be replaced by tmux/xterm.js
configuration. Stable `@xterm/xterm` (6.0) encodes no modified-Enter protocol at
all; the kitty keyboard protocol landed upstream (xterm.js#5600, Jan 2026) but
only in 6.1.0-beta — and tmux deliberately speaks only modifyOtherKeys to the
outer terminal (tmux#4038; kitty outward refused in tmux#3335), so the beta
alone would not close our chain. Every xterm.js project ships the same
customKeyEventHandler shim (e.g. Kilo-Org/cloud#963). Re-evaluate when
`@xterm/xterm` ≥ 6.1 goes stable: enabling kitty via `vtExtensions` plus a
local activation on attach (`term.write("\x1b[>1u")`) could retire the three
layers — tmux accepts CSI u as input without negotiating — but measure first:
flag 1 changes Esc/Ctrl+C encoding too. Also watch `coder/ghostty-web`
(Ghostty's VT parser in WASM, xterm.js-compatible API, kitty encoder built in):
the natural mid-term engine candidate, but born ~Nov 2025 with basic input gaps
open (coder/ghostty-web#145) — and swapping engines does not remove the shim
while tmux sits in the middle.

**ADR-0024 presets, held for the owner's call.** The ADR has the user creating,
editing and deleting presets and stamping them onto a terminal. With exactly
one flag offered, a preset would carry a single boolean — more clicks than the
toggle it replaces. The recommendation given to the owner on 2026-08-30 was to
build them once there are three or four flags, and the store shape is already
ready for it (overrides are a map, not a column). Not refused, deferred.

**Multi-runtime TUIs in terminals** (owner direction): the research
landed 2026-09-03 — study
`docs/benchmarks/2026-09-03-guest-tui-agent-state.md` + **ADR-0056
(proposed, owner's call)**: guests keep their TUIs in terminal
surfaces; sidebar working/needs-you arrives via per-tool sensors
(Claude hooks → HTTP, Codex `notify`, opencode server events) on the
ADR-0048 feed; screen-scraping refused as primary; ACP/control is the
named future track (own ADR; re-measures ADR-0003's Pi-only clause).
The unified TermSurface/ShellTerm path stays the substrate. Known
landmine for that track: tmux's `extended-keys-format` is server-wide
— we force `xterm` (modifyOtherKeys) so Shift+Enter survives, but
some TUIs prefer csi-u/Kitty; per-session key format may be needed.

**PiCode Desktop is installed and running on the owner's machine. What is
left is validation — the list below is what has *not* been proved.**

The one that matters: **`wsl --shutdown`, then a Windows sign-out and back
in.** Everything else is inference until this runs. It is the only test that
proves linger and the logon task work together with nobody signed in — which
is the entire problem PiCode Desktop was built for. Cost: it ends the running
tmux sessions, so it is the owner's call. Expected result: sign in, the tray
appears on its own, `https://localhost:8445` answers without anyone opening a
terminal.

Never exercised, because this machine was already past them:

| Path | Why it never ran | How to reach it |
|---|---|---|
| `/etc/wsl.conf` write (backup + line merge) | `systemd=true` was already set, so the fix branch never fired here | a distro without it, or a scratch `ConfPath` |
| mkcert CA import into Windows | the CA was already trusted from an earlier `setup-cert.sh` | a machine that never ran that script |
| mkcert certificate issuance | the cert is valid until 2028-11-25 | `renewWithin` is 30 days, or delete `~/.picode/cert.pem` |
| Clean-machine bootstrap (M4) | no spare Windows VM: `wsl --install`, the 3010 reboot, `RunOnce` resume, account creation | a bare Windows VM |
| Release workflow | it only fires on a tag, and none has been cut | tag `v0.1.x` |
| `picode-desktop update` | needs a published release to find | after the first tag |
| `picode-desktop uninstall` | never run | it only removes the logon task; safe to try |
| Tray menu actions | the process and its keepalive were verified, but Open / Restart / View logs were never clicked | click them |
| `install`'s own output | the elevated copy writes to a console this session could not see; only its *effects* were checked (task registered, CA trusted) | run it from a Windows terminal |

## Backlog

- llama.cpp: in-app installer / start router, SSE progress + cancel, delete `.gguf`, Ollama/vLLM (`models.json`).
- Mobile phases 2 and 3 **fully proved on the owner's iPhone**, real device throughout — this line closes the loop: subscribe, `Send test`, a blocking inbox item with the desktop closed → push → tap → Accept → follow-up queued for the agent (ADR-0047); the live-dialog push path (an extension prompt with the desktop closed) confirmed too; pull-to-refresh and the Inbox row swipe confirmed on real touch (ADR-0044 phase 3). Nothing left to validate here — the mobile shell's three phases are done end to end.
- `/tree` in-place leaf jump needs pi RPC `navigate_tree` ([pi#8645](https://github.com/earendil-works/pi/issues/8645)); today click forks.
- Cold start parses the whole session JSONL (10 s on a 129 MB session) — filed upstream: [pi#8843](https://github.com/earendil-works/pi/issues/8843) (lazy resume / load checkpoint).
- Worktrees / parallel isolated agents (Orca + Herdr) — after Track E.

## Known debts / open questions

- **Copilot and Google quota adapters are out of date (found by the
  2026-09-03 providers study).** GitHub retired Copilot *premium requests*
  on 2026-06-01 in favour of token-metered AI Credits with budgets, so
  `internal/usage`'s `copilot_internal/user` reading of
  `quotaSnapshots.premiumInteractions` needs re-measuring (it also never
  had a reset date — the field does not exist; the reset is the 1st at
  00:00 UTC and must be computed). Gemini CLI stopped serving Google AI
  Pro/Ultra individuals on 2026-06-18; the consumer quota surface moved to
  Antigravity's `cloudcode-pa.googleapis.com/v1internal:retrieveUserQuotaSummary`
  (`buckets[].remainingFraction`/`resetTime`), which PiCode does not read.
  Other endpoints that fit `usage.Report` unchanged: Anthropic
  `oauth/profile` (email + plan), `overage_spend_limit`, `prepaid/credits`;
  `openrouter.ai/api/v1/credits`; `api.deepseek.com/user/balance`;
  `api.moonshot.ai/v1/users/me/balance`; Fireworks `billing/summary`; the
  Qwen coding-plan endpoint ADR-0031 refused in V3 for lack of an API-key
  path. Groq, Mistral and Cursor stay cookie-scoped — honest `unavailable`,
  not scraping.

- **CI on `main` is red on macOS and Windows — environment-dependent
  tests, pre-existing (every run since at least 2026-08-30 failed; each
  era's first failing step masked the rest: dead links →
  `remote-server.md` → the git-guard self-test, fixed in c503b9da).
  Diagnosed 2026-09-03:** `internal/backup`'s `TestValidateDest` and
  `TestSnapshotRestoreMatrix` fail because `canon()` `EvalSymlinks` the
  destination — a **not-yet-existing** dest under a symlinked `TMPDIR`
  (macOS `/var` → `/private/var`) stays unresolved while `dataDir`
  resolves, so `filepath.Rel` misses the containment and a dest inside
  data is accepted (fails on macos/windows runners, passes on linux and
  locally). Also failing off-linux: `gitgraph`
  `TestWorktreeSharesTheKey`, `install`
  `TestDeployRefusesBeforeCopyingWhenThereIsNoSession`, `presence`
  `TestExpireAnnouncesOnceAndPingRevives`, and several `internal/server`
  tests (`TestAutomationStartRunEndToEnd`, `TestBackupAPI`,
  `TestGraphCollapsesWorktreesAndNamesOccupants`,
  `TestOccupantScanAsksGitOncePerDirectory`,
  `TestNewSessionFreeAgentTUIRestart` — the last one new with the
  ADR-0053 push). On **ubuntu** the self-test fix unmasked the next
  layer: the job's `apt-get install tmux` gets **3.4** while PiCode
  needs 3.5+ — `TestEnsureExtendedKeysXterm` dies on `invalid option:
  extended-keys-format` and the option-catalog tests see different
  kinds (`default-shell kind = ""`); `presence`'s
  `TestExpireAnnouncesOnceAndPingRevives` also fails there (passes
  locally on tmux 3.6; suspect timing, not root-caused). Candidate
  cures: install tmux 3.5+ from source/PPA in CI, or make tmux-gated
  tests skip with a loud reason when `tmux -V` < 3.5. Not yet
  root-caused individually; needs a proper worktree session, reading
  each failure against the runner environment.

- `docs/handoff.md` is still ~2.8× the ~150-line cap after archiving
  the pre-09-02 activity — the *Current state* / *In flight* ADR
  paragraphs need a summarizing pass (owner's call on what to cut).
- **z.ai (GLM Coding Plan) banked quota reset not implemented — endpoint
  unknown.** Owner pointed at z.ai's `#/manage-apikey/coding-plan/personal/usage`
  page, which now shows a "Reset Quota" card ("1 Times Unused… Manage")
  the same shape Codex and Grok already expose via `resets[]` +
  confirm-to-redeem (ADR-0031, `internal/usage/resets.go`,
  `UsageDialog.jsx`'s "Redeem" button — that plumbing is generic and
  z.ai would slot into it with no frontend change). The blocker: Codex's
  and Grok's reset/redeem endpoints (`chatgpt.com/backend-api/wham/
  rate-limit-reset-credits[/consume]`, `grok.com/.../GetRemainingResets`
  + `RedeemReset`) were reverse-engineered from real app traffic. z.ai's
  quota-*read* endpoint is public (`api.z.ai/api/monitor/usage/quota/limit`,
  already implemented, API-key auth) but no reset/redeem endpoint for it
  is documented anywhere searched (z.ai/ZCode docs, GitHub reverse-engineering
  repos — `openusage`, `pi-zai-usage`, `opencode-glm-quota`). The "Manage"
  button lives on z.ai's cookie-authenticated web dashboard, not the
  API-key surface PiCode already talks to, so it may need a different auth
  path entirely. Also: `usage.RedeemAccount` currently requires an OAuth
  cred (`okOAuth`, `usage.go:356`) to redeem — z.ai is API-key only
  (`catalog.LoginAPIKey`), so that gate needs loosening too once the real
  endpoint is known. Next step: capture the real request the "Manage"
  button fires (browser DevTools → Network → Copy as cURL) — owner chose
  to defer this rather than capture it now (2026-09-02).

- **Automations (ADR-0045):** two due automations share the active
  `auth.json` credential; `/automate` relies on the agent honouring the
  JSON fence (fallback opens the editor with the description). Closed
  by ADR-0048: list and runs are live on the feed. Closed 2026-09-02:
  the webhook is reachable through the gateway (`/-/hook/<user>/<id>`). Closed 2026-09-01: cost cap is now enforced per assistant
  message from pi's own usage events (the 30 s poll remains only for the
  session path, the timeout and a file-based fallback); runs closed by a
  restart are priced from their session file; the server fake pi runs a
  whole turn, so `TestAutomationStartRunEndToEnd` covers a real `start`
  run and the cap.
- ADR-0028: npm publish for `pi-roles` is not wired (path-triggered workflow +
  `pi-roles-v*` tag). Local path install only. Live RPC dogfood on Grok passed.
- ADR-0022's occupant scan: the cliff is gone for agents sharing a directory
  (200 in one subfolder went from 4.6s to 22ms, and the cost is now flat in the
  number of agents). Agents in *distinct* subfolders still cost one git call
  each, ~23ms, by nature — there is nothing to reuse. Agents at a worktree
  root, which is the shape the product produces, remain free.

- Machine state left by the ADR-0020 session: `~/.local/bin/picode` was
  replaced with a build of `main` (the old one is not kept in the repo — it
  predates `provision`), `picode-desktop.exe` sits in
  `%LOCALAPPDATA%\PiCode\`, and the `PiCodeDesktop` logon task is
  registered. `picode.service` is enabled and running as PID 447165, which
  still runs the *old* binary — the new one takes effect on the next restart.

- MCP GET redacts env/header values (keys only). `bearerToken` stays write-only.
- MCP **installed + zero servers** Add form is live (`docs/screenshots/mcp-named.png`). Sidebar-on-pane → `#/` checked in browser, not unit-tested.
- MCP Sign in auto-start after Add/On is UI-only (no unit test). Reuse of a running agent shares `beginAuthOn` with the short-pi tests.
- Two concurrent agents share whichever credential is **active** in `auth.json` (pi limitation; vault does not fork that).
- Token auth: ADR-0007 personal-network trust; mandatory only if exposed beyond the tailnet.
- `internal/proclock` leftover `picode.lock` after a Windows crash.
- **Ephemeral feed events are lossy by design** (ADR-0048): a dropped
  frame or a reconnecting client never replays them, and the change
  watchers (mcp/git/packages) publish only on change — a missed event
  leaves that surface stale until the next state change or a manual
  refresh/reconcile. Tolerated for badges/trees; a periodic heartbeat
  republish would be the cure if it ever bites.
- Vendored xterm.js 5.5.0 — manual upgrade (ADR-0004).
- Branch protection + CODEOWNERS — owner action on GitHub.
- tmux-gated tests skip on windows/macos CI (accepted).
- Mobile shell has no waiting card (C1 is desktop).
- Terminal **ligatures** not offered: `@xterm/addon-ligatures` needs Node `font-finder`; browser xterm is canvas-only.
- `picode provision`'s cert step issues for loopback + local IPv4 (`tlsutil.LocalNames`) and falls back to self-signed without mkcert. It does **not** yet cover what `scripts/setup-cert.sh` still owns: installing mkcert, `mkcert -install` into the Linux trust store, and the Windows CA import (that one moves to `picode-desktop.exe`, which is already elevated).
- `install_windows.go` is a stub returning an error. ADR-0020 gives Windows a real path, but through `picode-desktop.exe`, not through that file.

## Recent activity

- **2026-09-03 — docs-harness benchmark study (plan presented).**
  Studied documentation benchmarks for a public-docs harness — Diátaxis
  IA, Scalar (MIT API reference, Vue), Mintlify/Fern (llms.txt),
  Vale (MIT prose linter, ships in Mintlify CI), Remotion license
  (free ≤3 people only — default refused), HyperFrames (installed,
  0.8.27) as the default video engine, D2/Mermaid for diagrams-as-code.
  Study: docs/benchmarks/2026-09-03-docs-harness.md. Plan: theme with
  app tokens, own-capture docs-shots pipeline over a seeded fixture
  daemon, route-walking openapi.json + Scalar page, llms.txt, Vale gate
  in make ci, 3 HyperFrames tutorials. Owner directive folded in:
  **full parity** — the harness captures its own screenshots
  (`docs/screenshots/` stays agent evidence, never user docs);
  `make docs-check` re-captures and diffs in CI so UI drift without
  regenerated images fails; video compositions declare their surfaces
  and are flagged stale the same way. ADR pending owner approval.

- **2026-09-03 — adversarial review of the feed migration + one fix**
  (`fix/git-watch-workspace-scope`): `gitDirs` attached the workspace id
  to an agent's own-workPath event, so the worktree's branch could
  poison `ws.git` (the fallback pills source) until the workspace path
  changed again; the id now rides only the directory it describes.
  Accepted trade-offs documented: ephemeral events are lossy by design
  (debt), package scans grow with workspace count (code comment).

- **2026-09-03 — Providers view v2: benchmark study, no code**
  ([`docs/benchmarks/2026-09-03-providers-view-v2.md`](benchmarks/2026-09-03-providers-view-v2.md)).
  Three web sweeps: agent IDEs/CLIs (Kilo four-section IA + source badges,
  Zed `ApiKeySource` origin display, Roo profiles + lock-across-modes,
  Cursor Verify, Raycast Verify + console link + key icon at point of use),
  multi-account switchers and quota monitors (cc-switch v3.13 renders quota
  **inline on the card**, claude-swap's proactive 90 % auto-switch with
  cooldown/hysteresis and per-terminal account scoping, ccusage burn-rate
  projection, CCUM's official-limit trust layer, CodexBar, oh-my-pi's
  round-robin credentials), and credential dashboards (OpenRouter
  Prioritized/Fallback BYOK, Cloudflare `cf-aig-byok-alias` over a
  `default`, Vercel Test Key with a raw-response badge, Anthropic graduated
  expiry mail, Stripe roll-with-grace, Zapier dependent count + blast
  radius). Confirmed against pi v0.84.4 that `auth.json` is still
  `Record<string, Credential>` — **native multi-account has not landed**,
  and `pi auth check --provider X --json --no-refresh` is the Verify
  primitive we never wired. Proposal is three tabs (Accounts / Models /
  Activity) with quota inline in three honest states (live / stale / —),
  identity from `oauth/profile`, credential-source badge, dependent count
  before Sign out, Pause vs Sign out, and 7-day spend from
  `stats.byProvider[]`. **Owner's call before any ADR:** per-agent
  credential pinning vs proactive auto-switch, both of which move
  ADR-0013's single-active-slot line. No code changed.

- **2026-09-03 — Degrau 2 shipped: inbox replies switch a TUI agent to
  chat mode** (`feat/inbox-reply-switch` + `fix/form-interactive-
  normalize` + `fix/open-terminal-goto`). The respond form on an
  interactive agent confirms the trade, parks the reply
  (`RespondAndPark`), switches the agent to managed (TUI ends — user
  consented), and the delivery loop drains into the same thread
  (ADR-0053). The action returns `ActionResult.Goto` ("agentchat:<id>")
  and the shell undocks the terminal, landing on the chat watching the
  answer. Open terminal (Degrau 1) navigates out of the inbox the same
  way (`Goto: "agent:<id>"` → docked TUI). Verified live end to end
  (qa-switch free agent): confirm dialog, mode → managed, task
  `delivered`, item responded. ADR-0037 amendment records the decision
  + the send-keys benchmark research (claude-squad uses guarded
  send-keys, marked experimental; PiCode keeps it parked). A gate
  regression (normalizeView dropped the form's interactive flag) was
  caught in QA and fixed forward.

- **2026-09-03 — workspace install refused + Packages tabs**
  (`fix/packages-local-trust-tabs`): `pi install -l … --no-approve`
  answered "Project is not trusted. Use --approve"; measured on 0.84.4:
  `--no-approve` distrusts the project for the run and blocks local
  config writes even with trust.json trusting the folder; `install -l`
  works with no flag, `remove -l` in an untrusted folder needs
  `--approve`, and `--approve` never writes trust.json. `MutateArgs` /
  `UpdateArgs` pass `--approve` for the project scope (the click is the
  approval). UI: Installed |
  Marketplace tabs, installed packages as the same cards (`pkgName`).
- **2026-09-03 — package update checks ride the change feed**
  (`feat/feed-packages-updates`, phase 4 — polling→feed migration
  complete). Fleet-wide scan ticker publishes `packages.updates` on
  change; the browser applies events and polls only as fallback.

- **2026-09-03 — CI's git-guard self-test unbroken (ubuntu).** The
  self-test's throwaway repo relied on the invoking clone's
  `init.defaultBranch`: CI runners have none, so `git init` started on
  `master`, the pre-commit guard rightly refused the init commit, the
  repo stayed unborn, and two assertions failed in cascade. One-line
  fix (`git init -b main`, c503b9da), reproduced locally under null git
  config — the old script under CI conditions fails exactly like run
  33769826099, the fixed one passes. macOS/Windows redness is older,
  separate debt (see Known debts).

- **2026-09-03 — Guest TUI agent state, benchmarked** (docs-only,
  `feat/guest-agent-state`): the owner runs Claude Code, Codex, Grok,
  Antigravity, opencode in PiCode terminals and wants the sidebar's
  spinner + "needs you" for them. Study
  `docs/benchmarks/2026-09-03-guest-tui-agent-state.md` (fetched live
  from ACP docs/registry, Claude Code hooks/statusline/terminal-config,
  Codex config/app-server/non-interactive docs, opencode server/ACP,
  Grok CLI, Vibe Kanban, Crystal, Conductor): four integration levels;
  nobody credible scrapes pixels as the primary source; Claude hooks
  can POST to HTTP (`Notification` types include `agent_needs_input`),
  Codex has `notify` + an official app-server, opencode has a server
  API — and the ACP registry already lists **pi (pi-acp)**.
  Deliverable: **ADR-0056 (proposed)** — per-tool sensors → ADR-0048
  feed with pi's state vocabulary + honest `unknown`; scraping refused
  as primary; ACP/control deferred to a future ADR; ADR-0003's letter
  (no vendored SDKs) untouched. Awaiting the owner's decision on the
  ADR.
- **2026-09-03 — manager could not remove a path package**
  (`fix/pkg-remove-relative`): owner installed `pi-checklist` for the
  machine and Remove failed. Cause: pi stores path sources relative to
  the settings dir; the daemon ran `pi remove <relative>` from its own
  cwd. Fix: `pipkg.AbsPathSource(source, settingsDir)` before remove and
  update (`packageSettingsDir` picks `~/.pi/agent` or `<ws>/.pi`), and
  `existingInstallPath` resolves the same way. Verified against the
  scratch home's real relative entry.
- **2026-09-03 — the public Pages site is green again.** The red
  `pages.yml` runs since 09-02 16:30 died in `make docs`:
  `www/guide/remote-server.md` opened a code span at a line break
  (`` `sudo picode users add ``), VitePress left it unclosed, and the
  dangling backtick let `<your login> <your user>` into the Vue
  template as raw HTML — "Element is missing end tag" (18:103),
  reproduced byte-for-byte locally. The cure was already on local
  `main` (a85ff516 + ec3a515c: one code span per line) but **55
  commits sat unpushed** since 09-02, so the site was frozen at the
  09-02 15:04 deploy. Pushed `7ffe8a9f..fe421595`; run 33769826235
  green end to end; live page verified serving the fixed guide.
  Earlier red runs (08-29 → 09-02) were the dead-link gate, fixed by
  PR #1. Lesson: run `make docs` before pushing `www/**` — docs that
  land straight on `main` have no PR gate for the Pages build.

- **2026-09-03 — internal checklist, ADR-0055** (`feat/pi-checklist`):
  researched Tachyon's checklist gate/reminder/sidebar line and pi
  0.84's extension API; built `packages/pi-checklist` (tool + gate on the
  first change per task + `always` reminder capped at 3 + POST to the
  daemon + TUI render), `agents.checklist` level + `PICODE_CHECKLIST`
  spawn env (read-only → never), `agent_checklists` + routes + feed
  event, `lib/checklist.js` line projection, sidebar/mobile lines, chat
  card, settings chip; guide `checklist.md`. Owner's decisions: level
  per agent, opt-in package, sidebar + card, hold the contract.
  Measured live on glm-5.3-flash: managed RPC path (plan → read → edit
  → 2/2), the gate (edit refused → checklist → edit ok), the TUI in tmux.
  Found: `pi -p` hangs on any blocked tool_call (pi's bug, minimal repro
  in the ADR); PiCode never uses `-p` for agents. Providers were flaky
  that morning (xai at capacity, anthropic out of usage, gemini 5/min).

- **2026-09-02 — git pills and the file tree ride the change feed**
  (`feat/feed-git-events`, phase 3 of the polling→feed plan). Fleet git
  watcher publishes `git.updated`; sidebar patches in place, tree
  reloads itself. visual-review: PASS (qa-git-live.png read;
  overlayAudit ok; e2e: commit cleared the badge, new file re-raised it
  — zero fleet refetches, one tree reload per event).

- **2026-09-02 — Benchmark study: live browser preview in chat / side
  panel** (`docs/study-browser-preview`). The owner asked to render
  `agent_browser` work live in the conversation or a side panel, and
  whether that needs a PiCode package system or a pi package. Researched
  how Cursor ("screenshots and actions in the chat, as well as the
  browser window itself either in a separate window or an inline pane"),
  Devin (session Browser tab + Progress unified log + take-over), Manus
  (Cloud Browser real-time view + Take Over), Operator (*inference*),
  Antigravity (browser subagent + action-video artifacts + allow/denylist),
  Browserbase + browser-use (embeddable live-view iframes, "URL is a
  credential"), OpenHands (rrweb recordings) and the local `agent-browser`
  (`stream enable` WebSocket, `record`, `artifactVerification`) solve it.
  Convergence: activity cards in chat + live surface with take-over.
  Key gap found: PiCode's web reducer never consumes
  `tool_execution_update` and tool pills render raw JSON — no image path.
  Answer: no new package system; add a generic `details.preview`
  rendering contract to core (never the string `agent_browser`) + a
  Browser panel fed by it; v2 = auth-gated WS proxy for `stream enable`.
  ADR is the next step (see *Next up* 0).
- **2026-09-02 — the inbox refusal to a TUI agent offers Open terminal**
  (`fix/inbox-open-terminal`). Replying to an agent-sourced item whose
  agent runs in a TUI is refused by design (ADR-0037/0006), but the
  refusal was a dead end. The item's view now shows an Open terminal
  action when the agent is interactive (same `h.AgentDeliverable` gate
  the reply uses); firing it starts the TUI server-side via
  `apps.Host.OpenAgentTerminal`, wired from `Deps.openAgentTUI`
  (handleAgentOpen's body extracted, sentinel errors keep the HTTP
  mapping identical). Works from the desktop and mobile inbox.
  Deployed; verified live on the owner's item + a QA item:
  action renders only when interactive, click starts/confirms the
  tmux session (mode → interactive), detail closes by design after
  acting. Follow-through (same day): the action now returns
  `ActionResult.Goto` ("agent:<id>") and the shell's AppSurface hands
  it to the host via `onGoto` → **leaves the inbox** and lands on the
  agent's tab with the TUI docked (owner decision: acting from the
  inbox must not dead-end there; mobile has no terminal surface and
  ignores the goto). visual-review: PASS (qa7 + qa8-landed.png,
  card 5/5). **Parked (owner's call, from the send-keys benchmark
  research):** Degrau 2 — consented mode-switch delivery (queue the
  reply + switch interactive→managed; ADR-0053 keeps the thread; on
  switch, the agent tab undocks the terminal and flips to chat; the
  TUI reopens later on the same session; trigger: owner hits the
  refusal again in dogfood); Degrau 3 — guarded send-keys delivery,
  parked with ADR-0002's rejection (claude-squad precedent noted:
  SendPrompt = keys + 100ms + TapEnter, autoyes experimental, per-CLI
  content sniffing).

- **2026-09-02 — MCP live status rides the change feed**
  (`feat/feed-mcp-events`, phase 1 of the polling→feed plan). Fleet-wide
  watcher publishes `mcp.updated`; the Mcps panel follows events and
  polls only as the feed-down fallback. visual-review: PASS
  (qa-mcps-empty.png + qa-mcps-live.png read; overlayAudit ok; e2e:
  live-file flip → exactly 1 request, badge Idle→Live→Failed).

- **2026-09-02 — Work head = Inbox head on the phone**
  (`fix/mobile-head-parity`): `.m-screen-head` takes the Inbox numbers
  (12/8 padding, hairline, margin 0); `.m-inbox .ft-head` matches and
  cancels the desktop 880px `margin-top: 6px`; the 32px segment rule is
  scoped to `.m-agent-state`. Both heads measure 57px, controls 36px.

- **2026-09-02 — LooksWorking anchors to pi's spinner frames**
  (`fix/looks-working-spinner-frames`). The working check grepped the
  pane tail for the substring "working": an idle agent whose last
  reply mentioned the word (this project's own docs/logs do, constantly)
  spun in the sidebar and showed "Working in the terminal" in the
  composer — the owner caught mobile doing it at 20:38 while pi sat at
  its prompt. The indicator is now what pi actually renders: a pane
  line whose first non-space rune is one of the braille spinner frames
  (⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏ — pi 0.84.4 DEFAULT_FRAMES; re-check on pi upgrades).
  Immune to prose and to a renamed working message; a custom indicator
  without braille degrades to a benign false-negative. Decision table
  with real captured tails (working / idle / the false-positive prose)
  pins it. Deployed (PID 412915); verified live: mobile working →
  detected via ⠏ while its pane is full of the word, agente-auto idle →
  not listed. Probes one open design question (unfixed): inbox replies
  to interactive agents are refused by design (ADR-0037/0006 — the TUI
  picks up no follow-ups); options range from an "Open terminal" action
  on the refusal note to send-keys delivery — owner to decide if/when
  it itches.

- **2026-09-02 — black strip under terminals** (`fix/term-viewport-strip`):
  owner spotted a near-black band at the bottom of every terminal. Cause:
  xterm 6 sets the theme background on `.xterm-scrollable-element`, not
  `.xterm-viewport`, which keeps xterm.css's `#000`; the row-remainder
  showed it. Fix: `.xterm .xterm-viewport { background-color: transparent }`
  in `app.css` (surface carries the theme colour). Pixel-verified in the
  scratch shell.
- **2026-09-02 — Termux layout for the mobile keys** (`feat/mobile-keys-termux`):
  owner sent Termux screenshots — "vamos fazer o nosso igual". `KeyBar`
  is Termux's 2×7 grid (`ROWS`), flat cells on `--bg-base`,
  `grid-template-columns: repeat(7, minmax(0, 1fr))`; ⌨/× dropped
  (tap the terminal for the keyboard; header icon toggles). Sticky
  modifiers, viewport lift and hardware heuristic unchanged.
- **2026-09-02 — the chat shows when a TUI agent is working**
  (`fix/tui-working-in-chat`). The chat's event socket exists only for
  managed agents, so an interactive agent read as idle there no matter
  what pi was doing (owner: sent a message in the TUI, switched to the
  chat, no working sign). The server already scraped panes and
  published `agent.tui` (ADR-0048) — sidebar/dashboard spun on it, the
  chat didn't. When the selected agent is interactive and its pane is
  working, the composer now shows a "Working in the terminal" row with
  an Open button that docks the TUI; no fake streaming, no Stop. UI
  only; the server pieces shipped with ADR-0048 (watch tick 3s).
  Deployed and verified live on agente-auto: send-keys → row appears
  within one tick with the spinner, Open docks the TUI mid-work, row
  clears when the pane's working line ends; overlayAudit ok.
  visual-review: PASS (qa4-row-live.png + qa4-open-docked.png,
  card 5/5).

- **2026-09-02 — Mobile terminal keys to the benchmark (ADR-0044
  amendment)** (`feat/mobile-terminal-keys`): library search found
  nothing terminal-aware (simple-keyboard & co. are full QWERTYs), so
  the bar matches Termux/Blink/terminal-web: `lib/termSticky.js`
  (createSticky: arm/apply/applyKey/subscribe, control bytes, modified
  arrows, 5 s expiry; tested), wired on the ShellTerm entry
  (`entry.sticky`, filters `term.onData`); `KeyBar` is one scrollable
  row (`KEYS`), Ctrl/Alt light up (`aria-pressed`); `Terminal.jsx` sizes
  the screen to `visualViewport` (lift above the iOS keyboard, refit)
  and hides the row when focus comes with no viewport shrink (hardware
  keyboard). iPhone verification is the owner's: keys never open the
  keyboard, ⌨ does, Ctrl then c interrupts, bar above the keyboard.
- **2026-09-02 — Mobile key bar** (`fix/mobile-keybar-focus`): owner:
  every key opened the iPhone keyboard. `sendKey` refocused xterm
  unconditionally; now it refocuses only if the terminal host already
  held the focus, and a ⌨ key (`onType`) is the deliberate way to open
  the keyboard. `KeyBar` regrouped (escapes/chords · arrows/symbols),
  44px keys, close at the end of row two. Not browser-verified here
  (owner tests on the phone).
- **2026-09-02 — the composer's "+ New" starts a fresh session for real**
  (`fix/new-session-fresh-start`). The button cleared the pointer and
  then lost it three ways: the restart spawned in `wk.Path` (free
  agents: the `ws_free` sentinel → "That folder doesn't exist" — the
  toast the owner screenshotted), ADR-0053 adoption re-adopted the
  abandoned thread, and the list's newest-fallback re-selected it.
  Fixes: `restartSameMode` resolves `store.AgentCwd` (+MkdirAll);
  `store.SealPendingAgentSessions` closes the adoption window — and
  moves attribution to the path row, because a plain UPDATE collided
  with the sibling path row on UNIQUE (agent_id, session_path)
  (probed live on agente-auto's DB); the list fallback now surfaces
  only a pointer the loop itself healed; the web pane clears
  optimistically. Decision table pinned (resume→list, New stopped→
  fresh, next spawn mints fresh ≠ abandoned id, free agent + TUI open,
  free agent + chat running). Deployed and verified live on
  agente-auto: chat clears, no toast, old sessions still listed,
  tmux respawned in the agent's own folder; overlayAudit ok.
  visual-review: PASS (qa2-before/qa2-after-stable.png, card 5/5).

- **2026-09-02 — Mobile Inbox without a title** (`fix/mobile-inbox-no-title`):
  owner: the Inbox was the only main tab with a header. CSS hides the
  AppSurface icon + `.ft-title` under `.m-inbox`; the filters stay as
  the sticky head; the section keeps its aria-label.
- **2026-09-02 — Mobile shell in Safari's own tab** (`fix/mobile-safari-headers`):
  owner's iPhone screenshots — heads clipped under the status bar,
  keys button over the TUI. Sticky heads lose the negative margin and
  the negative `top`; the screen hands its top padding to the head via
  `:has()` (`mobile.css`); the terminal's keys toggle is a header
  button (`.m-keys-btn`), `.m-fab` removed. Measured in Chromium at
  390×480: head border box at 0 at rest and when scrolled.
- **2026-09-02 — Content-Security-Policy (ADR-0052 amendment)**
  (`feat/csp`): `internal/server/csp.go` — the app shell's inline theme
  bootstrap is allowed by sha256 hash computed from the served
  index.html (no nonce); `script-src 'self' 'wasm-unsafe-eval' <hash>`,
  inline styles allowed (React), `img-src https:` (unpkg icons),
  `connect-src` names the request host's ws/wss; assets carry no policy
  (excalidraw's worker uses `new Function`). `/pair` and gateway pages:
  `PageCSP` (no scripts, no framing). `securityHeaders` wraps the UI
  handler. Test skips without a built UI; run after `make build`.
- **2026-09-02 — `make desktop-restart` guardrail after the tray/VM outage.**
  Swapping the Windows exes, this session killed the tray and relaunched it
  with `picode-desktop --tray &` from a WSL tool shell; the process died with
  the shell, the keepalive died with it, and the 60s WSL idle timeout
  reclaimed the VM — server down, open tmux sessions lost. Fix: the swap is
  now one audited command, `scripts/desktop-swap.sh` (kill → copy → re-register
  host → `schtasks /run /tn PiCodeDesktop` → verify tasklist), wired as
  `make desktop-restart` and listed in AGENTS.md's commands table. The script
  header and the make help state the rule: never background a Windows exe
  from WSL. Ran once for real after the fix: exes swapped, tray up, bootId
  unchanged (no VM bounce).
- **2026-09-02 — Chrome extension Track C, the actuator (ADR-0054)**
  (`feat/ext-track-c`). Send with "Let the agent act on this page" →
  `[browser-act]` prompt intro → settle watcher parses the last
  ```picode-act block → `act_batches` (migration 021) → panel polls
  `GET /api/extension/act/next` through the native host (origin-gated
  claim, blocked stays pending) → per-origin grant → visible
  step-by-step execution (`chrome.scripting`, one action per injection,
  highlight + native setters) → outcomes POST back as one more watched
  turn; 3-round cap, Stop, 10-min expiry. Parser/lifecycle/routes
  table-tested; panel states captured (`ext-act-grant/-acting/-actdone`).
  (Actuator ADR numbered 0054 — 0053 went to session isolation.)

- **2026-09-02 — Session isolation follow-through** (`fix/session-followthrough`):
  owner filed one bug with three faces on a freshly created agent: the bar
  opened with another agent's context/spend/cache, the picker said "No
  sessions yet" after it had chatted, and opening its TUI started a new
  session (then a chat send killed the TUI's work). Root causes: the picker
  never saw ADR-0040's private dir (so the lazy `session_path` backfill
  never fired), every run-mode switch minted a competing `--session-id`,
  and the desktop `/status` fetch omitted `?agent=` (server falls back to
  the workspace's first agent). Fix = ADR-0053: adopt-at-spawn + picker
  union + agent-scoped bar, table-tested; verified in the browser against a
  scratch instance with a two-agent fixture (mobile's bar bare, alpha's bar
  showing its own $23.88). Also fixed here: a pre-existing cross-line code
  span in `www/guide/remote-server.md` broke the Pages build on `main`
  (`make docs` failed before and after my changes until that span was
  closed — reproduced on main first, fix committed in this branch).

- **2026-09-02 — Chat follows an automation's turn** (`fix/chat-follows-automation-run`):
  owner: run "Running" but the chat showed nothing. Three causes in the
  desktop: the panel effect gated on `selected` (the *workspace*), so a
  free agent never got its socket; a closed panel for the same agent
  blocked reconnecting (onclose now nulls `panelRef`, the effect checks
  `readyState`); and a turn nobody typed here (automation, another tab)
  had no prompt bubble — `foreignTurnRef` reloads the session file on
  snapshot(streaming) and on settle so the thread mirrors the file, in
  order. Diagnosed on the scratch with a raw WebSocket and a patched
  `window.WebSocket` (the app opened no socket at all).
- **2026-09-02 — Terminal open vs a run in flight** (`fix/automation-run-guard`):
  the owner clicked the agent's terminal icon during a message run; the
  open path's `Runtime.Stop` killed the managed process and the run
  hung. `deps.automationRunOn` (ManagedAgent.Observed) → 409 on the four
  interactive-open paths; `runWatch.exited(expected)` fails the run as
  `reasonStopped` unless the run itself asked (`letGo`). Unit test.
- **2026-09-02 — Message runs deliver (ADR-0045 amendment)**
  (`fix/automation-message-delivers`): owner ran a message automation,
  got "Done" and an untouched agent. `messageRun` starts an idle agent
  (`startManaged`, its own session), sends a prompt (SendTurn makes it a
  follow-up when busy), observes with `runWatch` and stops it on settle;
  `keepAlive` for an already-running agent (cost cap aborts the turn
  only); decision table: pi needed unless the agent runs, busy when
  another run observes the agent; `ManagedAgent.Observed()`. E2E
  `TestAutomationMessageRunEndToEnd` against the fake pi.
- **2026-09-02 — Automation detail facts** (`fix/automation-detail-facts`):
  owner: a notify URL saved but nothing on the detail. Facts list gained
  Messages/Runs in (agent + workspace via `workspaceOfAgent`), Model (or
  "pi's default"), Notifies (host + copy), limits; `whenLine(a, agents)`
  names the agent (List needed the `agents` prop — a blank root until it
  had it); the secret box's Done is a primary button.
- **2026-09-02 — Automation editor layout** (`fix/automation-editor-layout`):
  owner: "componentes se amontoando, péssima UI/UX". The what-it-does
  block is a labeled `.auto-grid` (Workspace → Agent, or Workspace →
  Provider/Model/Thinking); message mode lists the chosen workspace's
  agents via `agentsOf` (free agents under "No workspace") with their
  model, and an existing automation's workspace is derived from its
  target agent; `lib/providers.js usableProviders` filters to signed-in
  providers (keeping a selected one, marked) and `ConfigFields` uses it
  too, so New agent matches.
- **2026-09-02 — Automations notify a channel (ADR-0045 amendment)**
  (`feat/automation-notify`): `notify_url` on automations (migration
  020, validated http(s) in the store), `automate.BuildNotify` (Slack
  shape, clipped summary), `runner.notifyOut` in
  `internal/server/automations_notify.go` (background, one retry on
  network/5xx, `automation.notify` event), hooked into `settled` and the
  failure path of `finish`; editor field + "Notifies" tag; guide section.
- **2026-09-02 — Automation webhooks through the gateway (ADR-0045
  amendment)** (`feat/gateway-webhook`): `POST /-/hook/<user>/<id>` on the
  gateway (no identity, per-peer limit, member check, Authorization
  passed through, cookies dropped); the automation view carries
  `webhookUrl` computed from the daemon's situation (origin / public URL
  / gateway form, via `os/user.Current`); the detail page shows it; guide
  recipes for GitHub Actions, Sentry and cron.
- **2026-09-02 — Track D, public access (ADR-0052)** (`feat/public-access`):
  gateway identity chain (tailnet whois → signed cookie → login page),
  `plainListen` + `trustedProxies` (last XFF hop from a trusted CIDR
  only), Google OIDC (discovery, PKCE, nonce, JWKS RS256) and GitHub
  OAuth (`<login>@github`) in `internal/gateway/oidc.go`, signed session
  cookie (`session.go`), per-peer limiters on `/-/auth/*` and `POST
  /pair`, `/-/` routes and a login page, extra security headers;
  `picode gateway oidc set|unset`, `--plain` (alias `--insecure-listen`),
  secrets in `gateway.secret.json`; the SPA shows **Sign in** on a 401
  that carries `login`. D.2: `ContainerSteps` + `install.ContainerUnitFile`
  (nspawn, private users, limits, host networking), `--container`/`--remove`.
  Tests: full login round trip against a fake OIDC provider, claim
  checks (aud/exp/nonce/iss/verified/unknown login), forged cookie,
  untrusted XFF, logout, limiter, GitHub spelling, unit text, steps.
- **2026-09-02 — ADR-0053 deployed to the owner's service; the "not
  fixed" report was a stale binary.** The installed
  `~/.local/bin/picode` predated the 17:13–17:14 fixes (16:49 build, 0
  hits for `ResolvePendingAgentSession`) while the on-disk UI was
  fresh — so the chat still read "No sessions yet" for an agent whose
  sessions were visible in the TUI, and the status bar fell back to
  the workspace's first agent (grok's $23.88 / 100% cached shown on
  the brand-new `mobile` agent). `make deploy`; no code changes.
  Verified live on `mobile-6bf740`: the picker returns the agent's two
  private-dir sessions and adopts the TUI's pending `6903ca7b…` as
  current; `GET /status?agent=` returns that agent's own bar ($0.02,
  4.3% of 1M) instead of grok's. visual-review: UNVERIFIED (API-level
  checks only).
