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
- `#/mcps` missing adapter: one line + Open packages (`www/guide/mcp.md`). No npm/architecture in the view.
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
