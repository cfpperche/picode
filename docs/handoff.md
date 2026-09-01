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
- Mobile parity (shell exists; not feature-complete).
- `/tree` in-place leaf jump needs pi RPC `navigate_tree` ([pi#8645](https://github.com/earendil-works/pi/issues/8645)); today click forks.
- Cold start parses the whole session JSONL (10 s on a 129 MB session) — filed upstream: [pi#8843](https://github.com/earendil-works/pi/issues/8843) (lazy resume / load checkpoint).
- Worktrees / parallel isolated agents (Orca + Herdr) — after Track E.

## Known debts / open questions

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
