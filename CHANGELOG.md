# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

**Agent contract:** every commit with a user-visible change MUST add an entry
to the `[Unreleased]` section. The repository's official language is English
(see `AGENTS.md`); changelog entries included.

## [Unreleased]

### Changed

- **Extension questions stay in the thread as one form.** Sequential
  selects grow a compact stepper (labeled pills + filterable dropdown).
  Click a prior pill to go back and clear later steps. Done is one line
  (`vision — xai/grok-4.5 · medium`), kept across reload and tab switches.
  Cancel is a quiet line, not an empty card.

### Fixed

- **Going back in `/roles edit` no longer drops prior pills or stuck on Thinking.**
  Clicking a previous step reopens that field and ignores the wrong follow-up
  selects until the matching one arrives.
- **Z.AI Usage** (ADR-0031): GLM Coding Plan Pro reports `CREDIT_LIMIT`
  (5h + weekly credits) instead of `TOKENS_LIMIT`. The dialog showed the
  plan name and "No usage windows on this plan." It now draws those bars.

### Added

- **pi-roles v2** (ADR-0033): a PiCode agent sets `PI_ROLES_AGENT` and can
  overlay `<cwd>/.pi/roles/<id>.json` on the workspace `.pi/roles.json`.
  Sibling agents keep different defaults; a `pi` in a terminal still uses
  only the workspace file. `/roles edit|add` in that agent writes the overlay.
- **Provider Usage V3** (ADR-0031): Usage sits on each vault account, not
  only the active one — no **Use** required. OpenRouter, MiniMax, MiniMax
  CN, and Kimi API keys get meters. Grok reset credits also try the Grok
  CLI session (`~/.grok/auth.json`) and an explicit `GROK_COOKIE`. Qwen
  Token Plan has no API-key quota URL, so Usage stays hidden.
- **File tree V2** (ADR-0032). Clicking a row in the tree's Changes section
  now opens that file's working-tree diff in a pane beside the tree (+n −n,
  binary and truncation handled; "Open file" jumps to the editor). The
  header gains **Reveal** — opens the folder in your OS file manager
  (Linux, Windows+WSL, macOS). The surface refreshes when you come back to
  the app (focus/visibility), and the sidebar's branch pill shows a count
  of uncommitted changes.

### Changed

- **`/roles edit` and `/roles add` pick provider, then model, then thinking**
  instead of dumping every `provider/id` in one list.
- The second line of agent and terminal rows is now two pills: filesystem
  and repository, separated — `[folder icon + dir]` opens the file tree,
  `[git icon + branch]` opens the git graph. A non-repo row shows the dir
  pill alone; a narrow sidebar ellipsizes the dir and keeps the branch.

### Added

- **Provider Usage V2** (ADR-0031): Usage also covers ZAI and OpenCode Go
  API-key plans. Codex and Grok show banked **reset** credits when the
  vendor answers; Redeem asks first, then spends one. Grok omits the reset
  row if that call needs a grok.com cookie and fails — weekly usage still
  shows.
- **Provider Usage dialog** (ADR-0031): on `#/providers`, a **Usage** button
  appears on signed-in Claude, Codex, Copilot, Kimi and xAI accounts. It
  opens live 5-hour / 7-day / weekly / extra windows for the active login.
  API keys and providers without a plan meter stay without the button. The
  composer statusbar is still session tokens and cost only.
- **Composer `/` lists commands from the running agent** (ADR-0029): extension
  commands such as `/roles` appear only while that agent is in managed mode
  and the package is loaded. Picking one sends the command; it does not open
  a PiCode page.
- **`packages/pi-roles`**: opt-in pi extension that routes models by role
  (vision on images, plan on `/plan`, named presets via `/role`). Workspace
  file `.pi/roles.json`; MIT in that directory only (ADR-0028). Not installed
  by default. `/roles edit|add|remove` writes that file (no PiCode page).
- **A file tree per workspace, terminal and agent** (ADR-0030). A folder
  icon on any sidebar row (and a Files action on the workspace card and in
  the palette) opens `#/tree/…`: a read-only tree of that owner's folder,
  with a **Changes** section on top listing what the working tree touched —
  modified, untracked, added, deleted, renamed, conflicted — and dots on
  changed files and their folders. Clicking a file opens the normal file
  tab. Two owners of one folder share one tab; Refresh is manual, like the
  git graph. Workspaces became file-reading owners for this
  (`/api/workspaces/{id}/browse|text|blob|file|gitstatus`), so an empty
  workspace can browse and open its files with no agent in it.

### Fixed

- **Stop ends a stuck Working after `/roles` (and other extension dialogs).**
  The composer set Working as soon as the command was sent; a picker that
  never drew a card left the turn open, and Stop did not cancel that wait.
- **Packages still lists what is on this machine when a terminal is selected.**
  Opening `#/packages` from the Terminals tab sent the terminal id as an
  agent id, the API answered 404, and the Installed row vanished — the
  packages themselves were never removed.

### Changed

- **Creating a workspace no longer creates an agent** (ADR-0027). The
  New-workspace form asks for a name and a folder — Provider, Model,
  Thinking and the session shortcut moved out; a workspace can stay without
  agents and terminals for as long as you like. Breaking for API readers:
  `POST /api/workspaces` answers with `agents: []` and no `agent` key, the
  `agent` field is omitted whenever a workspace is empty, re-adding a
  registered folder no longer resurrects a deleted agent, and
  workspace-scoped calls that need an agent (open, close, sessions, status)
  answer 409 instead of a 500 or a misleading 404.

### Fixed

- The user menu no longer clips its Theme and Layout segments: the popover
  widened to fit "Desktop · Auto · Mobile" with their icons, segments share
  equal widths with centered content, and the Install app button spans the
  full row with a download icon.

- The workspace card's header hugs the left edge: the chevron sits on the
  section title's column and the workspace name lands on the same column
  as its cards' names — the container no longer sat further right than
  its content.

- The workspace favicon lookup now checks a frontend living in a subfolder
  (`web/public`, `www/public`, `ui/public`, `frontend/public`,
  `client/public`) — PiCode's own repo keeps its favicon in `web/public/`
  and the card showed a plain folder.

### Added

- **Workspaces hold terminals.** The terminal button on a workspace card
  creates a terminal owned by that workspace, born in the workspace folder.
  Terminals carry a `workspaceId` (`ws_free` for the loose ones); existing
  terminals stay loose (ADR-0026).

- **tmux list options are editable in the settings page** (`#/termset`). The
  five array options this tmux carries — `command-alias`, `terminal-features`,
  `terminal-overrides`, `status-format`, `update-environment` — are edited as
  text, one entry per line: line *n* is `name[n]`, blank lines are ignored,
  and Apply replaces the whole list. **Start from inherited** copies the
  entries you are inheriting into the editor to edit them; Reset drops the
  override. Shrinking a list unsets the indexes it no longer has, so a removed
  entry stays removed. This closes the last "shown but not editable" gap in
  ADR-0025.

### Fixed

- **Clearing an option in the global panel now unsets it on the terminals it
  reached.** The store forgot the value but nothing removed it from the live
  tmux sessions, so a cleared option kept applying until the session died. The
  per-terminal panel had always done this; the global one had not.

### Changed

- **The sidebar is four flat tabs — Workspaces, Agents, Terminals, Pins.**
  One kind per tab, no duplication: the Agents tab is a flat name-sorted
  list of free agents (the Terminals shape), the Workspaces tab holds one
  collapsible card per workspace with its agents and terminals inside, and
  the Terminals tab lists only loose terminals. The section-level collapses
  are gone — only workspace cards collapse. Every tab's empty state is one
  line plus one action. Workspaces is the first tab and the first-run
  landing; agent cards share the terminal card's flat shape (no chevron
  indent), and the gear is the last icon on the Terminals header. Below
  254px the header drops the version number so four tabs never truncate
  the name (ADR-0026).
- **Removing a workspace removes its terminals** — tmux sessions killed,
  records and settings overrides deleted with it, their tabs closed. The
  confirm dialog says how many terminals are going.

- **Agent cards rename the same way terminals do.** Hovering an agent's name
  in the sidebar paints it accent with a dotted underline; clicking it opens
  "Rename agent" and saves through `PATCH /api/agents/{id}`. A workspace
  agent still carrying the default name opens the field on the name the card
  shows, never blank. The model suffix and the rest of the row are untouched,
  so clicking anywhere else still selects the agent.

- **Renaming a terminal lives on its name.** Hovering a terminal's name in
  the sidebar now shows it as editable (accent color, dotted underline) and
  clicking it opens the rename prompt — the pencil button is gone. The
  hover action row keeps two icons, remove first and the settings gear
  last, so the gear no longer sits over the pencil's old spot.

- **One place to configure terminals.** The appearance controls (font,
  colors, cursor, padding, keys) moved from Preferences → Terminal into the
  terminal settings page, as an "Appearance — this browser" section above
  the behaviour sections — every scope on one page, each labelled with its
  reach. Preferences lost its Terminal tab; old links fall back gracefully.
  Nothing moved in storage: appearance is still remembered per browser.
  Follow-up from the owner's screenshot: the page uses the same 1080px width
  as the other wide pages (it sat at 760 in a sea of margin), and the live
  preview pins to a sane height and sticks while the form scrolls instead of
  stretching to the form's full height.

### Fixed

- **Terminal cards show where the terminal IS.** The sidebar printed the
  folder each terminal was created in, forever — a `cd` inside the pane
  never reached it, while the git info beside it was already read from the
  live path. Every endpoint that answers with a terminal (list, create,
  open, rename) now reports the live pane cwd and the git facts read from
  it — fixing the selected terminal too, whose open response used to
  overwrite the live path with the stale record.
- A workspace agent on its own workPath showed its workspace's branch
  instead of its own: agent views now carry git facts for the agent's
  effective directory, and the sidebar never pairs one directory's path with
  another's branch.

### Changed

- **Terminal and agent cards share one design.** The second line is a folder
  icon and the path — or, in a repository, a git icon (click: the graph) and
  `path / branch`. Paths are ~-shortened everywhere; the duplicate git
  button in the terminal hover actions is gone; the git icon's tooltip
  finally names the branch.

### Added

- **The whole tmux catalog is a settings surface** (ADR-0025). The terminal
  settings dialog grew into a page: the curated flags with their explanations
  on top, then every option of the running tmux — searchable, grouped by
  reach ("This terminal" / "All PiCode terminals" / "This machine's tmux"),
  the dangerous ones labelled with their consequence instead of hidden.
  Values are checked by tmux itself; what it refuses is reported in its own
  words. The options PiCode used to force in code (status bar off, clipboard
  passthrough on, extended keys) became defaults on this page, so overriding
  them is now the user's call, warning attached.

### Changed

- **Compaction progress moved from the composer statusbar into the chat**
  (Claude Code pattern, PiCode tokens): a live line at the end of the
  conversation — pulsing accent dot, “Compacting session…”, elapsed time in
  the chat's own `1m 05s` format — replaces the “Compacting” segment under
  the composer. The finished compact folds into the existing one-line
  collapsible card, now also live for auto-compacts (`compaction_end` folds
  pi's summary in without waiting for a reload), and “Nothing left to
  compact.” / failure lines stay as chat alerts. `picode-compacting`
  (localStorage) still survives reloads and panel rebuilds.

### Added

- **Terminal settings** (ADR-0024). A gear beside **+ New terminal** opens the
  defaults every terminal inherits; a gear on a terminal's row opens what that
  one changes. Each field is inherit / on / off, and the inherit segment shows
  the value it falls back to. Changing a default reaches every terminal that
  has not set its own, live and without reopening anything. Today the one
  setting is whether the mouse belongs to the terminal — on, the wheel scrolls
  a TUI that ignores the mouse; off, dragging selects text the usual way.
  A change reaches only the terminals and agents this PiCode owns: the list
  comes from its own records, never from `tmux list-sessions`, which answers
  for the whole machine.

### Fixed

- **The wheel scrolls again in a TUI that does not track the mouse** (Pi's).
  Terminal sessions get `mouse on` back, the default that was removed on
  2026-08-30 to recover native text selection — it took Pi's scrollback with
  it, while Claude Code was unaffected because it enables mouse tracking
  itself. Copying still leaves the pane: a copy-mode drag emits OSC 52, which
  the browser terminal now handles. Per-terminal opt-out lands with ADR-0024.

### Added

- **Pi update card in System**: installed → latest with a Copy command (`pi update --self`) and **Update now**, which runs the self-update and reports the new version (running agents keep the old one until restarted). The registry check rides on GET /api/system with a 6 h cache.

### Fixed

- `picode install` and `picode deploy` refuse a binary built without the UI
  inside, instead of handing the service one that shows "the UI has not been
  built yet" in the browser. `make build` produces the right binary; a plain
  `go build` does not (ADR-0023).

### Fixed

- An agent whose folder is a *subfolder* of the repository now shows on the git
  graph. It never did: the repository was resolved one level too high for any
  directory below the top, so the agent was quietly left off the branch it was
  working on.

### Fixed

- `picode install` and `picode deploy` work when run outside a login shell — a
  script, a cron job, an agent. They used to copy the new binary and then fail
  to restart the service, leaving the old one running and the update looking
  done. They now find the user's service manager on their own, and when they
  genuinely cannot, they say what to do and stop **before** copying anything.

### Fixed

- `POST /api/workspaces/{id}/agents` accepts `workPath`, so a workspace can
  hold agents in sibling git worktrees. It was hardcoded empty, which meant the
  case the git graph exists for — seeing which agent is on which branch — could
  only be assembled out of free agents.

### Changed

- The built UI is no longer committed to the repository (ADR-0023). Released
  binaries are unchanged — they still carry the UI inside. For contributors:
  `make build` works as before, and `make dev` now needs `make web` once on a
  fresh clone. A UI change stops rewriting ~133 generated files per commit.

### Changed

- **Terminals no longer wear tmux's skin.** PiCode used to force tmux's own
  mouse mode on every session, so dragging entered tmux copy-mode instead of
  selecting text. It was there to keep the wheel working, and the wheel no
  longer needs it — selecting with the mouse works again in terminals and in
  agent TUIs. An app that wants the mouse (Claude Code does) still gets it, and
  Shift still bypasses that, as in any terminal.

### Added

- Copying inside a PiCode terminal now reaches the system clipboard. A copy in
  tmux's copy-mode, or one an app performs itself (`OSC 52`), lands where you
  can paste it. Reading the clipboard from inside the terminal stays
  unsupported on purpose.

### Added

- **Workspace Sessions view** (`#/sessions/<id>`, folder icon in the sidebar): every Pi session under the workspace folder — size, age, messages, cost, provider/model, and whether it is an agent's current session. Actions: Open with… (switches a workspace agent to that session, no copy), Compact (in-use), Delete (orphans only, with confirm; in-use is blocked and says why).
- **All-folders Sessions view** (`#/sessions`, "All folders →" link): every Pi session on the machine, grouped by folder — workspaces first, others marked "not a workspace". Same actions; Open with… only where a PiCode workspace owns the folder; Delete is validated against the machine's sessions root.
- The user menu (bottom of the sidebar) has a **Sessions** item that opens the machine-wide view.
- The Sessions card no longer repeats "Back to agents" in the all-folders mode — the page header Back is the way out; "All folders →" stays on the per-workspace card.
- **Auto-clean orphans**: Off / 30 / 60 / 90 days in the Sessions view. Orphan sessions (not the current session of any agent) untouched for that long are deleted — swept at boot, daily, and right after the setting changes. Default Off.

### Fixed

- A workspace that never chatted showed a raw error in the Sessions view; a missing sessions directory now reads as zero sessions.

### Added (prior)

- **Git graph.** The repository icon beside an agent or a terminal in the
  sidebar opens the commit history as a tab: branches, merges, tags, and the
  agents living in each worktree shown on the branch they are on. One graph per
  repository, so two agents working in two worktrees of the same repo share a
  single tab. Clicking a commit opens its diff below the graph, one card per
  file. Read-only — nothing in it changes the repository (ADR-0022).

### Fixed

- The green bar under the agent TUI is gone: agent tmux sessions now get `status off` at creation and on every attach (first-class terminals already did), which also gives the TUI one extra row.
- The agent's Pi TUI now renders through the same terminal surface as first-class terminals (one xterm.js engine, same padding, scrollbar, and fit) instead of the old dock wrapper; in managed mode the view shows a one-line hint with an Open TUI action.
- After /compact, reloading no longer resurrects pre-compaction history: the transcript window now starts at the last compaction boundary (exactly what pi replays), and the summary renders as a collapsible "Session compacted" card instead of a giant message.
- The "run /compact" hint no longer nags a session that was already compacted; it now says plainly that the file stays large on disk (cold boots stay slow until pi loads sessions lazily).
- /compact shows a live "Compacting m:ss" segment with spinner in the composer statusbar — it survives the TUI→managed switch, page reloads (persisted per agent), and closes on the server answer or `compaction_end`.

### Changed

- Transcript endpoint responses now include `compacted` (a boundary was found) and the window/`total`/`remaining` count only post-boundary events.

- The sidebar spinner now also shows when the agent is working from the terminal (TUI), not only from chat.
- /compact shows "Compacting session…" in the thread and closes the line when pi finishes — no more silent minutes on huge sessions.
- Reload lands back in Terminal when the agent was being viewed in the terminal (view mode persists per agent).
- Opening an agent loads only the newest slice of the transcript; Load earlier fetches older turns from the server. Huge sessions no longer dump 100+ MB into the browser.
- Sending shows the message and Working immediately; the server answer reconciles after.
- Huge sessions render the last ~60 turns; older ones load on scroll (no more whole-history lag).
- From a Pi session copies provider, model, and thinking from the JSONL.
- Typing in the composer no longer re-renders the whole conversation on each key.
- From a Pi session: search, no horizontal scrollbar, list scrolls above Cancel.
- An empty workspace no longer shows a ghost agent row.
- Closing PiCode Desktop no longer leaves a process behind holding WSL open. Ending it from Task Manager, or a crash, now shuts the keepalive down too.
- `picode-desktop doctor` no longer tells you to run the installer on a machine that is already fully set up.
- Setting PiCode up no longer reports the service as "present but not enabled" when it is running fine. Part of the setup runs as another account, and from there the answer was about the wrong account entirely; it now says it cannot tell, and the part that can tell reports it.
- `picode <typo>` now says so instead of starting a second server. Any unrecognised word used to fall through and launch PiCode again — under a different account it would come up with its own data directory and its own port, with nothing to say it had happened.
- PiCode Desktop finds `picode` inside the distro even when running as root, which it does for part of the setup. It looked the name up on a PATH that never had it.

### Added

- **From a Pi session** (New agent) copies that JSONL into a new agent. The original Pi is not stopped. Known folder → that workspace; otherwise a free agent.
- `picode-desktop update` replaces PiCode Desktop with a newer release, and `picode-desktop version` says which build you have. Tagging the repository now publishes both binaries to a GitHub release, so `picode update` has something to find.
- PiCode Desktop asks for administrator rights only when installing — the tray itself never runs elevated — and shows the same PiCode mark in the notification area that the browser tab shows.
- PiCode Desktop also sets up a Windows machine that has no WSL at all: it installs WSL, restarts once (setup resumes by itself when you sign back in), installs Ubuntu, and creates your Linux account — named after your Windows one. The account has no password until you set one; PiCode does not need it. `doctor` says which of these steps are pending before you commit to anything.
- **PiCode Desktop** (`picode-desktop.exe`): a Windows tray app that starts WSL at logon, sets the distro up, and keeps PiCode reachable in the browser. `doctor` reports what it would change without touching anything; `install` applies it and registers the logon task. The tray shows whether PiCode is up and offers Open, Restart and View logs. It holds WSL open so the idle timeout does not shut PiCode down mid-session, and trusts the certificate so the browser stops warning. One binary, no installer, no runtime.
- Sidebar Terminals stay A–Z by name (opening a tab no longer shuffles the list).
- Switching terminal tabs keeps the other pane (no blank until you click an agent first).
- Ctrl+click in a terminal uses the shell's current folder (`cd` then a relative path opens the right file).
- Chat file cards open under the turn's file names (one card per click), not over the composer.
- `picode provision` sets a machine up in one command: systemd in `/etc/wsl.conf`, lingering so PiCode starts without a login, the TLS certificate, the systemd user unit, and a health check to prove it came up. `--dry-run` shows the plan without touching anything, `--json` prints it for a caller. It never rewrites what is already correct: a `wsl.conf` that already enables systemd is left byte for byte identical, and `~/.picode` is never written.
- Terminal: Shift+drag selects, Ctrl+C copies if selected (else interrupt), Ctrl+V pastes. tmux mouse still handles click/scroll without Shift.
- Sidebar Terminals list is flush like Pins (no tree indent).
- Click a path in chat opens a closable card in the thread. **Open in tab** is the same file tab as the terminal. The old split pane is gone.
- File Preview also covers png, pdf, markdown, audio, video, and 3D (`.glb` / `.gltf` via model-viewer). Binary Raw is “Can't show this file.”
- SVG and mermaid (`.mmd`) files open in **Preview** | **Raw**.
- Ctrl+click a path in the terminal does not type mouse junk (`<16;…m`) into the agent composer.
- Ctrl+click (Cmd on Mac) a path in a terminal opens it as an editor tab. Same map as the chat file pane, for that terminal's folder. Links (`https://…`) open in the browser. Bare names like `App.jsx` / `foo.js` count too.
- Settings **Keys** edits Pi's keybindings (this machine). Search, Add (press a key), Reset. Same file as `/keybindings`.
- If the server drops (restart, deploy), the UI shows **Reconnecting** with a spinner, then reloads when `/api/health` is back. Fast restarts (shorter than the poll) are caught by a `bootId` comparison, and dead WebSockets trigger an immediate check — the tab reloads itself.
- Composer **Sketch** opens Excalidraw. Insert puts a PNG on the message (same as paste). Empty board does not insert.
- Sketch Cancel / Insert are 28px, not the full header height.
- Composer chips are joined button groups: session + New on top, provider / model / thinking / mode (and kind) in one bar.
- Machine menu: Theme and Layout are one joined left-aligned control; menu entries have icons.
- Composer top-right is Expand + More. More lists Settings, MCPs, Packages (icon + label).
- Composer chips (session, New, provider, model, thinking, mode) use the same light fill as the page-icon group.
- Settings without an open agent still shows this machine’s pi config (same idea as Packages).
- Preferences → **Terminal** is two columns: controls on the left, a live xterm preview on the right.
- Preferences → **Terminal**: colors, font, size, line height, letter spacing, cursor, blink, scrollback, padding. Ctrl/Cmd + + / − / 0 still change size in the tab. Ligatures are not offered (xterm in the browser cannot join glyphs).
- Terminal tab fills the view (padding around the edges, no white strip). New shells get `TERM=xterm-256color`.
- `picode update` checks GitHub for a newer release (for a normal install).
- `make deploy` / `picode deploy` rebuilds this repo and restarts the service.
- `picode install` starts PiCode with this Linux user (systemd). `picode uninstall` undoes it. `--purge` also deletes `~/.picode`. A Windows reboot still needs WSL opened first.
- Rename a terminal from the sidebar (pencil or double-click). New ones still start as **Terminal**.
- Preferences → Appearance: terminal Light or Dark, separate from the app theme (default Dark).
- Sidebar **Terminals** (icon next to Agents). **+** opens a shell as a main tab, like an agent. Closing the tab leaves tmux running; Remove stops it. Ctrl/Cmd+` also creates one. The GUI chrome uses JetBrains Mono; the conversation stays as it was.
- File pane header is shorter. Save, Close, then Expand (fills the chat area; Escape or the button to leave).
- Drag the left edge of the file pane to resize it.
- The file pane uses the same light/dark colors as the rest of the app (gutter, syntax, selection).
- After a turn that edited files, the names sit under the work row. Click one to open it.
- On an edit in the chat, **Keep** or **Undo** each change. Undo puts the old lines back; if the file moved on, **Open** it. A whole-file write cannot restore the previous file.
- Click the file name on an edit in the chat to open it beside the conversation. You can edit and **Save** (Ctrl/Cmd+S). If the file changed on disk, Open again instead of overwriting. Binary or huge files say so in one line.
- Packages that are behind show **Update**. The user menu marks Packages when any are. Nothing updates until you click.
- `/tree` is **Prompts**. The current one says Now. Pick another to continue from there (new session; this one stays).
- `@` in the composer also lists other agents and skills (as `@agent:…` / `@skill:…`). Files still work. This is a mention, not a message to that agent.
- Session chip shows `$0.12` when the session has spent money. Zero stays quiet.
- The address bar keeps the open agent (`#/agent/…`). Reload or a shared link opens that agent. A missing id says the agent is gone.
- Unsent composer text (and Prompt / Steer / Follow-up) comes back after reload or switching agents. Images in the composer do not.
- Send still works while the agent is busy or waiting. The message shows as **Follow-up** or **Steer** in the chat (Queued) until the turn takes it. Follow-up stays on this machine until the turn ends — **Edit** or **Remove** it. Stop drops a queued Steer. Prompt while busy becomes follow-up.
- When the agent asks a question (confirm / pick / type), the chat shows a **waiting** card. Yes, No, Cancel, or type an answer — no need to open the terminal. Notify messages are toasts. The sidebar row says Waiting.

### Fixed

- Terminal wheel scrolls the terminal. If xterm has no scrollback (Pi TUI), the wheel is sent as mouse-wheel to the TUI transcript (not the composer). Stretching the xterm canvas to 100% height had also broken hit-testing.
- Terminal scrollbar is clickable (screen no longer covers it) and tmux mouse is on so the TUI gets wheel events.
- Terminal resizes as soon as its pane changes (sidebar, split, window) — not only on a browser window resize. Sidebar drag is debounced (one SIGWINCH after you stop).
- Terminal **New line** key (Preferences → Terminal): Shift+Enter by default, or Ctrl+Enter / Alt+Enter. Ctrl+Shift+C/V copy and paste.
- Agent tabs start flush (no extra left pad). Drag a tab to reorder.
- Switching agents no longer loops the tabs against the address bar (Chrome freeze).

### Changed

- Terminal Shift+Enter (and Ctrl/Alt+Enter) match the VS Code integrated terminal: xterm `modifyOtherKeys` (`ESC [27;2;13~`) through tmux attach, with `extended-keys` + `format xterm` set on attach (probed: tmux answers only DA1 to Kitty, so pi's fallback expects this format). Three defense layers in `termKeys.js`: keydown send, keypress block, and an `onData` filter that swaps any stray `\r` within 120ms of a modified Enter (Windows Chrome emits it after a canceled keydown). Ctrl+C interrupts by default (VS Code parity); **Copy if selected** (Warp-style) is opt-in in Preferences → Terminal → Keys.
- Terminal **Ctrl+C** copies the selection when there is one (Warp / Windows Terminal); nothing selected still interrupts. Toggle in Preferences → Terminal → Keys. `/hotkeys` lists terminal shortcuts.
- MCP **Sign in** no longer needs the agent running. Add or On on an OAuth server starts Sign in. One login is shared by every agent on this machine.
- MCP **Use from…** sits with Add (first chip). The card no longer says which app is mirrored — each row already has SHARED. An empty server list is one line: No servers yet.
- MCP **Sign in** is automatic like Claude/Codex: PiCode opens one tab, approve, overlay ends when the callback lands (token save continues in the background). The row swaps Sign in for **Sign out** (no extra “Signed in” label). No paste. Add still saves the server if that login cannot run (GitHub Copilot has no dynamic client registration). Sign in forces a localhost callback so Linear does not dump you on linear.app.
- MCP list API keeps env/header **keys** and hides values (token was already write-only).
- MCP server list is alphabetical by name (no more jumping while live status refreshes).
- Remove on a shared server no longer turns it back On (it was only hiding the other app). Use Off.
- Conversation source (fences, inline code, **diff cards**) uses JetBrains Mono with Fira Code ligatures.
- MCP / Packages card names the agent at the top (with an agent icon). Scope pills say **This agent** again.
- MCP, Packages, and Settings name the selected agent as the first line inside the card (not under the title). Clicking an agent in the sidebar from a pane opens that agent (`#/`) instead of silently retargeting the form.
- MCP page: no setup lecture. Missing adapter is one line + **Open packages**; Add is the empty state when the adapter is in. Unavailable scopes are hidden. Guide: [MCP](https://cfpperche.github.io/picode/guide/mcp).
- Folder picker on WSL lists Windows drives (`C:`, `D:`) and accepts `C:\\…` paths. Place chips use home/drive icons; current path is a labeled card.
- Removing an agent or workspace now offers to delete that folder's pi sessions (and a free-agent work folder) when nobody else uses it. Project folders stay. All agents in a workspace are stopped first.

### Added

- MCP **Sign out** on a signed-in server forgets that login on this machine.
- MCP **Sign in** is a real button next to On (OAuth servers that are On). Same as TUI `/mcp-auth`.
- MCP list shows live state: **Idle** / **Live** / **Failed** / **Sign in** (file Off stays Off).
- MCP Add **More**: environment on command servers; headers and Sign in / Token on URL servers.
- MCP **Use from…** is a tree: app → servers. You pick which servers. Unchecked ones stay Off. It does not copy files.
- Local backup in Preferences: folder, interval, retention, explicit **Schedule**, Backup now, Restore. Snapshots are inspectable directories (`VACUUM INTO` + hardlinks). Choosing a folder does not start the schedule.
- Preferences uses section tabs (Appearance / Notifications / Server / Backup).
- Backup now shows a step overlay (same pattern as package install/remove).
- Snapshot rows: Restore, Reveal in Explorer, Remove. Reveal on WSL uses the Windows Explorer binary (not PATH).
- Backup job card animates each step (spinner + check) instead of jumping to done.
- Package install/remove uses the same stepped motion.
- Restore uses the same stepped overlay and asks to reload when done.
- Roadmap for composer files (`@`, images, `!`) then MCP (`docs/design/composer-mcp-roadmap.md`).
- Composer `@` fuzzy-picks a file in the agent folder and inserts `@path`.
- Clip in the composer toolbar browses the workspace: images attach as chips, other files insert `@path`. Reads never leave the workspace folder. The clip lives in the bottom action row (next to mic and send).
- Composer `!cmd` runs in the agent folder (RPC bash): output streams into the chat, Stop cancels, the next prompt includes it. `!!` stays TUI.
- MCP page is a manager: list / add / toggle / remove against the adapter's own files (machine, this folder, this agent). No adapter → install CTA. No SQLite.
- Composer paste/drop sends images on the live RPC call (not stored in SQLite).
- Composer image chips stay 64px. `@` file list has a filter field and hides dotfiles until you type them.
- Click a composer or chat thumbnail to open the image full size. X, Escape, or click anywhere closes.
- Pin sketches V3: Excalidraw (blank or annotate an image). Scene + preview on disk.
- Pin file cards: type badge (PDF/ZIP/…) + always-on remove. No artifact preview.
- Pin body is a TipTap editor (saves markdown). Toolbar: bold, italic, heading, lists, code, quote.
- Pin attachments V2: paste/drop/import images and files. Bytes on disk, not in the list.
- Pin form is a route (`#/pins/new`, `#/pins/:id`). Sidebar Pins is the list only.
- Console easter egg. Free-agent status no longer 404s (was using agent id as workspace).
- Browser QA: `npm:pi-agent-browser-native` (project + machine). Skill shrunk to PiCode-only map.
- Pins: flat list + form (title, tags, markdown body). `+` on the Pins title bar.
- Sidebar tabs: Agents / Pins (title left, tabs right). QR moved to user menu → Open on phone.
- Conversation blockquotes and ```diff``` fences (same hunks as tool diffs).
- Conversation images (`![alt](https://…)`) and Mermaid fences.
- Conversation math (KaTeX: `$…$` inline, `$$…$$` display).
- Conversation markdown tables (GFM), plus strikethrough and task lists.
- Conversation source blocks: language label, copy, highlight.js tokens on PiCode colors.
- **Run** on bash / python / javascript / go blocks — executes in the agent's folder.
- Sidebar shows the pi braille spinner on an agent card while that agent is working.
- Chat expands `web_search` / `url_context` into source cards (title, url).

### Fixed

- Closed agent tabs stay closed across reload (no longer reopen the first running agent).
- Model errors (quota, 400) show in chat and as a toast; empty assistant replies are no longer silent.
- Chat applies `message_end` / `turn_end` text (Codex often does not stream `text_delta`). Sending no longer reloads the transcript (that wiped the new message).
- One assistant reply per turn — `agent_end` no longer replays every message into the chat.
- Live `web_search` opens source cards in the tool pill (same as after reload); tool dumps stay out of the assistant text.
- Packages **This agent**: only that agent, every session (`pi -e` on start). Optional isolate skips machine and folder extras.
- `/llama` dialog: URL, load/unload/download. Setup is docs, not an in-app installer.
- `/share` creates a secret GitHub gist (`gh`). Not the phone QR.
- Composer `/` lists skills and prompt templates (insert; pi expands on send).
- `/export` JSONL download, `/import` file picker, `/hotkeys`, `/changelog` (installed pi).
- Multiple logins per provider (**Add account** / **Use**). pi still has one active slot.
  Re-login on the same OAuth account updates tokens (no duplicate). Click the name to rename.
- Claude and Codex **account** login via the same loopback ports as pi TUI.
- kimi-coding is account-or-api-key (TUI parity).
- `#/providers` Add-provider wizard (full loginable set, account vs API key).
  `/login` opens Add.
- Folder field: type a path or Browse (list + create directory on this machine).

### Changed

- Providers page: no docs copy in chrome (auth.json / TUI notes live in [public docs](https://cfpperche.github.io/picode/guide/providers)).
- Sidebar tree: one indent step (12px) per level; chevron / icon / label share a grid so names and paths line up.
- Controls share `--ctl-h` 36px (shadcn h-9). Input + button in a row match.
- Create forms validate with Zod (`noValidate` — no native browser bubbles).
- New agent / workspace forms are a dialog (desktop) or Vaul drawer (mobile),
  not inline in the sidebar.
- Palette, session picker, kind chip, slash menu, and user menu use cmdk / Radix.
  Icons are lucide-react.
- Toasts use Sonner. Position and options live in Preferences.
- `/quit` stops the agent and closes its tab.
- `/session` dialog: name, file, git, model, usage, plus copy/rename/new/compact/tree.
- Sidebar agent row: provider icon, name, and model (`Grok - grok-4.6`).
- Slash hints: docs icon, italic + dotted underline, slightly more context.
- Collapsed sidebar groups show stacked provider favicons (max 5, then +N)
  instead of a count.
- License: MIT → PolyForm Noncommercial 1.0.0 for personal/noncommercial
  use; commercial/enterprise needs a signed license
  ([LICENSING.md](LICENSING.md)). Prior MIT tags stay MIT.

### Added

- Composer `/copy` `/quit` `/reload` `/logout` `/session` `/trust` open PiCode UI
  (clipboard, stop agent, restart, providers sign-out, session dialog, trust.json).
- Public docs are VitePress (Markdown in `www/` → GitHub Pages). Slash hints
  open `/commands#{id}` in a new tab. No in-app docs viewer.
- Session tree dots sit mid-card; the spine runs through the last card.
- Session tree is a chain of prompt cards; replies/tools sit on the card.
- Composer `/tree` `/fork` `/clone`: session tree dialog. Fork is a new
  session from a user prompt; clone duplicates this branch (RPC in chat).
- `/scoped-models` opens Settings; `enabledModels` patterns + default
  tools (native checkboxes) on global/workspace cards.
- Settings layers resolve by depth (workspace beats global, like skills).
  No "Pi default" empty option.
- Pi Settings S3: agent card writes provider/model/thinking and Full/Read-only
  (existing PATCH, all sessions of that pi).
- Pi Settings S2: workspace `.pi/settings.json` when the folder is in
  `trust.json`. Untrusted → 409, run `/trust` in the terminal.
- Pi Settings S1: read/write global `settings.json` (auto-compact, steering,
  follow-up, defaults). Radix switch + native selects.

### Changed

- Packages (and other pane cards) have inner padding.
- Rail tick preview uses Radix Tooltip (no longer misplaced by the rail transform).
- Settings and Packages sit in the same gray panel card as System.
- Conversation rail is compact and centered; it grows with the thread up to 360px.
- Sidebar type is JetBrains Mono (same face Tachyon ships as Tachyon Mono).
- Sidebar section + aligns right. Empty groups show "— empty" collapsed
  and a one-line placeholder expanded (never a 0 badge).
- `#/settings` is pi (composer `/settings`). PiCode theme/port moved to
  `#/preferences` (ADR-0012). Settings page is a scoped shell (S0).
- One chrome gray (`--bg-panel`); `--bg-elevated` aliases it. Hover/focus
  still use `--bg-hover`. Canvas stays `--bg-base`.
- Compact 8px overlay scrollbar on every overflow (no Windows arrows).
  VS Code / Cursor / Linear pattern.

### Added

- Plan: `#/settings` becomes pi GUI; PiCode chrome → `#/preferences`
  (ADR-0012 proposed, `docs/design/pi-settings.md`).
- Slash parity matrix: TUI `/` vs PiCode composer
  (`docs/design/slash-parity.md`). 7 of 24 have UI.
- Workspace groups collapse (agent count), hover-only action icons, and a
  real git branch/worktree line (or "local").
- Sidebar splits **Agents** (`~/.picode/work/<name>/`, optional folder) and **Workspaces** with many
  agents per folder (own model/config). ADR-0011.
- Package install scope: **This machine** or **This workspace** (`pi install -l`).
  Source field accepts npm, git, and path.
- Package install/remove freezes the page behind a blur overlay and lists
  the real `pi install` / `pi remove` steps (no more every-button Working…).
- Packages gallery shows **skeleton cards** while npm search is in flight.
  Optimistic UI is now a project bar (`docs/philosophy.md` §7).
- Composer **dictation** (mic, Ctrl+D) and **voice mode** (waveform,
  Ctrl+Shift+O). Voice replaces the composer like Grok on x.ai; silence
  sends through pi. Dictation stays in the textarea until Send.

### Fixed

- Chat column and composer share one width (`--chat-col`).
- Composer bar hides scrolled chat text underneath (no leak at the bottom).
- Voice composer mic actually requests the microphone (and retries).
  Speaker toggles spoken replies (browser TTS) instead of a dead control.
- Dictation swaps the send cluster for a live waveform + cancel/confirm
  (ChatGPT pattern) so the recording state is obvious.

### Changed

- Agent cockpit lives in the **composer**: searchable provider, model,
  per-model thinking, and **Full / Read-only** tool-set. Read-only starts
  pi with `--tools read,grep,find,ls` and restarts a live agent.
- UI split into **desktop** and **mobile** shells (one Vite app). Phone
  gets a bottom-nav ADE; desktop is unchanged. `?desktop=1` / `?mobile=1`.
- Product benchmarks now include **t3code** and **paseo** alongside
  Cursor (`docs/benchmarks/`). Studies required before substantial UI.
- Sidebar is resizable (drag the right edge; 180–480px; remembered).
- **Settings is PiCode-only.** Providers and MCPs are their own routes
  (`#/providers`, `#/mcps`), listed in the user menu next to Settings and
  Documentation. Documented in `docs/architecture.md`.

### Added

- Chat body type is **16px / 1.7** (Claude/Grok reading size); chrome stays 13px.
- Date marks between turns (Today / Yesterday / weekday), ChatGPT-style.
- Conversation **section rail** (Grok-style): ticks per message, hover
  preview, jump. Native scrollbar hides while the rail is on.
- Agent work (thinking + tools) collapses to **Worked for Xm Ys**;
  expand for the step list. Duration from pi timestamps.

- **Packages** gallery is a **2-column card grid** (adapted from
  [pi.dev/packages](https://pi.dev/packages)). Preview frame matches the
  official catalog: real `pi.image` when pi.dev has one, graph-paper
  placeholder otherwise. No invented charts.
- App icon is the official **pi favicon**.
- Assistant replies render **markdown**. Sessions can be **renamed**
  (picker or `/name`).
- Confirms use a **Radix dialog** (compact, remove workspace), not `window.confirm`.
- Click the context bar (or `/compact`) to **compact** the session via pi RPC.
- **Copy** on assistant replies.
- Composer **floats** over the conversation (Claude/ChatGPT); the
  scrollbar runs the full pane height.
- **Send** from the composer works when the agent is stopped or in the
  TUI: the prompt is queued and the agent runs in managed mode (chat).
- Composer **status line**: cwd, git (dirty *), context bar +(auto), ↑↓
  tokens, cache hit, cost, session name.
  (same facts as the pi TUI footer; no invented provider quotas).
- Stopped state uses the **composer chrome** (Run / Open terminal), not a
  floating card over the chat.
- Session **replay** paints the JSONL conversation in the chat (view-only).
- Visual QA is a **gate**: overlay geometry audit + 5-question card.
  Clipped menus fail the build ritual (see `/skill:visual-review`).
- **Sessions** in the tab strip: list / resume / new from pi JSONL files.
  `/new` and `/resume` hit that UI, not the TUI.
- Host, network, deps and About live on **`#/system`**. Settings is only
  theme and port.
- In-app **toasts** replace browser `alert()` for errors (slash commands,
  start/stop, login).
- **Install app** button (Chrome `beforeinstallprompt`). iPhone still
  uses Share → Add to Home Screen — Apple has no install API.
- Mobile **PWA**: installable home-screen app (`standalone`, Apple meta,
  service worker). start_url forces the mobile shell.
- **Devices** (`#/devices`): connected browsers. Host vs other machines
  (iPhone/Android + IP). Online if pinged in the last 45s.
- Phone trust page is a 3-step wizard, iOS vs Android from User-Agent
  (profile download / CA install / enable trust / open) — not a text dump.
- Phone QR follows the selected path: Tailscale row → `http://<tailnet>:8470`,
  LAN row → Wi-Fi IP. Mixing them caused infinite load on the phone.
- Phone trust: QR opens an HTTP page (`:8470`) to install the mkcert CA
  (iOS profile / Android .cer) before the HTTPS app — fixes “Not Secure”.
- Cert SANs refresh automatically when the LAN/tailnet IP changes
  (`mkcert` rewrite + TLS reload on the next handshake — no rebuild).
  Only this machine's Tailscale IP is offered (not the Windows host node).
- **Open on phone**: QR button in the sidebar. Drawer diagnoses HTTPS,
  bind, reachable IP, cert SAN and mkcert CA (`GET /api/share`). QR only
  when ready; otherwise a how-to of the missing actions.
- Chrome no longer scrolls with Settings: sidebar stays pinned; only the
  main pane scrolls (`100vh` lock + `#settings-view` overflow).
- Agent provider/model/thinking lives on the **agent tab bar**, not in
  system Settings. Change saves immediately (applies on next start).
- **M3 lifecycle (ADR-0009)**: catalog from `pi --list-models` (no curated
  subset); workspace form + Settings → Agents set provider/model/thinking
  (empty = inherit). Settings → Providers shows signed-in state from
  `auth.json` keys; Sign in opens the TUI and types `/login`. Settings →
  MCP is status-only (not in the wizard). Startup exclusive lock on
  `picode.lock` (overlapping-restart incident).
- **Inline diffs** for `edit`/`write` tool pills (`+N −M`, expandable hunks).
  Uses `result.details.patch`/`diff` when pi sends them; otherwise the
  replacement args. View-only — no accept/reject (not an editor).
- **N files changed** summary at the end of a turn.
- **Command palette** (`Ctrl+K` / `⌘K`): open/run/stop/terminal per agent,
  Settings. Esc / click-outside closes.

### Changed

- **UI is React + Vite + Tailwind** (ADR-0008, supersedes ADR-0004).
  Same design tokens and component classes; vanilla `app.js` is gone.
  `make build` / CI run `npm ci && npm run build` into `internal/web/public`.
  Terminal dock is resizable (drag the top edge) and can maximize to fill
  the agent tab.

### Removed

- Yellow tmux-config tip from the sidebar. UI chrome carries state and
  actions only (owner rule); the check still lives in Settings → System.

### Fixed (the dock closer was a lie)

- **`.dock { display:flex }` beat the `hidden` attribute** — `hideDock()`
  set `hidden=true` but the panel stayed painted. Same class of bug as
  the Run-CTA ghost overlay; previous QA checked the attribute, not
  `getComputedStyle`. Fixed with a global `[hidden]{display:none !important}`.
- **Terminal dock no longer has its own tabs.** The panel belongs to the
  active agent tab (title = `Terminal · <agent>`). Closing `×` hides it
  (`display:none` verified) and it stays hidden across sidebar clicks.
  Preference is per agent tab.

### Changed (owner feedback: agents must be IDE tabs; dock must not auto-open)

- **Agents now open as tabs in the editor area** (IDE convention): clicking
  a workspace in the sidebar opens/selects a tab; tabs carry a status dot
  and a closer; closing a tab detaches the terminal attach but the agent
  keeps running. Multiple agents coexist as tabs.
- **Terminal dock never auto-opens** (bug + UX): previously selecting an
  interactive agent always re-opened the dock, defeating the closer — the
  #1 "unusable" report. Dock now opens ONLY via the explicit Terminal
  action and stays closed across sidebar clicks and reloads.
- Sidebar active state syncs with tab selection.
- `agent-browser` skill gains an **exploratory QA recipe** (clickability
  sweep, open/close persistence cycles, reload-state checks, IDE
  conventions) — the lesson from this round's QA gap.

### Fixed (exploratory QA round — agent-browser skill dogfooding)

- **Ghost overlay blocked the chat after Run**: `.run-cta`'s explicit
  `display:grid` defeated the `hidden` attribute, leaving an invisible
  full-surface overlay eating every click (Send/Terminal/Stop dead).
  Fixed with `.run-cta[hidden]{display:none}`.
- **Dead zone after reload**: with all agents stopped, no workspace was
  selected and the main area rendered empty. Boot now selects the first
  workspace when none is running.
- **Interactive mode orphaned**: the anatomy redesign dropped the only UI
  path to the terminal (tmux TUI) mode — `/open` had no caller. Restored:
  the stopped-agent CTA now offers **Run agent** and **Open terminal**
  (door, not cage).
- **Raw backend errors in the form**: `store: path…: stat…` leaked Go
  internals to users; `humanizeError` maps common cases to plain language.
- **tmux warning jargon** in the sidebar humanized (raw detail stays in
  Settings → System); managed panel now greets with a connected line.

### Added

- QA evidence: `docs/screenshots/qa-m2-final-dark.png`, `qa-interactive-tui.png`.

### Added

- **`agent-browser` skill** (ported from the agentdeck-proven skill, adapted):
  stateful Chromium automation via bash — agents can now open PiCode,
  snapshot the accessibility tree, click through flows, eval JS and take
  live screenshots against the running server. Includes PiCode specifics:
  port discovery via `server.json`, HTTPS ignore-flag, the go:embed
  rebuild rule, the observed `close`+`open` session-reset gotcha, and the
  UI map. `visual-review` skill now points to it for interactive checks.

### Fixed

- **Theme buttons inside the user-menu popover were dead**: the popover's
  `stopPropagation` blocked clicks from reaching the document-level theme
  handler. Reworked outside-click close logic; picking a theme now applies,
  persists and closes the popover (Vercel-style). `cmd/uicheck` extended to
  three functional assertions: popover opens, theme click applies+persists
  (dataset + localStorage + closed), Settings link navigates to `#/settings`.

### Fixed

- **User-menu popover not opening**: a `func`-instead-of-`function` typo
  in app.js (SyntaxError at boot) shipped inside the last binary; JS
  syntax-check was not run after the final edit (process slip, noted in
  handoff). Fixed, plus two systemic guards: `Cache-Control: no-cache` on
  the app shell (vendor caches 1h) so browsers never serve stale UI after
  binary upgrades, and a new `go run ./cmd/uicheck <url>` debug tool that
  loads the app headlessly, captures console exceptions, and clicks the
  user-menu trigger to verify the popover opens.

### Added

- **HTTPS by default with mkcert trust (ADR-0007)**: serves TLS on
  `0.0.0.0` with a zero-config self-signed bootstrap; `make cert`
  (`scripts/setup-cert.sh`) upgrades to a mkcert-issued cert (installs
  mkcert if missing, SANs = localhost + LAN + tailscale, CA exported to
  the Windows trust store on WSL, optional iOS import `--ios`); weekly
  renewal via systemd timer (`scripts/install-systemd.sh`).
- **Runtime port configuration**: Settings UI (`#/settings` → Server) now
  changes the port — validated, probe-bound (busy port → clear error), and
  applied via graceful rebind (new listener up before the old one drops;
  automatic revert on failure). Precedence: UI/DB > `PICODE_PORT` env >
  default range `8445-8455` (first free port wins). Discovery file
  `~/.picode/server.json` (url/port/pid) for scripts and tooling.

### Changed

- **UI redesigned to agent-IDE anatomy** (owner feedback: "looks nothing
  like Cursor/t3code/paseo"): conversation is now the centered hero surface
  (~760px column, flat blocks with actor labels — Cursor composer style),
  tool calls render as pills inside the conversation (28px, expandable),
  composer is a rounded elevated box (textarea + Prompt/Steer/Follow-up
  chip + send), terminal became a bottom **dock** with compact tabs —
  fixing the stacked-surfaces bug (panel residue above tabs).
- New visual spec documented (`docs/design/benchmark-visual-anatomy.md`)
  with provenance note (trained knowledge, not live capture);
  `docs/design/references/` created for owner screenshots — vision sessions
  must verify/refine against them. Tokens updated: `#0e0e11` base,
  `#131317` panel, `#7c8cf8` accent.

### Added

- **M2 core — managed agents (ADR-0006)**: agents run in exactly one mode
  (interactive tmux TUI **or** managed rpc). `internal/rpc`: JSONL client for
  `pi --mode rpc` (strict `\n` framing, response correlation, event fan-out)
  plus the managed runtime — **task delivery engine** claims queued tasks,
  maps kind→command (`prompt`/`steer`/`follow_up`), gates on `agent_settled`,
  finishes delivered/failed with audit events. Verified against real pi.
- **Agent panel UI**: live streaming view (text/thinking deltas), compact
  tool-call rows (≤32px collapsed, click to expand args/result — Cursor bar),
  task composer with kind selector; state indicators (streaming/idle).
- **Mode-switch API**: `POST /api/agents/{id}/managed/start|stop` (stops the
  other mode first); `GET /ws/agent?agent=` streams events + accepts
  `enqueue` commands; workspace views now expose `agent.mode`.

### Added

- **Database layer (ADR-0005)**: SQLite store via pure-Go `modernc.org/sqlite`
  at `~/.picode/picode.db` — orchestration overlay only (sessions, creds,
  MCP and skills stay in pi's own files; never duplicated). Schema v1:
  `workspaces`, `agents` (default agent per workspace), `tasks` (prompt/
  steer/follow_up queue, delivery state machine, claim/finish for the M2
  engine), `messages` (reserved M4), `events` (audit), `settings`.
  Embedded sequential migrations; one-time import of the M1 JSON registry
  (retired to `workspaces.json.migrated`).
- **Task queue API** (M2 prep): `POST/GET /api/agents/{id}/tasks`; tasks
  persist as `queued` until the RPC delivery engine lands.
- `/api/workspaces` responses now embed the workspace's default agent
  (`agent.id`, `agent.running`, `lastStatus` cache).

### Changed

- Workspace persistence moved from flat JSON (`internal/workspace`, removed)
  to the SQLite store; API surface unchanged except the embedded `agent`
  object; tmux session names now derive from the agent id.

### Added

- **User menu** (Vercel-inspired) in the sidebar footer: local identity
  (hostname from `/api/system`), inline theme switcher
  (Light/System/Dark), Settings and Documentation links, version.
- **Settings view** (`#/settings`, hash routing): Appearance (theme cards,
  applied instantly and persisted), System status (tmux/pi/extended-keys),
  About (version, repo, license).
- **Theme system**: light/dark/system with `localStorage` persistence,
  no-flash bootstrap in `<head>`, OS color-scheme tracking, `?theme=`
  override for testing/links. Applies to app chrome only — agent terminals
  keep their own colors.
- **Live statusbar**: connection state (connected/detached) and terminal
  dimensions (cols×rows) replacing the former static hint.

### Changed

- **UI copy de-documentarized** (owner directive): teaching-steps empty
  state replaced with a minimal one (one line + one action); sidebar
  footer system info moved into Settings; statusbar hint removed —
  documentation lives in README/docs, UI carries state and actions only
  (`docs/benchmarks.md` records the rule).
- `/api/system` now reports the machine hostname.

### Added

- **M1 — Terminal grid** (complete vertical slice):
  - `picode screenshot --url --out [--full --width --height --wait-ms]`
    subcommand (chromedp) powering the visual-review loop end-to-end.
  - tmux session manager (`internal/tmux`): create/kill/list with a
    `picode-` namespace, exact-name matching (`=` prefix), sanitized ids,
    availability/version/extended-keys detection.
  - Terminal bridge (`internal/term`): WebSocket ↔ PTY (`tmux attach`);
    binary frames carry terminal bytes, text frames carry `resize`
    control messages; detaching never kills the agent.
  - Workspace registry (`internal/workspace`): file-backed JSON at
    `~/.picode/workspaces.json`, idempotent add, validation.
  - HTTP API: workspace CRUD + open/close lifecycle + `running` status,
    `GET /api/system` (pi/tmux detection with actionable warnings).
  - UI: dark-first terminal grid (vanilla ES, vendored xterm.js 5.5.0 +
    fit addon — ADR-0004) — sidebar workspaces with status dots, tabs,
    teaching empty state, auto-attach to running agents on load.
- **ADR-0004**: frontend framework decision deferred with explicit
  adoption thresholds.
- **First visual evidence**: `docs/screenshots/m1-termgrid-first-look.png`
  (capture pipeline proven; visual verdict UNVERIFIED — see handoff).

- **Visual validation loop**: new `visual-review` skill (capture → read
  pixels → judge against Cursor bar/benchmarks → verdict + evidence),
  `docs/screenshots/` for committed evidence, gitignored `var/` working
  dir, and an honesty rule in `AGENTS.md`: no visual verdict without an
  actual screenshot.

- **MCP strategy**: documented adoption of `pi-mcp-adapter` (community Pi
  extension — proxy tool, lazy servers, standard configs, Cursor/Claude/Codex
  config import) with PiCode's value-add being a visual MCP Server Manager
  per workspace/agent (M3–M4). No native MCP in Pi by design; PiCode does
  not fork the ecosystem, it orchestrates it (`docs/architecture.md`).

- **Cursor benchmark**: product patterns (agent activity feed, checkpoints,
  diff review, per-agent model picker, background-agent cards, @-mentions,
  rules management, command palette) mapped to milestones, plus an
  aesthetic/density bar with design tokens — enforced via the `uiux-review`
  skill (`docs/benchmark-cursor.md`).

### Fixed

- **Race in terminal bridge shutdown** (caught by CI's `-race` on ubuntu):
  redesign from `sync.Once`-based teardown to single-owner pty access with
  cooperative unblocking; handler returns only after both pumps finish.

### Changed

- Dependencies: chromedp, gorilla/websocket, creack/pty added (each with
  justification; go.mod toolchain follows current Go).
- **Official language set to English**: changelog, docs references and skills
  translated; language policy added to `AGENTS.md` and `CONTRIBUTING.md`.

## [0.1.0] - 2026-08-23

### Added

- **Project bootstrap**: public repository, MIT license, CI (GitHub Actions
  with gofmt/vet/test/build across linux/macos/windows), Makefile.
- **Living documentation system**: `docs/` with architecture, philosophy,
  engineering + UI/UX benchmarks, handoff (`docs/handoff.md`) and ADRs
  (`docs/decisions/`) — with an explicit contract that documentation evolves
  with the code (see `AGENTS.md`).
- **Pi agent harness**: root `AGENTS.md` (operating contract), quality skills
  in `.pi/skills/` (`quality-gate`, `uiux-review`, `handoff-update`) and
  project settings in `.pi/`.
- **Go server skeleton**: `picode` binary with UI embedded via `go:embed`,
  `/api/health` and `/api/version` endpoints, dark-first placeholder page
  with a live health check.
- **Initial decision records (ADRs)**: browser app served by a single Go
  binary (0001), dual-channel tmux+RPC agent control (0002), dependence on
  user-installed `pi` (0003).

[Unreleased]: https://github.com/cfpperche/picode/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/cfpperche/picode/releases/tag/v0.1.0
