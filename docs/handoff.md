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
- Providers: API key + OAuth. Claude/Codex loopback `53692`/`1455`. Copilot/Kimi/xAI **device-code**. Radius stays TUI (gateway URL).
- **ADR-0013** vault `~/.picode/accounts.json`; `auth.json` is the one slot pi reads. Add account / Use / Sign out; OAuth re-login updates the same account.
- Composer `/` opens **PiCode UI** (never the dock). Skills/templates = picker insert; pi RPC expands. `/share` = secret gist (`gh`).
- `/llama` dialog on the **current view** (URL, Save, Retry, load/unload, HF download). Link **Set up llama.cpp** → `www/guide/llama.md`. **No GUI installer** (reverted).
- Voice **V1 shipped** (dictation + Grok composer + browser TTS). Owner dogfood done (Chrome Windows mic).
- Public docs: VitePress `www/` → GitHub Pages. GUI chrome carries **state**, not docs. ADRs 0001–0013.
- `#/mcps` missing adapter: one line + Open packages (`www/guide/mcp.md`). No npm/architecture in the view.
- MCP / Packages / Settings name the selected agent (icon + name) as the first line in the card. Scope pills say **This agent**. Sidebar agent click from a pane goes to `#/`.
- MCP **Use from…** mirrors other apps. User picks; Off hides a server. Empty host files are not offered.
- MCP Add **More**: env on command servers; headers + Sign in / Token on URL servers.
- MCP list live state: Idle / Live / Failed / Sign in when the GUI agent is running. File Off has no word.
- MCP **Sign in** is automatic like providers: short `pi -e` runs headless `authenticate()` (no paste). Pi does not open the browser. GUI `window.open`s the URL once. Overlay ends when the callback HTML is served (keyring write continues). OAuth rows with tokens show **Sign out** (forgets the keyring login on this machine). No extra “Signed in” label.
- Composer `@` / images / `!cmd` shipped. MCP list/add/toggle/remove/Use from/live/auth shipped.
- Track **C1 waiting**: confirm/select/input/editor is a chat card; `POST /api/agents/{id}/ui`; sidebar says Waiting. Notify is a toast. Mobile has no waiting card.
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

## In flight

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
built and visually verified. The ADR is delivered.**

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
- Mobile parity (shell exists; not feature-complete).
- `/tree` in-place leaf jump needs pi RPC `navigate_tree` ([pi#8645](https://github.com/earendil-works/pi/issues/8645)); today click forks.
- Cold start parses the whole session JSONL (10 s on a 129 MB session) — filed upstream: [pi#8843](https://github.com/earendil-works/pi/issues/8843) (lazy resume / load checkpoint).
- Worktrees / parallel isolated agents (Orca + Herdr) — after Track E.

## Known debts / open questions


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
