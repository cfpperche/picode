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
- **ADR-0041** session observability dashboard: the no-tabs-open main pane now shows spend/activity/fleet stat tiles + spend-by-provider (Today/7d/30d/All), replacing both the bare "No agents yet" copy for that case and the branch's own earlier rejected `HomeView` (workspace/agent list) attempt. `GET /api/sessions/stats` bucket at message granularity via `entryTS`, not file mtime. No chart library; hand-rolled SVG mirrors the `GitGraph.jsx`/`lib/gitgraph.js` split.
- **ADR-0047** Web Push: `internal/push` (stdlib VAPID + RFC 8291), `store` migration 018, `/api/push/*`, `PushPrefs.jsx`; presence-aware (host online → no push); real-device dogfood pending (needs the mkcert cert on the phone; iOS needs Home Screen install).
- **ADR-0046** one modal primitive: `components/ResponsiveDialog.jsx` (Radix dialog ≥720px, Vaul sheet below, `Alert` for confirms); `.dlg.dlg-sheet` CSS; `lib/dialogPolicy.test.js` refuses raw dialog/drawer imports outside it (allowlist: `Palette`, `Hotkeys`).
- **ADR-0044** mobile shell v2 — supervision console (amended same day: no header; Work tab = workspaces / free agents / terminals; `#/term/<id>` screen with key bar). `web/src/mobile/` is now screens (`Now`, `Inbox`, `Agents`, `Agent`, `More`) + hooks (`useHashRoute`, `usePoll`, `useFleet`, `useAgentSocket`) + components (`TabBar`, `ScreenHeader`, `StateChip`, `NeedsYouCard`, `AgentRow`, `StatStrip`, `CreateSheet`) over pure libs `lib/mobileRoutes.js`, `lib/agentEvents.js` (reducer of the agent WS stream; desktop `handleEvent` untouched), `lib/needsYou.js`, `lib/createSubmit.js` (lifted from `submitNew`). Server: `agentView.streaming/waiting/dialog` from `Runtime.Get(id).Snapshot()`; `AppSurface` gained `initialPath`; `summarizeArgs` moved to `lib/toolArgs.js` (re-exported). The waiting card exists on mobile now — the two "Mobile has no waiting card" lines below are history. Push is the next ADR (phase 2 of the approved plan: Web Push/VAPID in Go stdlib, presence-aware).
- **ADR-0042** dashboard v2: same scan now yields `byModel`, `byWorkspace` (server labels cwd → workspace via `canonDir`), `tokens` (+ cache hit), `tools` (top 8), `turns` (assistant/user/errors/aborted/compactions), `topSessions` (top 5, name/cwd/lastAt, never preview); `Series[].turns`. Server memoises per range behind `session.Fingerprint` (stat sweep: count/size/newest mtime). UI: 4 tiles (Sessions + Fleet strip from `workingIds`/`waitingId`), `DailyChart` (`lib/barchart.js`), `RankedBars` (folds tail into "N more"), `TokenBar`, Reliability facts, `TopSessions`; 60 s auto-refresh + 30 s tick, paused when hidden. Not built (refused in the ADR): projection/burn rate, LOC/commits, latency, per-agent context % on the home.
- **ADR-0043** Chrome extension Track A (sensor) and Track B (devices + Windows console host): `ext/` MV3; `GET/POST /api/extension/*`; `make desktop` also builds `picode-nmh.exe`; side panel pings `kind=extension`. Isolated Chromium unchanged. Chrome-only. Not an App, not a pi package.
- **ADR-0045** Automations: `#/automations` (user menu, palette). Scheduler goroutine in the daemon (`internal/automate`, 1-min tick, deterministic jitter, single catch-up, boot reconcile) + webhook `POST /api/automations/{id}/fire` (bearer secret, 64 KB). One agent per automation, fresh session per run; `RunObserver` on the managed agent; cost cap / 2 h watchdog; runs table; Inbox `sourceKind: automation` (migration 017 rebuilt `inbox_items`). Decision table in `internal/server/automations_run.go`, tested row by row. v1 only: no templates, no `/automate`.
- **ADR-0049** authentication: `internal/auth` gate wraps the mux (`Deps.Auth`, nil = ungated in tests); modes off/remote/all (`auth.mode`); `auth_sessions` + `auth_pairings` (migration 019); install token `<data>/token`; routes `/api/auth/*` + `/pair`; `picode pair`, `picode token [rotate]`; Devices section in Preferences → Server; pairing screen on 401; QR carries a pairing link; native host and `pi-inbox` send the bearer; `PICODE_DATA` in spawn env. Roadmap: `docs/design/remote-modes-roadmap.md` (Tracks B–D next).
- **ADR-0048** change feed: `events` is the change log (every store mutation appends in-tx; `Store.OnEvent` after commit; invariant test enumerates mutators), `internal/feed` (replay under the subscribe lock, `ErrReset`, ephemeral id 0), `GET /api/events` (SSE `hello`/`change`/`reset`, `Last-Event-ID` or `?after=`), push consumes the feed (`Notifier.OnEvent`), presence `OnChange` → `device.online`, runtime `OnState` → `agent.state`. Web: `lib/feed.js` (cursor in sessionStorage, bootId kick), `lib/feedReducers.js` (fleet / inbox / automations / runs; `null` = refetch), desktop sidebar + badges + AppSurface + Devices + Automations and mobile `useFleet` / `usePoll` / inbox loop on the feed; timers tick only while the feed is down.
- **ADR-0045 v2** `/automate` + templates: `lib/automateDraft.js` (prompt, fence/object parser, command detection), `lib/automationDraft.js` (read-once `sessionStorage` handoff), `automate.Templates()` + `GET /api/automations/templates`, Suggested cards with category chips, *Start from template…* in the editor, "Drafted by / From template" origin line. Turn correlation is client-side (`automateRef` + `agent_settled` + `lastAssistantText`); no server change for `/automate`.

## In flight

**ADR-0045 Automations — merged to `main` and shipping.** Scheduler,
webhook, notify channel, gateway hook route, `/automate` + templates;
editor/detail UI fixes and the message-run amendment landed 2026-09-02.
Owed: acceptance runs listed under *Next up* (owner's sudo).

**ADR-0043 extension — Tracks A and B merged** (`78131e53`, `25ec36a5`); the
Windows console host (`picode-nmh.exe`) and the `--parent-window` fix are
dogfooded on the owner's Chrome. Track C (actuator) is next, in a worktree.

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

1. **Remote modes — acceptance runs owed** (owner's sudo): Track C on a real second account (`sudo picode gateway install`, `sudo picode provision --user demo --shared`, `sudo picode users add <login> demo`); Track D.2 (`sudo apt install systemd-container debootstrap`, `sudo picode provision --user demo --shared --container`); Track D.1 with a real GitHub OAuth app behind Caddy on a public name. Then decide on Track E (SaaS: signup, quotas, billing, VM per client) — the roadmap sketches it.
2. **Gateway CSP** — the SPA's inline theme bootstrap needs a nonce before the gateway can send a content-security policy.
2. **Feed follow-ups (ADR-0048)** — git changes as events for the file
   tree and the sidebar's branch line (would retire the last refetches:
   workspace create / clone, config PATCH); outbound webhooks fed by the
   log; a fake pi fixture that emits `usage.totalTokens` so the context
   bar can be tested end to end.
2. **Automations** — `pi-automate` only if `/automate`'s fence parsing
   proves flaky. Notify URL, webhook recipes and the gateway hook route
   shipped 2026-09-02.
**ADR-0043 Track C** (actuator on the current tab) after Track B merges.
Track D/E stay parked.

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

**Multi-runtime TUIs in terminals** (owner direction, research in flight in
another session): the ADE will host TUIs from several agent runtimes — in
**terminal surfaces only**; agents stay **Pi-only for now**. The unified
TermSurface/ShellTerm path is the substrate (any tmux session renders the
same). Known landmine for that research: tmux's `extended-keys-format` is
server-wide — we force `xterm` (modifyOtherKeys) so Shift+Enter survives,
but some TUIs prefer csi-u/Kitty; per-session key format may be needed.

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
- Vendored xterm.js 5.5.0 — manual upgrade (ADR-0004).
- Branch protection + CODEOWNERS — owner action on GitHub.
- tmux-gated tests skip on windows/macos CI (accepted).
- Mobile shell has no waiting card (C1 is desktop).
- Terminal **ligatures** not offered: `@xterm/addon-ligatures` needs Node `font-finder`; browser xterm is canvas-only.
- `picode provision`'s cert step issues for loopback + local IPv4 (`tlsutil.LocalNames`) and falls back to self-signed without mkcert. It does **not** yet cover what `scripts/setup-cert.sh` still owns: installing mkcert, `mkcert -install` into the Linux trust store, and the Windows CA import (that one moves to `picode-desktop.exe`, which is already elevated).
- `install_windows.go` is a stub returning an error. ADR-0020 gives Windows a real path, but through `picode-desktop.exe`, not through that file.

## Recent activity

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
- **2026-09-02 — Devices footer spacing + centering.** List rows
  unchanged. `.devs-foot` adds 16px spacer + divider under the last
  device; Pair / Copy link and the extension note are centred. Empty
  state untouched. visual-review: PASS (devices-foot.png + overlayAudit
  ok; card 5/5).
- **2026-09-02 — `picode gateway uninstall [--purge]`**: the owner asked
  how to remove the gateway completely, not just disable it. Stops and
  removes the system unit; `--purge` deletes `/etc/picode` and the
  shared binary unless a member unit still runs it. Members are never
  touched (their own `picode uninstall`).
- **2026-09-02 — Track C, shared tailnet box (ADR-0051)**
  (`feat/shared-server`): `internal/gateway` — config (`/etc/picode/
  gateway.json`, login → Linux user), identity via `tailscale whois`
  (60 s cache, non-tailnet refused), backend from the member's own
  `server.json`/`token`, reverse proxy with header hygiene and
  `FlushInterval: -1`, first-visit auto-pair (mint with the daemon's
  token, redirect to `/pair`), branded 403/503/502 pages. CLI: `picode
  gateway [install|status] [--insecure-listen]`, `picode users`,
  `provision --shared` (`MemberSteps`: account, linger, binary, env
  drop-in, member's own pass via `runuser`, loopback health). Daemon
  side: `PICODE_AUTH_MODE` and `PICODE_PUBLIC_URL` env fallbacks;
  `share.Diagnose` treats a proxied daemon as trusted; no trust listener
  when insecure. Scratch seams: `PICODE_GATEWAY_FAKE_IDENTITY` and
  `PICODE_GATEWAY_FAKE_HOMES`, only with a loopback plain listener.
  Not yet done: the two-real-accounts acceptance run (owner's sudo).
- **2026-09-02 — GitHub Pages frozen since 2026-08-29**
  (`fix/pages-dead-links`). VitePress 1.6.4 fails the build on a bare
  `https://localhost:8445` in `www/guide/getting-started.md` (dead-link
  check). Four Pages runs failed in ~20s; the live site stayed on the
  last green artifact (2026-08-26), so `/guide/mcp` 404'd even though
  `mcp.md` is on `origin/main`. Fix: wrap the URL as inline code,
  `ignoreDeadLinks` for localhost / 127.0.0.1, `make docs` in `make ci`
  (ubuntu job) so Pages is not the first gate. Merged as PR #1.

- **2026-09-02 — provision reads systemd from any shell**
  (`fix/provision-session-env`): from a PiCode terminal (no
  `XDG_RUNTIME_DIR`), the service row said "present but not enabled" and
  its fix "still fails" — the check's `systemctl --user` could not reach
  the manager while the fix (install.Run, which fills the session env)
  could. provision's `run`/`output` now carry `install.SessionEnv()`; a
  check that still cannot reach systemd says so (blocked) instead of
  claiming the unit is off. `tailscale cert` errors now carry the action:
  the account error → enable HTTPS Certificates in the admin console;
  access denied → `tailscale set --operator`.
- **2026-09-02 — Track B.2, the Tailscale certificate (ADR-0050
  amendment)** (`feat/tailscale-cert`): `internal/tlsutil/tailscale.go`
  issues `tailscale-cert.pem`/`-key.pem` for the MagicDNS name via
  `tailscale cert`, `LiveConfig` picks the leaf by the requested name,
  `KeepTailscaleCert` renews daily (logs a failure once, with tailscale's
  hint); `provision` row `tailnet-cert`; `share.Diagnose` lists the name
  as a `trusted` target picked first after a public URL, the trust page
  is skipped for it (drawer and `pairLinks`). On this WSL box issuance
  needs `sudo tailscale set --operator=goat` once — not run by the agent.
- **2026-09-02 — Bind keeps loopback (ADR-0050 amendment)**
  (`fix/bind-keeps-loopback`): the owner chose *Tailnet only* and the
  tab hung: the new bind on the same port could not bind-new-first
  (0.0.0.0 overlaps), the loop reverted, and the UI had already moved to
  the tailnet address (unreachable from the Windows browser, and with
  no cookie there). Now an outside bind also listens on 127.0.0.1; host
  changes shut the old listener, bind the new, restore on failure; the
  tab keeps a local origin; the options read "Tailnet and this machine".
- **2026-09-02 — Track B.1, a PiCode on a tailnet server (ADR-0050)**
  (`feat/tailnet-server`): `server.host` and `server.public_url` are
  settings with routes (`PUT /api/server/host` rebinds, `/public-url` is
  advisory) and a Reach-this-server block in Preferences → Server that
  suggests the tailnet name/IP; `server.json` gained `bind` and
  `publicUrl`; `picode install --env` writes a systemd drop-in; `picode
  update` verifies `SHA256SUMS`; `provision --dry-run` gained pi/tailnet/
  reach rows (no `doctor` verb — same engine); `remote.json` +
  `extension-install --server --token --ca` point the native host at
  another machine, `pi-inbox` reads `PICODE_URL`/`PICODE_TOKEN`. Bug fixed
  on the way: the install token is now a real token session, so the
  extension shows on Devices. Guide: `www/guide/remote-server.md`.
  Deferred to B.2: `tailscale cert` by SNI (see ADR-0050).
- **2026-09-02 — Sidebar no longer duplicates a just-created terminal**
  (`feat/zai-reset-and-terminal-dup`). Owner screenshot: two identical
  "Terminal 4" cards after clicking "+". Root cause: `store.CreateTerminalIn`
  publishes `terminal.created` to the change feed (deduplicated by
  `applyFleet`) before `handleCreateTerminal` finishes `ensureShell`
  (spawns the tmux session) and responds — so the SSE subscriber's
  `setTerminals(next.terminals)` in `App.jsx` almost always lands *before*
  the POST response, and `createTerminal`'s own `setTerminals((cur) =>
  [...cur, page])` then appended the same terminal a second time,
  unconditionally (unlike `openTermTab`, which already replaces-or-appends
  by id). Fixed by skipping the append when the id is already present.
  Verified on a scratch instance (`:8471`) via `agent-browser`: created two
  terminals back to back, sidebar showed exactly one card each
  (`after-fix.png`). `make fmt-check vet test test-js build` all green.
  z.ai's own "Reset Quota" ask from the same session is a separate open
  question — see *Known debts* above.

- **2026-09-02 — /pair got PiCode's own look** (`fix/pair-page-branding`):
  the confirm and error pages were an unstyled system page with no
  branding — owner: "pessima UXUI, sem branding do picode". Rewrote
  `pairPage`/`pairPageTemplate` in `internal/server/auth.go` as a
  dark-first card matching the app's tokens (bg/border/accent, the π
  wordmark, "The browser is a door, not a cage." footer), with
  `prefers-color-scheme: light` for a light OS. The heading and button
  now name the device (`presence.Label`): "Pair this iPhone", "Pair this
  Browser" for an unrecognized UA. No React here — this loads before any
  cookie exists, so the CSS is inlined rather than shared with app.css.
- **2026-09-02 — /pair confirms on POST** (`fix/pair-confirm-post`): the
  iPhone camera prefetched the pairing link, spending the one-time code
  before Safari opened it ("This link was already used"). GET /pair now
  renders a one-tap page and only the POST pairs; a browser that already
  holds a session goes straight in; error copy points at Devices.
- **2026-09-02 — One pairing QR, tailnet first** (`fix/pair-one-qr`):
  owner scanned two different QRs (Devices and Open on phone) and the LAN
  address did not reach the phone. Devices → Pair a device now opens the
  phone drawer (`openPairDrawer`, one QR with the LAN/Tailnet choice);
  `share.Diagnose` prefers the official tailnet address (LAN needs the
  same Wi-Fi, the Windows firewall rule and WSL mirrored networking);
  each target carries a `note` saying what it needs; the "address" check
  no longer claims the phone can reach it. Trust listener: 8470 first,
  else a random port — the firewall rule only opens 8445/8470.
- **2026-09-02 — Devices is one surface** (`fix/devices-one-surface`):
  owner asked why the user menu and Preferences both had "Devices". The
  presence page (`#/devices`) now lists paired sessions with liveness
  joined by session id (`PingSession`); Preferences → Server keeps the
  pairing rule and the token (`AccessSection`). `DevicesSection` removed.
- **2026-09-02 — Pairing link reachable from the phone** (`fix/pair-url-reachable`):
  a request on loopback built `https://localhost:8445/pair?…`, which Safari
  on the phone cannot open. `pairLinks` now takes the share report's
  reachable address and, with mkcert, wraps the QR in the trust page.
- **2026-09-01 — ADR-0049 authentication, Track A of the remote-modes
  roadmap** (`feat/auth-core`): gate, modes, pairing, install token,
  Devices UI, CLI, client updates. Dogfood on a populated DB copy (note:
  the copy carries `server.port`, override it or the scratch collides
  with :8445): loopback auto-pairs (cookie), LAN without cookie → 401 +
  pairing screen, bearer works, foreign Origin and unknown Host → 403,
  pairing link → cookie → app.
- **2026-09-01 — Hotfix: empty sidebar after the feed-debts deploy**
  (`fix/boot-fleet-load`, `1688619`). The pattern-driven swap of
  post-mutation `loadWorkspaces()` → `refreshFleetFallback()` also hit
  the two **boot** loads; with the stream open before boot finished the
  desktop showed "No workspaces yet" (data untouched — the API and the
  events log proved it). Lessons: review every site of a mass
  replacement; dogfood boot paths against a populated database, not the
  scratch instance's empty one.
- **2026-09-01 — ADR-0048 debts closed** (`fix/feed-debts`): tmux
  watcher (`agent.tui`), per-message `agent.usage`, `device.offline`
  ticker, run mode on `agent.status` at all ten start sites, health
  watch idles at 20 s with the feed up, 17 redundant `loadWorkspaces()`
  → `refreshFleetFallback()`. Lesson: chaining `make build` behind a
  `grep` masked a vite failure and a broken commit went in for two
  minutes — verify by exit code, never by grep.
- **2026-09-01** — **Hotfix: mobile shell crashed to a blank screen on
  every Inbox visit**, on branch `fix/appsurface-feed-crash`. Owner:
  "app no web quebrado nao abre .. erro no console". The concurrent
  ADR-0048 change-feed merge (`a7712ee0`) left
  `AppSurface.jsx`'s feed-subscription effect referencing `app.id` —
  `app` was never declared in that component (only the `appId` prop
  exists). On desktop this went unnoticed: a coincidental `id="app"`
  element elsewhere in the page made the browser's legacy named-element-
  as-global-property behavor resolve `app` to that DOM node instead of
  throwing (`app.id` silently evaluated to the string `"app"`, quietly
  breaking the auto-reload-on-touch feature but not crashing). The
  mobile shell has no such element, so the reference genuinely threw
  `ReferenceError: app is not defined` on every render of the Inbox
  (list or item screen) — with no error boundary, React 18 unmounts the
  whole tree, emptying `#root`. Confirmed reproducible on a byte-
  identical isolated build (not a caching artifact) via an
  `--init-script` early error listener (agent-browser's own `console`/
  `errors` capture missed it — too early in the page lifecycle). Fix:
  both references now read `appId`. Verified: Inbox list and item
  screens both mount clean on mobile with zero captured errors.

- **2026-09-01 — ADR-0048 change feed** (`feat/change-feed`): durable
  event log + SSE with replay, push and presence on the feed, desktop and
  mobile patch from it. Scratch dogfood: desktop sidebar showed an agent
  created by curl without reload, Inbox badge rose on `inbox.created`;
  phone showed a new agent with no fleet poll (only health, presence and
  tui-working in 20 s); `curl -H Last-Event-ID` replayed; daemon
  restart → `hello.bootId` changed → reload. Owner pushed back on an
  invalidation-only design; the durable log is the accepted one.
- **2026-09-01** — **Two small UI fixes reported from the phone/desktop**,
  on branch `fix/apps-header-row-spacing`: (1) `AppsGrid` dropped its
  "Apps" section header — the sidebar's icon rail already marks which
  section is open, and the single "Inbox" tile beneath it repeated the
  same word (owner: "informação duplicada"). Follow-up
  (`fix/inbox-done-header`): the owner then pointed at a second
  duplicate — the Inbox's own "DONE 14" section label directly under
  the "Done 14" tab pill (same number, same word). `internal/apps/
  inbox.go`'s `doneView`/`donePane` now build that block with no Title
  at all (`BlockHead` already renders nothing when title/meta/at are
  all empty); the "All" tab's own trailing "Done" block keeps its title
  since there it names one of three real subsets (Needs you/Feed/Done),
  not a restatement of the active tab. Two inbox_test.go assertions
  that used the block's Title to find the Done pane now match on the
  done item's own ID instead. (2) `.app-row` (the shared
  list row `AppSurface.jsx` renders for every app, desktop and mobile)
  dropped its reserved 16px unread-dot gutter, blank on every read row;
  the unread mark is now an inline `●` before the title, the same
  convention the mobile Now screen already used. (3) The mobile swipe's
  `translateX` on `.app-row-actions` was silently replacing that row's
  own `translateY(-50%)`, so the revealed action buttons rendered
  vertically off-row ("fica pela metade") — fixed by combining both
  transforms. Verified on scratch: desktop Apps tab and Inbox list
  (`overlayAudit` n/a, plain layout), mobile swipe with the action
  button's rect centred within 3px of the row's.

- **2026-09-01** — **Mobile phase 3 (ADR-0044 addendum)**, on branch
  `feat/mobile-phase3`: `usePullToRefresh` + `PullScreen` (Now, Work;
  Inbox via `AppSurface.refreshKey`), touch swipe on `AppSurface` rows
  (`app-row-swiped` reveals the hover-only actions), `#/changes/<kind>/<id>`
  → `Changes.jsx` wrapping `UncommittedDetail` (now accepts owner kind
  `workspace`); entry buttons on the agent state row, the terminal header
  and the workspace card when `git.dirty > 0`. QA on scratch with the
  real dirty checkout: file list + expanded patch, synthesized touch
  swipe reveals actions, synthesized pull shows "Release to refresh" and
  refetches. Then: "a página inteira rola, o header mais botões devem
  ficar fixos e somente as listas devem rolar" — `.m-screen-head` (Work's
  segmented control + New, More's title) and `.m-inbox .ft-head`
  (Active/Done/All + filter) now use the same negative-margin/sticky-top
  trick `.m-head` already used for pushed screens, pinning them to the
  scroll container's own top edge while the rows beneath slide past
  (`fix/mobile-sticky-headers`). Agent and Terminal screens needed no
  change — their content panes already size via flex `min-height:0` and
  scroll internally, so the outer `.m-screen` never engages there.
  Verified on scratch with seeded overflow (9 workspaces, 13 inbox
  items): `getComputedStyle(...).position === "sticky"` and the header's
  `getBoundingClientRect().top` unchanged at `scrollTop 300`. Owner on
  the phone: pull did not fire on Inbox and the
  stacked split overflowed. `fix/mobile-inbox-subroute`: `AppSurface`
  `paneMode="list"|"detail"` + `onOpenItem`; mobile Inbox = list tab
  (`PullScreen` is the scroller) + pushed `#/inbox/<id>` item screen with
  Back; an action returning to the root pops the screen. Owner: an Inbox
  row needed two taps on the iPhone and swipe never showed — iOS Safari
  turns the first tap into hover when `:hover` changes content (the
  hover-only row actions). `fix/inbox-hover-touch`: the `.app-row` hover
  rules live under `@media (hover: hover)`; touch gets the swipe. Then: a
  left swipe was firing the pull — `usePullToRefresh` now decides the
  axis on the first 10px of movement and stands down for a horizontal
  drag (`fix/pull-axis`). Owner: "destaca o botão remover com a cor de
  perigo… faz um motion nesse swipe, tá muito seco" — the row now
  follows the finger (inline transform during the drag, `--swipe-w` from
  the action count) and snaps open/shut with `--motion-med`; the actions
  slide in from the right edge as 40px buttons, Delete in the danger
  tint; terminal Remove uses the same tint (`feat/swipe-motion`).

- **2026-09-01 — Automations debts closed** (`fix/automations-debts`):
  `RunObserver.OnCost` (sum of `message_end` usage) makes the cost cap
  event-driven and synchronous; `FailStaleRuns` takes a cost resolver
  (`automate.SessionCost`); the server test fake emits
  start / message_end(+usage) / end / settled, and
  `automations_e2e_test.go` runs a real `start` run plus a capped one.
- **2026-09-01 — ADR-0045 v2 `/automate` + templates** (`feat/automations-v2`):
  seven built-in templates, Suggested cards + editor select, `/automate`
  drafting from the current agent (verified on scratch: repo-specific
  prompt, weekdays 09:00, origin line), draft handoff lib. ADR renumbered
  0044 → **0045** after a same-day collision on main (dialogs moved to 0046).
- **2026-09-01** — **Web Push (ADR-0047)**, on branch `feat/mobile-push`
  — phase 2 of the approved mobile plan. `internal/push` in stdlib: VAPID
  keys (`<DataDir>/vapid.json`, 0600), RFC 8291 `aes128gcm` (Appendix A
  vector pinned in the test), sender (TTL/Urgency/Topic, 410 → drop),
  notifier with the decision table (host online → skip; result →
  *finished*; blocking → *actions*; unobserved dialog → *actions*). Hooks:
  `store.OnInboxCreated`, `rpc.Runtime.OnWaiting` (via
  `ManagedAgent.onWaiting`, `hub.Len()==0`), `presence.AnyHostOnline`.
  Store migration 018 + `store/push.go`; `server/push.go` endpoints;
  `main.go` wires one presence registry for both. Client: `sw.js` push +
  notificationclick, `main.jsx` navigate listener (SW now also on
  localhost), `lib/push.js` (+ blocked-reason test), `PushPrefs.jsx` in
  mobile More → Notifications and desktop Preferences → Notifications.
  Also: the responsive-dialogs ADR was renumbered 0045 → **0046** (another
  session's Automations ADR took 0045 in parallel); push is 0047.
  **Real-device dogfood (owner's iPhone, iOS 18.7, Home Screen install):**
  subscribe worked, `Send test` arrived. The first real trigger (blocking
  inbox question, host browser closed) got **400 from Apple** — the
  `Topic` header must be ≤32 URL-safe base64 chars and we sent
  `inbox:<slug>`; the test push only passed because its tag was `test`.
  Fixed: `topicFor` hashes the tag (`fix/push-topic`).

- **2026-09-01** — **Responsive dialogs (ADR-0046)**, on branch
  `feat/responsive-dialogs`. Owner on the phone: "New Agent abre como um
  drawer… o dialog Choose Folder já não abre assim… precisamos mapear
  todos os dialogs e aplicar a mesma regra". Inventory: 16 files with raw
  `@radix-ui/react-dialog`, 1 with `react-alert-dialog`, 1 with `vaul`
  (`CreateForm`'s own fork); 4 anchored popovers left alone. New
  `components/ResponsiveDialog.jsx` (Radix API, `useMedia("(min-width:
  720px)")` → Radix dialog or Vaul sheet; `Alert` set for confirms); one
  import line changed in 14 files; `CreateForm` lost its `useMedia`
  branch; `.dlg.dlg-sheet` in app.css reshapes the whole `.dlg-*` family;
  `lib/dialogPolicy.test.js` walks `web/src` and fails on any raw import
  outside the primitive + `Palette`/`Hotkeys`. Rule written into
  `docs/guidelines.md` and the uiux-review checklist. Owner's phone
  follow-up (`fix/mobile-sheet-touch`): the free-agent sheet sat half off
  the screen — Vaul's `repositionInputs` fought `interactive-widget=
  resizes-content`; the sheet now passes `repositionInputs={false}`. And
  a finger could not scroll a terminal: xterm scrolls its own buffer on
  touch but a tmux pane needs wheel events, so the terminal screen turns
  a vertical drag into one synthetic `wheel` per 16px on `.xterm-screen`
  (`touch-action: none` on the host), which then takes the desktop's own
  path (xterm SGR reports / lib/termWheel). Then: "os diálogos não
  deveriam ativar o teclado automaticamente" — `SheetContent` in
  ResponsiveDialog cancels Radix open-autofocus and blurs a React
  `autoFocus` in a layout effect; desktop unchanged.

- **2026-09-01 — ADR-0046 Automations** (`feat/automations`): store +
  migrations 016/017, `internal/cron`, `internal/automate`, server routes
  + runner, `Automations.jsx`, guide, benchmark study of Devin/Cursor/
  Claude Code/Codex/GitHub. Visual gate: empty, blocked (pi missing),
  editor, detail with secret, list with runs — overlayAudit ok.
- **2026-09-01** — **Mobile shell v2 (ADR-0044)**, on branch
  `feat/mobile-shell`. Owner: "o mobile não precisa refletir todo o shell
  web… observar, responder a chamados, tomar decisões, ações rápidas".
  Benchmarks (Remote Control, Codex mobile, Cursor iOS, AgentWatch,
  Tactic Remote, Nimbalyst, PagerDuty) → `docs/benchmarks/2026-09-01-
  mobile-agent-supervision.md`. Rewrote `web/src/mobile/` around Now /
  Inbox / Agents / More + a pushed agent screen; one server change
  (`agentView` live state) so the home knows who is waiting without a
  socket; agent events as a pure, tested reducer instead of touching the
  desktop's inline handler. Dogfooded on a scratch instance (`:8472`) with
  a **fake pi** on PATH (`scratchpad/dog/fakebin/pi`, speaks the JSONL RPC:
  streams a reply, or asks confirm/select/input on `ASK:*`): empty
  machine → create sheet → two agents asking → answered from Now → agent
  screen with the desktop's ask card → inbox question answered in the
  stacked detail → More → Providers. First captures found: the inbox's
  "Pick an item on the left" and Ctrl+Enter hint on a phone, Providers'
  account rows overlapping at 390px, Start/Stop icons reading as
  checkboxes, no history on the agent screen — fixed (CSS overrides
  scoped to `#m-app`, text Start/Stop, transcript tail replayed via
  `sock.seed`, verified against a real adopted session). Light + dark,
  `overlayAudit` ok on the sheet. Curated: `docs/screenshots/mobile-*.png`.
  Fixed on the way: mobile `!!` toast ReferenceError.
  Owner tested on the phone: "celular não tem view workspaces (agentes e
  terminais), não tem view de terminais… fica ruim agregar tudo em uma tab
  Agents… podemos tirar o header". Follow-up `feat/mobile-work`: header
  removed (QR stays in More); Agents tab → **Work** with the rail's three
  segments (`#/work/workspaces|agents|terminals`, last one remembered in
  `picode-mobile-work`); workspace cards list agents + terminals with
  "+ Agent / + Terminal"; `useFleet` also polls `/api/terminals`; pushed
  `#/term/<id>` screen = `TermSurface` + `KeyBar` (sends byte sequences
  through the `terms` map's socket, refocuses xterm); live terminals in
  Now → Running. QA on scratch with tmux: created a terminal from the
  phone, shell prompt rendered (34 rows), key bar clicks reach the pane,
  Remove stops the session. Owner: a workspace's terminal must not repeat
  in the Terminals segment — it now lists free terminals only
  (`freeTerminals`), like the desktop rail. Third pass (`feat/mobile-
  polish`): zoom lock (meta rewritten on mobile mount + `gesturestart`
  cancel + `touch-action: manipulation`), tab bar hidden on pushed screens
  (`#m-app.is-pushed`), key bar behind a floating `.m-fab` keyboard button
  with a close key, row actions as icons (`IconPlay`/`IconStop`/`IconTrash`,
  new `IconKeyboard` in Icons.jsx).

- **2026-09-01** — **Chrome extension Track B**, on branch `feat/ext-track-b`.
  `make desktop` builds `picode-nmh.exe`; `picode-desktop extension-install`
  registers that console host. Side panel pings `kind=extension`.
  Preferences → Server: connected / not connected + Open guide.
- **2026-09-01** — **Chrome extension Track A (ADR-0043)**, on branch
  `feat/browser-extension`. Sideload `ext/`; native host in `picode` /
  `picode-desktop`; `GET/POST /api/extension/*`. Sensor only (URL, title,
  selection, optional JPEG). Stopped agents Start and send; interactive
  refused; busy maps to follow_up via existing `SendTurn`. visual-review:
  PASS (ext-host / ext-down / ext-none / ext-chrome / ext-form / ext-terminal
  preview states; no SPA overlay). Owner dogfood of Windows Chrome still
  open (Next up).
- **2026-09-01** — **Dashboard v2 (ADR-0042)**, on branch
  `feat/dashboard-v2`. Owner asked for a benchmark sweep and a v2 plan
  after dogfooding v1; the plan was approved as written (four tiles,
  daily chart, six breakdown panels, fingerprint cache, live refresh,
  three refusals). Web research added ccusage, the Claude Code OTel
  metrics + Grafana 25052, the Claude Code Analytics API,
  pi-agent-dashboard, nicknisi/fleet and the 5of10 design rules to the
  study. Backend: `stats.go` reads the assistant message's inline
  `provider`/`model`/`usage`/`stopReason`/`toolCall`s in the same pass,
  the inline provider becomes the running truth for following lines
  (v1's `$6.15` "unknown" bucket was exactly this), `session.Fingerprint`
  + a per-range cache in `session_stats.go` (0.4 ms stat sweep vs 0.9 s
  cold all-time scan on the real 175 MB tree). Frontend: `RankedBars`
  generalises v1's `SpendByProvider`, `lib/barchart.js` + `DailyChart`,
  `TokenBar`, `TopSessions`, `StatTile` gained `deltaText`/`children`,
  `fleetStats` splits working/waiting/idle from the ids App already
  tracks for the sidebar spinner. Dogfooded on a scratch instance
  (`:8471`, copied sessions tree, two seeded workspaces): first capture
  showed a stretched "Spend by model" well beside a 25-row workspace
  list, a truncated "not a wor…" sub on every folder row, raw
  `--home-goat-…` fallback names, a `$0.00 unknown/unknown` model row
  and a squashed "where" column on top sessions — fixed with
  `align-items:start`, a six-row cap + folded tail, `folderLabel`,
  dropping the zero-cost unknown row, and a fixed-width column plus
  `lastAt` from the backend. Light + dark + 600 px + empty period all
  captured and read; `overlayAudit` ok. Curated:
  `docs/screenshots/dashboard-v2-{7d-light,breakdowns-dark,empty}.png`.
  Merged + deployed; owner's first look on real data (25 folders, 9
  models): "cards desalinhados" — the 2-column row grid with
  `align-items:start` left a hole under the short model panel. Follow-up
  `feat/dashboard-masonry`: `.dash-grid-2` is now CSS multi-column
  (`columns: 2`, `break-inside: avoid`), panels pack top-to-bottom per
  column; label column widened to `minmax(110px,190px)`. Re-captured on
  the scratch instance at All time — no holes. Second report: the
  scrollbar sat mid-window — `.dashboard-view` was both the scroller
  and the 1040px centred box. `feat/dashboard-scroll`: the outer element
  scrolls full-width, a new `.dash-inner` wrapper carries the cap and
  centring (`.dashboard-on` still keys on `.dashboard-view`). Measured
  on scratch at 1700px: scroller right = window right, inner right =
  1492.

- **2026-09-01** — **Logo opens the dashboard from anywhere**, on branch
  `feat/dashboard-logo-link`. Owner dogfooded ADR-0041's dashboard for
  real and found it unreachable in practice: it only ever showed with
  zero tabs open, so checking spend meant closing every open agent
  first. Considered a 6th sidebar icon (ADR-0026 already flagged the
  header as tight — 4 icons yield the version string below ~286px);
  owner's call instead: make the existing "PiCode" wordmark clickable.
  `App.jsx` gained `dashboardPinned` state — `showHome` is now
  `(noTabs || dashboardPinned) && hasData` — and a `.dashboard-on` class
  on `#workspace-view` (`> *:not(.main-tabs):not(.dashboard-view) {
  display:none}`) hides whatever surface was showing underneath, rather
  than enumerating every surface type the way `.term-on`/`.file-on`
  already do (those two only ever named the cases that visibly
  collided; this one has to be exhaustive by construction). Every
  open/select entry point (`openTab`, `openTermTab`, `openFileTab`,
  `openGitTab`, `openTreeTab`) unpins on call — an early version tried a
  single `useEffect` keyed on `selectedId` instead, which looked
  simpler but silently failed to unpin when the user clicked the very
  tab that was already selected before pinning (its value never
  changed, so the effect never re-fired): caught live via `agent-browser`
  clicking the same already-open tab twice, not by reasoning about the
  code. Verified against real session data (workspace with an open
  agent + 4 terminals): pin via logo, dismiss via the same tab, dismiss
  via a different terminal (the direct `openTermTab` path, which
  bypasses `openTab` entirely) — all three re-tested after the fix.
  `overlayAudit` clean, both themes screenshotted.

- **2026-09-01** — **Session observability dashboard (ADR-0041)**, on
  branch `feat/session-dashboard`. Owner rejected a first attempt
  (`feat/home-dashboard`, unmerged): a `HomeView` that listed workspaces/
  agents/terminals in the no-tabs-open main pane — "ux fraca... sem
  sentido visto que a sidebar já está disponível" — and asked for
  analytics/observability instead, explicit that it should not just be
  another quick-access list.
  - Research surfaced two existing refusals that needed reconciling
    before building anything: `docs/design/session-surface-roadmap.md`'s
    "Cost as a new page | it belongs on the session chip" (about *one
    session's* cost; this is a fleet aggregate, different question) and
    the "X is not the home" pattern used twice against other surfaces
    annexing the live agent-conversation identity (this only ever renders
    when nothing is open, so nothing is displaced). Owner confirmed this
    reading explicitly before implementation.
  - New `GET /api/sessions/stats?range=today|7d|30d|all`
    (`internal/session/stats.go`, `internal/server/session_stats.go`):
    scans session JSONL at **message** granularity (`entryTS`/`costFrom`,
    already unexported in the same package) rather than bucketing by
    `Summary.UpdatedAt` — file-mtime bucketing would dump a multi-day
    session's whole cost onto one day. Response carries only aggregates,
    never `Preview`/raw rows (sabotage-tested). `range=all` has no cheap
    pre-filter and no cache — accepted debt, same posture `ListAll()`
    already has.
  - Frontend: `DashboardView.jsx` + `StatTile.jsx` (value/delta/sparkline)
    + `SpendByProvider.jsx` (ranked one-hue bar list) + `DateRangePicker.jsx`
    (native radio segmented control, mirrors `.termset-seg`). New
    `lib/sparkline.js` (pure SVG path geometry, mirrors the
    `lib/gitgraph.js`/`GitGraph.jsx` split — no charting dependency
    added) and `lib/dashboardStats.js`, both unit-tested. Range choice
    persists per-viewer in `localStorage` (`lib/openTabs.js`), not the
    hash router — this app's `#` routes name what's open, never a view
    filter.
  - Visual-review caught one real defect before ship: the stat-tile
    loading skeleton's `<span class="skel-line">` had no block-level
    context (`.stat-tile-skel` lacked `display: flex`, unlike the
    sibling `.spend-skel` that already worked), so the skeleton bar was
    invisible — fixed, re-verified against an artificially slowed
    fixture (`agent-browser wait --fn` polling for `.stat-tile-skel`,
    since local fetches are normally too fast to catch by a fixed delay).
  - Cross-check: `range=all`'s `current.cost` matched `/api/sessions/all`'s
    summed `Summary.Cost` exactly (7.33 == 7.33) on the same fixture data
    — two independent code paths agreeing.
  - Ported the `WorkspaceRows.jsx` extraction (`AgentRow`/`TermRow` pulled
    out of `Sidebar.jsx`) from the abandoned `feat/home-dashboard` branch
    via `git apply` of just that file pair — still valid on its own,
    independent of the rejected `HomeView` it was built for. `HomeView.jsx`
    and its ADR were not ported.
  - Not merged; `feat/home-dashboard` (superseded, unmerged) left as-is
    for the owner to delete at their convenience.
  - **Follow-up fix, same day, caught during the owner's live preview**:
    `scanFile`'s message-timestamp read used `time.Unix(ts, 0)`, treating
    pi's real `message.timestamp` (epoch **milliseconds**, JS `Date.now()`
    convention) as seconds — the parsed time landed around the year
    58620, tripped the "future relative to window" guard, and silently
    dropped almost every real message. The unit tests never caught it
    because `msgLine()`'s synthetic fixtures also used `.Unix()`, so they
    validated the code's assumption against itself, not against pi's real
    wire format — the earlier "confirmed seconds from a real fixture"
    research note was reading a `compaction` event's unrelated top-level
    `timestamp`, not a `message.timestamp`. Found by copying real
    `~/.pi/agent/sessions` (83 files, 174M) into a scratch preview instead
    of trusting synthetic-only QA: `range=7d` and `range=all` came back
    identical (4 messages total) against 24,811 real ones. Fixed to
    `time.UnixMilli`; re-verified the cross-check from above against real
    data (854.9008726400004 vs `/api/sessions/all`'s independent
    854.9008726400006 — last-digit float ordering only). Commit
    `d532c31c` on the same branch.

- **2026-09-01** — **Apps host: tab height unified, split pane
  resizable** (`feat/apps-host-chrome`). Owner used the just-shipped
  Inbox tabs and flagged two chrome inconsistencies: "tabbar" height
  differs by location, and a "sidebar" (the Inbox's list pane) isn't
  resizable — clarified to mean the apps host's `.app-split` list pane,
  not `#sidebar` (confirmed live via dispatched PointerEvents that
  `#sidebar`'s own resize already works correctly; CDP's synthetic
  mouse events just don't trigger it reliably, a test-harness quirk,
  not a product bug).
  Audited every `role="tab"`/`role="tablist"` in `web/src/components/`:
  4 patterns, 3 heights — `.mtab` (window tab strip) and `.brand-tab`
  (sidebar's own Workspaces/Agents/Terminals/Apps/Pins switcher) both
  already resolve to 40px via `--chrome-h` (structural — that row is
  shared with the sidebar header, left alone); `.pref-tab` (Preferences
  nav) was `36px` hardcoded; `.app-tab` (Inbox Active/Done/All) was
  `30px` hardcoded, the value introduced this session without
  reconciling it against anything else. Both now reference `--ctl-h`
  (36px) — already the file's dominant control-height token, and the
  smaller visual delta for the Inbox. `.termset-seg-face` (Terminal
  Settings scope picker) stays on `--ctl-h` unchanged: confirmed it's a
  mutually-exclusive value picker (radio group), not view navigation —
  already correct, not part of the inconsistency.
  Surveyed every existing drag-to-resize sizer before adding a 5th:
  `Sidebar.jsx` (`picode-sidebar-w`), `FileTreeSurface.jsx`
  (`picode-ft-w`, `.ft-split`/`.ft-sizer`), `FilePane.jsx`
  (`picode-file-w`), `GitGraphSurface.jsx` (vertical,
  `picode.gg.detail-h`) — all four persist to `localStorage`, all four
  use a bare 6px hover-highlight handle with no permanent affordance.
  `.app-split` (`AppSurface.jsx`) converted from CSS Grid to flexbox
  (nothing else in `app.css` depended on it being a grid — checked) so
  the new `.app-split-sizer` could use the same
  flex-sibling-with-negative-margin technique as `.ft-sizer` instead of
  fighting the stacked-breakpoint media query with `!important` (a
  pattern that doesn't otherwise exist in this file). New `AppSurface.jsx`
  state (`listW`/`resizing`/`stacked`, the last one now tracked live via
  `matchMedia("(min-width: 881px)").addEventListener("change", …)`
  rather than the one-shot check the mount effect already had) and an
  `onSizerDown` handler copied in the same shape as the other four —
  deliberately not extracted into a shared hook, since the codebase has
  independently reimplemented this same ~15-line pattern four times
  already rather than factoring it out. Persistence key
  (`picode-app-split-w`) is global, not per-app-id, matching
  `FileTreeSurface`'s own global `TREE_KEY` precedent — one open
  question inherited unchanged from that precedent: two split-app tabs
  open at once don't live-sync a drag between them, only the next
  mount picks up the saved width.
  QA on isolated :8616: tab height measured 36px (was 30) in a live
  DOM read; list pane measured 380→530px across a real drag and held
  530px after a full page reload; at 760px width the sizer is absent
  from the DOM (not just hidden) and `.app-split`'s computed
  `flex-direction` is `column`; dark theme screenshot clean. No new
  tests: the JS runner (`node --test src/lib/*.test.js`) only covers
  `web/src/lib/`, there is no `.test.jsx` or DOM harness in the repo,
  and this change adds no new pure-logic surface to test (the clamp
  math is inline, same as the four precedents it copies).

- **2026-09-01** — **Per-agent `--session-dir` (ADR-0040)**, on branch
  `worktree-pi-tui-session-dir`. Owner reported (screenshots) that after
  ADR-0039 shipped, an Agent's **chat** picker correctly showed only its
  own session — but the *same* agent's raw interactive pi TUI still
  showed every session sharing that cwd via pi's own native "Resume
  Session" picker. Root cause: that picker is pi's own code, reading
  `~/.pi/agent/sessions/<cwd>/` straight off disk, with zero awareness of
  PiCode's HTTP API or `agent_sessions` — a DB-side filter structurally
  cannot reach it.
  - Reopened ADR-0039's own rejected alternative ("give each agent a
    private `--session-dir`") after three facts confirmed live against
    `pi 0.84.4`: it lands a fresh session directly in the given dir (no
    cwd sub-bucketing), `--continue` finds/appends there and never
    touches the default bucket (proves it governs lookup, not just
    storage), and an explicit `--session <path>` outside the dir still
    wins for a resume (so **no physical migration** of existing sessions
    was needed — only future fresh starts move).
  - New `session.AgentDir(agentID)` (`~/.pi/agent/sessions/<agentID>/`),
    nested *inside* `Root()` specifically so `ListAll`/`ListRoot`/
    `UnderRoot` needed zero changes — confirmed by reading `ListRoot`'s
    loop (no naming-shape filter on subdirectories).
  - `Agent.CLIFlags()` unconditionally appends `--session-dir` — reaches
    all three spawn chokepoints (tmux, managed RPC, and the headless MCP
    OAuth spawn) through one function. Placed last in the flag list to
    minimize test churn (4 assertions in `store_test.go` needed updated
    expected lengths; the rest only checked a lower bound or leading
    elements and were untouched).
  - `safeSessionPath` generalized from a single `(cwd, path)` root to
    `(path, dirs ...string)`; `handleManageSessions` and
    `sweepOrphanSessions` gained a `workspaceSessionDirs` union so the
    workspace manage view and the age sweep also cover each agent's
    private dir. The delete/purge-preview flow (`cleanup.go`) got the
    same treatment (`DirStatsAt`, `RemoveAgentDir`) — required, not
    optional, since without it "delete sessions too" would silently leave
    an agent's `AgentDir` behind after this change, a regression this
    design itself introduces.
  - **Bundled by owner's choice**: a separate, adjacent gap found while
    researching `agent_sessions` — the age-based orphan sweep only
    checked an agent's *current* `session_path`, never its full
    `agent_sessions` history, so an older session already resumable in
    that agent's own chat picker (ADR-0039) could be silently deleted
    once it aged out. Closed with `Store.AllAgentSessionPaths()` and one
    added skip-check in the sweep.
  - Verified live end-to-end, not just proxied through `--print`/
    `--continue`: scratch instance, two agents in one workspace sharing a
    cwd, both started in **interactive** mode, each sent a real message
    over `tmux send-keys`, then `/resume` triggered directly inside each
    agent's own tmux pane (`tmux capture-pane`) — each agent's native
    "Resume Session (Current Folder)" overlay showed **only its own**
    session, the other agent's completely absent. This was the one fact
    the design couldn't verify itself and had to be checked by hand.
    Also verified: `pi --mode rpc --no-session --session-dir <X>` starts
    clean and leaves `<X>` empty (the MCP OAuth spawn is unaffected by
    inheriting the flag).
  - `make fmt-check`/`vet`/`test`/`build` all green.
  - Not yet merged to main or deployed to `:8445` — branch
    `worktree-pi-tui-session-dir` ready for review.

- **2026-09-01** — **Inbox: Active/Done/All tabs + manual cleanup**
  (`worktree-feat-inbox-tabs`, ADR-0037 follow-up). Items were never
  deleted (`UPDATE` only, confirmed by reading every write path) but
  `done` was invisible — the default view only ever queried
  `unread`/`read`, so a resolved question just disappeared, recoverable
  only via a raw `?state=done` API call. Owner's call, in order:
  visibility first, manual cleanup second, retention policy explicitly
  deferred. A segmented `Active | Done | All` control (`View.Tabs`, new
  optional field — ADR-0036 amendment, not a new block type) sits above
  the list; Done rows carry a truncated summary of the reply; a done
  item's detail now mirrors the Done list beside it instead of Active
  (it used to "jump" to the wrong list). Delete is both per-row (hover
  reveals a trash icon, confirm) and bulk (**Clear all done**, confirm
  with the count); store gained `DeleteInboxItem`/
  `DeleteDoneInboxItems`/`CountInboxItems`/`CountAllInboxItems`,
  `ListInboxItems` untouched — Active and Done stay two existing calls
  combined at the app layer rather than a new "all states" filter mode.
  Two REST routes for API parity: `DELETE /api/inbox/{id}`, and `DELETE
  /api/inbox?state=done` (bare `DELETE /api/inbox` refused, 400 — no
  accidental full wipe).
  Two real bugs, both found only by driving a real browser (chromedp
  against an isolated scratch server), not by reading the diff: (1)
  `AppSurface`'s selected-row lookup assumed every list-pane block had
  `.items`, and crashed (`Cannot read properties of undefined`) on the
  new actions-only "Clear all done" block — fixed generally (`b.items
  || []`), not just for this one block. (2) the post-action
  `returnPath` rule collapsed to Active root whenever the acted-on path
  merely started with `item/`, which also fired for deleting a
  *sibling* row while a *different* item's detail was open; it now
  compares the acted-on id against the currently-open item's own id,
  and only then collapses — a regression test
  (`TestInboxDeleteSiblingRowWhileDetailOpen`) pins the distinction.

- **2026-09-01** — **Per-agent session ownership (ADR-0039)**, on branch
  `worktree-session-ownership`. Owner reported (screenshot) an Agent tab's
  Search sessions picker showing a session that actually belonged to a
  Terminal in the same folder. Root cause traced to
  `handleListSessions` (`internal/server/sessions.go`): `agent=<id>` only
  resolved a cwd, then listed every `.jsonl` pi had written there,
  unfiltered — confirmed against live code, not just the report.
  - New `agent_sessions` table (migration `015_agent_session_history.sql`;
    renumbered from `014` after merging main, which had independently
    claimed `014_inbox.sql` for ADR-0037)
    historizes every session id/path an agent is pointed at.
    `store.UpdateAgent` historizes on every `SessionPath` write (resume/
    fork/clone/adopt/import, one hook, not nine call sites).
  - Fresh spawns pre-mint a pi `--session-id` (confirmed live against the
    installed `pi 0.84.4`: creates-if-missing, reuses on repeat, not
    gated behind `--mode`) so a brand-new session is attributable from
    the moment it exists — `Agent.CLIFlagsForSpawn`, wired at the two
    real spawn chokepoints: `rpc.Runtime.Start` (managed) and a new
    `Deps.spawnFlags` helper (5 interactive/tmux call sites).
    `handleListSessions` filters `session.List(cwd)` against the agent's
    own historized set; current session is always shown as a safety net.
  - Terminals needed zero changes — confirmed they have no `SessionPath`
    field and never call this endpoint. Machine-wide `/sessions/manage`
    and `/sessions/all` deliberately stay unfiltered.
  - Caught in the same pass: the new `--session-id` flag broke
    `TestOpenCloseLifecycle`, which used bare `cat` as pi's stand-in
    (relies on zero args to block on stdin; real cat errors on
    unrecognized `--` flags). Fixed with a tiny wrapper script that
    ignores argv, not a change to production spawn logic.
  - Tests: new unit coverage in `internal/store`
    (`CLIFlagsForSpawn`, the historization hook, and a migration test
    that genuinely re-applies `015_...sql` via `Store.migrate()` against
    a pre-existing `session_path`, not a hand-copied approximation) and
    `internal/server` (two agents sharing a cwd each see only their own
    session; an unowned file is invisible to both but still shows in
    `/sessions/manage`). `make fmt-check`/`vet`/`test` green.
  - `make ci` (fmt-check, vet, test, test-js, build) green — no frontend
    files touched; `SessionBar.jsx` needed no changes since the scoping
    is entirely server-side.
  - **Verified live**, not just by unit test: scratch `HOME`/`PICODE_DATA`
    instance on :8491 (real `pi 0.84.4`, real auth), one workspace, two
    agents both defaulted to the same cwd — the exact shared-cwd
    precondition from the bug. Managed-started both, sent each a real
    prompt ("pong" / "ping"). Each agent's picker showed **only its own**
    session; `/sessions/manage` showed both, unfiltered, as intended.
  - **Second bug found in that same live pass, fixed before calling this
    done**: `agents.session_path` (not just the new `agent_sessions`
    table) never got backfilled after a *fresh* spawn — only an explicit
    resume ever wrote it. Harmless for the picker itself (it falls back
    to the newest owned session), but `/sessions/manage`'s `inUseBy` —
    the guard that stops the orphan-cleanup sweep from deleting an
    actively-used session — reads `agents.session_path` directly, so a
    freshly-started, never-resumed agent's session looked orphaned.
    `ResolveAgentSessionID` already existed for exactly this (written,
    never wired); `handleListSessions` now calls it — and backfills
    `agents.session_path` — the first time it notices an owned-by-id
    session whose path wasn't known yet. Re-verified live after the fix:
    `inUseBy` correctly attributes each session to its own agent.
    Regression test added (`TestListSessionsResolvesFreshSessionPath`).
  - Not yet merged to main or deployed to `:8445` — branch
    `worktree-session-ownership` is ready for review.

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

- **2026-09-01** — **portrait video preview fits the pane** (`feat/video-preview-fit`):
  a 1080×1920 `.mp4` used to set `width: min(100%, 960px)` with no max-height,
  so Preview grew a scrollbar and hid the controls under the viewport.
  `.file-preview-fit` is a one-cell grid that can shrink below intrinsic size;
  the `<video>` / `<img>` then `object-fit: contain` inside it. Markdown, SVG
  and mermaid still scroll. Scratch `:8460` QA: video 290×516 inside a 548px
  pane, `paneScroll === paneClient`, `overlayAudit ok`. Captures: happy
  portrait, Raw ("Can't show this file."), gone, decode error.
  visual-review: PASS (file-video-portrait-fit.png + overlayAudit ok; card 5/5).

- **2026-09-01** — **tab surfaces keep their state** (`fix/tab-state`):
  the file tree, git graph and app surfaces were rendered only while
  selected, so every tab switch unmounted them and threw away the
  expanded folders, the scrolled-in history with its open commit,
  search and branch filter, and the app's open item. They now follow
  the shape `TermSurface` already used — one mounted instance per open
  tab, `hidden` while another is selected — in `App.jsx` via
  `tabs.filter(isTreeTab|isGitTab|isAppTab).map(...)`, with the
  handlers closing over the tab's own id (a hidden tree resolving its
  root used to rename whatever tab was selected). `hidden` is set on
  every `<section>` a surface can return, skeleton and error included:
  one branch left out paints its skeleton beside the visible tab.
  New `web/src/lib/keepScroll.js` — `useKeptScroll(hidden, selectors)`
  restores scroll offsets in a layout effect (display:none drops the
  scroll box, so reading it at hide time reads 0) using a *capturing*
  listener, which is how `.gg-rows` is covered without threading a ref
  through `GitGraph`. Refresh discipline: hidden surfaces skip the
  window-focus refetch (otherwise every open tab re-reads folders
  nobody is looking at), and a reveal refetches only past
  `REVEAL_STALE_MS` (10s) — the git graph is deliberately excluded,
  its refresh has been manual since the ADR-0038 poll was dropped.
  QA on isolated :8613: 501-commit graph + open commit + `fix` search
  + scroll 3000 survived a switch; inbox item stayed open; tree kept 3
  folders and scroll 750; cycling all three tabs showed 3 mounted, 1
  visible, 1 overflowing scroller each time; branches overlay
  `overlayAudit ok`. Accepted cost: a restored tab loads eagerly at
  boot (one browse+gitstatus per tree tab), far less than the
  websocket+xterm terminals already pay. Not covered: a browser reload
  still starts collapsed — the state lives in the mounted component,
  and a reload is a new page.

- **2026-09-01** — **Inbox UI refined** (branch `worktree-feat-inbox-ux`):
  split view (list left / detail right; stacked and scroll-into-view
  under 880px), rows with unread dot + relative time (absolute on hover)
  + kind lozenge + hairline-separated meta, per-row hairlines with the
  selected row flush and stripe-marked, sections as uppercase eyebrows
  with counts and a divider between them, icon-only Done/Snooze revealed
  on hover/focus (the timestamp steps aside), reply form with
  Ctrl+Enter, one filled button per row (the app declares
  `Action.primary`), approvals gained **Decline** (forwards a real "no"
  instead of silence), centred blankslates for empty inbox and empty
  detail pane. Apps host grew optional fields only — layout/pane/empty
  hints, block+row meta/at, row tone/unread, action icon/primary — the
  four block types are unchanged (ADR-0036 amendment 2026-09-01).
  Global side-fixes: `.md pre` never had block styling (fences rendered
  as inline chips) and Tailwind's preflight had stripped markdown list
  markers — both fixed for every markdown surface.
  New: `scripts/inbox-smoke.sh <base-url> [agent-id]` fills one item of
  every shape (incl. long title, dead-agent question, snoozed, answered)
  for visual QA; `web/src/lib/relTime.js` is the shared relative-time
  helper (SessionsView/Packages still carry their own divergent copies —
  worth folding into it next time either is touched).
  Reviewed against Sentry/Linear/Geist/Primer/Grafana/Atlassian specs;
  measured computed styles to check claims (meta/timestamps were already
  on the right tokens). **Known, deliberately left**: `.btn-primary` in
  dark is `#7c8cf8` with white text (~2.5:1) — a product-wide contrast
  bug, not inbox-specific; fix deserves its own pass. No keyboard list
  navigation (j/k) yet.

- **2026-09-01** — **The repository root is now pinned to main by git
  itself**, not by convention (AGENTS.md §5). `.githooks/reference-transaction`
  aborts a `git switch`/`git checkout` that would move the root checkout off
  main; `.githooks/pre-commit` refuses feature commits made there. Git has no
  pre-checkout hook, but a branch switch is a symref update of HEAD and a
  reference transaction can be aborted (git ≥ 2.28) — so enforcement is
  tool-agnostic: it holds for every agent runtime, editor and script, which a
  Claude-Code-only hook would not.
  The trap worth remembering: `git worktree add -b <branch>` writes the NEW
  worktree's HEAD through the *same* transaction, from the same directory,
  with the same git dir, the same common dir and the same
  `ref:refs/heads/...` payload — the naive hook blocked the very flow it is
  meant to force. The only discriminator is the invoking command, read from
  `/proc/$PPID/cmdline` (with a `ps` fallback); so the hook is Linux/WSL-first
  by design.
  `make hooks` wires `core.hooksPath` (implied by `make dev` and `make ci`);
  a clone that never ran make has no guard, and `git -c core.hooksPath= …`
  bypasses everything — this is protection against carelessness, not malice.
  Escape hatch: `PICODE_ALLOW_SWITCH=1`. Note `gh pr checkout <N>` shells out
  to `git checkout` and is therefore refused in the root: review PRs from a
  worktree, or use the override.
  Verified on a clone of this repo, not in theory: switch/-c/checkout -b
  blocked, worktree add + switch/commit inside a worktree fine, commit on
  main in the root fine, `checkout -- <file>` fine, return to main always
  allowed, override works, and pre-commit blocks a feature commit when the
  root is off main. GitHub Actions calls individual make targets, so it never
  sets hooksPath.

- **2026-09-01** — **`make ci` now proves the git guards instead of assuming
  them.** `scripts/hooks-selftest.sh` runs the whole policy matrix against a
  throwaway repo (never this clone): switch/`-c`/`checkout -b` refused in the
  root, worktree add + switch + feature commit inside a worktree allowed,
  commit on main in the root allowed, `checkout -- <path>` allowed, override
  works, return to main always allowed, and pre-commit refusing a feature
  commit when the root is off main. Wired as `make hooks-check` (a `make ci`
  prerequisite) and as a Linux-only step in GitHub CI, which calls individual
  make targets and would otherwise never see it.
  It is sabotage-verified both ways: removing the parent-command
  discriminator makes the worktree checks fail (the guard would block the
  flow it demands), and making the hook permissive makes the refusal checks
  fail. Either regression is silent without this test.
  `make hooks` moved into `scripts/hooks-enable.sh` and now **fails** when
  `core.hooksPath` was redirected elsewhere. Trap found while building it:
  `core.hooksPath` is a path-type config, so git reads it back **absolute**
  from a linked worktree even when written as `.githooks` — comparing the
  literal string reported every correctly-configured worktree as broken,
  which would have failed `make ci` for every agent. The script compares
  resolved paths against the main worktree's `.githooks`.

- **2026-09-01** — **Two Inbox fixes, both found by testing the feature
  live** (`pi install -l ./packages/pi-inbox`, a real `ask_human` turn).

  **Typography**: the split-view refactor had set `.app-surface { font-
  family: var(--sans) }`, flipping the whole app surface (titles, meta,
  buttons, section labels) to sans. That breaks the house rule — `body`
  is mono by default and only `.conversation` (chat prose) opts into
  sans; git graph, file tree, sidebar, settings all stay mono. Removed
  both overrides (`.app-surface` and `.app-surface .ft-title`) and the
  reply textarea's explicit `--sans`; the surface now inherits mono like
  every other list/detail app in the product.

  **Park-and-wake gap**: replying to a question from an agent parked in
  a TUI/tmux session queued a `follow_up` task that nothing ever drains
  — `deliverLoop` only exists for the RPC runtime (ADR-0037 promised
  "the agent picks it up on the next start" without that caveat).
  Fixed by refusing early: `store.RespondAndForward` gained a
  `deliverable AgentDeliverable` parameter; when it says no the item is
  annotated and stays open instead of enqueueing a message that would
  sit forever. `deps.agentInteractive` (internal/server/agents.go, next
  to the existing `runMode`) computes it from `Runtime.Get` + tmux.
  **The regression that shipped first**: the raw `/api/inbox/{id}/respond`
  route negated correctly; `internal/server/apps.go`'s `appsHost()` —
  which the actual UI goes through — passed the same boolean straight
  through under a *differently named* field
  (`Host.AgentInteractive`, true=interactive) into a parameter
  expecting the opposite polarity (`deliverable`, true=ok). Silently
  inverted: the UI accepted the reply and queued it forever. Caught
  live in the browser, not by the unit test I'd written for the app
  layer — that test set the Host field directly, bypassing `appsHost()`
  entirely, so it could not see the wiring bug. Fixed by unifying the
  name and polarity (`Host.AgentDeliverable`, matching
  `store.AgentDeliverable` exactly) and adding
  `TestInboxAppActionRespondInteractiveAgent`, an HTTP round trip
  through the real apps route against a real tmux session — sabotage-
  verified: reintroducing the missing negation fails it.
  Also hardened in passing: `rpc.Runtime.Get` panicked on a nil
  receiver (`r.mu.Lock()`); every other test server in this codebase
  either sets `Runtime` or gets lucky and never calls `runMode`/
  `agentInteractive`. Now nil-safe, matching the rest of the codebase's
  nil-safe accessors (`apps.Registry`, `Deps.Apps`).
  `pi-inbox` stays installed in this workspace's `.pi/settings.json`
  (`../packages/pi-inbox`, relative — portable across clones) as the
  live proof the package works; nothing else depends on it being there.

- **2026-09-01** — **pi-inbox + pi-roles coexistence verified live.**
  `.pi/settings.json` now carries both as project packages
  (`../packages/pi-inbox`, `../packages/pi-roles`); `npm:pi-agent-
  browser-native` moved out of the project list but stays active from
  the user's machine-wide settings, so nothing lost there. Two real
  `pi -p --no-session` turns with both extensions loaded: (1) no
  `PICODE_AGENT_ID` — `ask_human` filed correctly under the
  `sourceKind: system, sourceId: "pi (unmanaged)"` fallback; (2) a real
  free agent, identity attributed correctly, reply POSTed → 200, item
  `done`, `follow_up` task `queued` → agent started managed →
  `delivered` in ~4s. No load conflict between the two extensions in
  either case.
