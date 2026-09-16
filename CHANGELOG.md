# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

**Agent contract:** every commit with a user-visible change MUST add an entry
to the `[Unreleased]` section. The repository's official language is English
(see `AGENTS.md`); changelog entries included.

## [Unreleased]

## [0.3.1] - 2026-09-15

### Fixed

- **The tray reopens the shell again.** Closing the shell window destroyed it
  instead of hiding it, so tray click and **Open PiCode** silently did
  nothing. Close now hides the window; reopening returns to the same page.
- **No more console window beside the shell.** The shell built as a console
  app, so every start opened a terminal window titled with the exe path that
  lived as long as the tray icon. It now builds as a GUI app (debug builds
  keep the console).

## [0.3.0] - 2026-09-15

### Added

- **Terminals appear in Agent permissions** (ADR-0143): the section now lists
  the terminals you started in PiCode beside the managed agents — name, tier
  and domains, with the same editor — and says plainly that a `pi` started
  outside PiCode has no identity and always reads the tab on screen.

- **Muse Code and Antigravity accept native handoffs.** `Continue in…`
  now offers them in `native` mode, not just `brief`: the conversation
  lands as a session their own CLI lists and resumes (Muse: an index row
  plus a session log; Antigravity: a summaries row plus the brain
  transcript). Both were proven against the real CLIs, including a
  same-shape round trip before the write counts.

- **Terminals are listed as principals in the browser grants API** (ADR-0143):
  `GET /api/browser/policies` returns each terminal with its effective grant,
  and `POST /api/browser/policy` accepts `term` beside `agent` — the key
  carries the namespace (`term:<id>`), so a terminal can never widen an
  agent's grant or the other way round. The settings rows that show them are
  the next slice.

- **One release, two binaries, one resident to swap (ADR-0142, slice 3).**
  The release workflow builds `picode-shell-windows-amd64.exe` beside the
  Go binaries (Rust + cargo-xwin recipe, proven on a fresh Ubuntu 24.04
  container) and stamps the tag into the shell. `picode-desktop update`
  downloads both exes, verifies them against `SHA256SUMS` — refusing an
  unverified binary like the daemon's updater — and swaps both or neither,
  rolling the tool back if the shell fails. `scripts/desktop-swap.sh`
  relaunches whoever the `PiCodeDesktop` task points at (tray today, shell
  after the migration).

- **Terminal agents identify themselves to the built-in browser** (ADR-0143):
  the `browser` tool now sends the house identity tuple — managed agent id
  first, then the PiCode terminal id — and the daemon resolves a terminal's
  own grant (`browser.policy.term:<id>`) beside the existing per-agent key.
  A caller with neither stays read-only on the tab on screen.

- **The shell tray reaches disk parity (ADR-0142, slice 2).** The tray now
  shows the disk line (`WSL … · ≈… held · C: … free`, with the `low`
  warning), a Give-back item with the readiness interlock, an explicit
  stop-the-distro confirmation and a measured result, plus Restart PiCode
  and View logs — every duty the Go tray menu owned. One board composes
  status, disk and tooltip so no timer erases another's write; the
  keepalive re-arms itself after a compact.

- **Antigravity launch is editable.** Same honest shape as Muse Code in
  the previous slice: real defaults editor, profiles, the Customize
  checkbox and the row menu's **Launch settings**. No hook surface in
  its build either, so Activity reporting stays off with the one-line
  note and `integration: true` is refused.

- **ADR-0143**: terminal agents are principals in the browser permissions —
  the identity tuple (managed agent → terminal → unmanaged), the
  `term:<id>` grant key and the decision table behind it.

- **The shell is becoming the only Windows resident (ADR-0142, slice 1).**
  `picode-shell.exe` now holds the WSL distro open with a supervised
  keepalive, polls daemon health every 5 seconds, and reports it in a tray
  status line and tooltip — the duties the Go tray owned. `--hidden` starts
  the resident with no window (the logon-task mode); a second launch brings
  the window up. `picode-desktop startup-repair --retarget-shell` moves the
  `PiCodeDesktop` task to the shell, refusing foreign tasks and missing
  executables; plain repair now accepts either resident.

- **Muse Code launch is editable.** Its Launch tab has the real defaults
  editor (executable, arguments, PATH, environment), profiles, the
  Customize checkbox and the row menu's **Launch settings** — the gaps in
  the terminal `···` menu are closed for Muse. Sessions and resume are
  unchanged.

- **JavaScript** (Settings ▸ Browser ▸ Browser permissions): sites may use
  JavaScript — on by default, applied to every open tab at once and to the
  ones created later, through the platform's own setting.

- **ADR-0142: the shell becomes the only Windows resident.** The Go systray
  (`picode-desktop.exe --tray`) retires; `picode-shell.exe` takes autostart,
  the WSL keepalive, health/disk polling and the single tray icon, while the
  Go binary stays as the headless CLI tool the shell drives. Window close
  hides, tray Quit exits. Migration runbook in
  `docs/plans/retire-go-tray.md`.

- **Site settings** (Settings ▸ Browser ▸ Browser permissions): one dialog
  with the six kinds that matter — camera, microphone, location,
  notifications, clipboard, autoplay — each set to **Allow**, **Block** or
  the platform's own default, applied to every site through the shell's
  policy, plus **Recent decisions**: every standing the browser recorded,
  with the site, the kind and a Reset. Until the Ask prompt lands, a kind
  with no choice follows the platform's default (deny), which the dialog
  says plainly.

- **Site permissions, the shell half** (Settings ▸ Browser): every tab now
  answers permission requests — camera, microphone, location, notifications,
  clipboard, autoplay, sensors, MIDI, fonts, file system — from the policy
  the user set (`btab_set_permission_policy`), and reports each outcome as
  `btat://permission`. The app records the standing through the daemon's
  API, so the Site settings dialog has real data to list and edit.

- **Site permissions, the data half** (Settings ▸ Browser): the daemon now
  keeps one standing per site and kind — camera, microphone, location,
  notifications, clipboard, autoplay, sensors, MIDI, fonts, file system —
  with `GET/POST /api/browser/permissions`, `DELETE
  /api/browser/permissions/{id}` and `POST /api/browser/permissions/clear`
  (optionally one kind), all tested, on migration 050 and the events
  invariant. The shell feeds it and the Site settings dialog reads it in the
  next slice.

- **Muse Code and Antigravity sessions can be continued elsewhere.**
  Both are now handoff sources: `Continue in…` appears on their session
  rows, offering every CLI that can receive the conversation. Muse reads
  through its own `export --session` (official transcript); Antigravity
  reads the CLI's per-conversation transcript, deliberately not the
  SQLite protobuf (unversioned field numbers). Reasoning never travels;
  oversized sessions fall back to the brief, which both already support.

- **Muse Code and Antigravity can receive a brief handoff.** `Continue in…`
  (on a session row, a terminal row, or the pane menu) now lists them as
  destinations: the conversation travels as an opening brief —
  `muse "<brief>"` starts the session with it,
  `agy --prompt-interactive "<brief>"` does the same. They are the first
  prompt-only targets (no native session import yet), so `brief` is the
  only mode offered until their writers land.

- **Downloads** (Settings ▸ Browser): Location shows where the built-in
  browser saves files (the system Downloads folder until changed), with a
  Change dialog; "Ask where to save downloads" decides between a save
  prompt and writing straight to the folder; Download history lists every
  file with its size, time and outcome, searchable, with Open file /
  Show in folder / Copy path / Remove per row and a two-step Clear all.
- The shell reports each download (start and outcome) through
  `btab://download`; the list lives in the daemon store
  (`GET/POST /api/browser/downloads`, `POST /api/browser/downloads/status`,
  `DELETE /api/browser/downloads/{id}`, `POST /api/browser/downloads/clear`).

- **Every Agent CLI now shows the same tabs.** Launch, Terminals, Sessions,
  Providers, Settings, Keyboard, Packages and Connectors appear for all of
  them, so the pane has one shape instead of three. The tabs a CLI has no
  native editor for — today all of them except Pi — say
  **"… for <CLI> are in development — coming soon"** instead of pointing at Pi,
  and a deep link on a phone scrolls its own tab into view.

- **The tmux app shows the machine's tmux servers.** A new **Sockets** tab
  lists every tmux server this instance can see — the default socket, every
  named one, and PiCode's own dedicated socket — with what each holds: N
  session(s), how many are PiCode's, and whether anything is listening (a
  socket file with no server is labelled as the leftover it is). The tab
  badge counts the running servers, and the row for PiCode's own socket says
  where new terminals will land.

- Custom provider form: per-model **display name**, **input** (`text`,
  `image`) and **cost** (USD per 1M tokens, all four rates or none), a
  **streaming usage** compat flag, and a provider string per thinking level
  (`xhigh` → `high`), so a gateway like `meta-ai` is fully describable in
  the GUI — no hand-edited `models.json`. Clearing a row field removes the
  stored key; Edit prefills everything the file holds. The API key flow is
  unchanged: a literal credential into `auth.json`, exactly as
  cheaperinference.

- **Servers** in the inspector rail: what is listening on this machine, who owns it (the PiCode terminal or agent whose process holds the port) and what the page calls itself, with one **Open** that shows it in PiCode's own browser tab.
- A loopback URL printed in a terminal (Ctrl+click, or the pane menu's **Open …**) opens in PiCode's own browser instead of the system browser — unless Browser settings says local development sites should open externally, in which case nothing changes.
- Without the desktop app, that browser tab renders a page from this machine in a frame, with one line saying so, and its address survives a reload.

- `POST /api/browser/history/delete` removes a selection of visits in
  one transaction.

- **PiCode terminals get their own tmux server.** Each instance runs tmux on
  a socket inside its data directory (`~/.picode/tmux.sock` for the main
  install), separate from your personal tmux: a `kill-server` on one side can
  no longer take the other down, and scratch/QA instances stop creating their
  sessions in your tmux. Attach from your own shell with
  `tmux -S ~/.picode/tmux.sock attach -t picode-…`; inside a PiCode terminal
  everything works as before.
- Terminals created before the change keep running: the daemon follows both
  servers while they drain, and the terminal list, status and tmux app show
  one fleet. They move to the new server when they are restarted or recreated.

- **The tmux guard has a switch.** Preferences ▸ Terminal (Terminal defaults)
  gains a **Safety** section: the tmux guard shows its state and toggles
  between *On* (refuses `kill-server`, pattern kills and other terminals'
  sessions inside PiCode terminals) and *Off*. The line says the change
  applies to terminals opened from now on, links to how it works, and keeps
  the switch honest under failure — a failed refresh says so and offers
  **Try again**, a failed toggle reverts and reports.
- The wiring status now installs the guard on first read, so a fresh instance
  reports the default-on state instead of "off" until its first terminal.

- **Antigravity sessions.** The Antigravity pane has a **Sessions** tab like
  Muse Code and the adapter CLIs: every conversation Google's CLI keeps on this
  machine, with its folder, age, size and turn count, grouped by folder and
  filtered to the workspace you are in. **Open in terminal** resumes that
  conversation (`agy --conversation <id>`) in its own folder. Read-only.
- Sessions for a CLI whose history exists before it has an adapter: the pane
  shows the tab whenever the server reports a session source, instead of only
  for CLIs with activity reporting.

- **tmux server loss is now recorded while it happens.** A daemon-side watch
  probes the tmux server every 15 seconds: a server that dies with sessions
  running gets a `terminal.server_lost` entry (last known session count,
  socket and when it was last seen) in the feed and a line in the daemon
  log; its recovery gets `terminal.server_back`, and a running server whose
  session count drops by five or more between probes leaves a log line.
  Until now a lost fleet was only reconstructed on the next daemon start.

- **tmux guard.** PiCode terminals now run a `tmux` wrapper that refuses
  server-wide kills (`kill-server`, `kill-window`, `kill-pane`), pattern
  kills, and kills of sessions another terminal or the user created. It
  allows exact kills of sessions your terminal created and stamps sessions
  you launch with your terminal's mark, so cleanup stays possible. On by
  default; every refusal is explained on the pane and logged to
  `<dataDir>/tmux-guard.log`.

- **Clear browsing data** (Settings ▸ Browser ▸ Browsing history): one
  two-step button clears cookies, site storage, cache, service workers,
  download history and WebView2's browsing history from the shared work
  profile — saved passwords and autofill are deliberately untouched (the
  Autofill switches own those). Our own history list clears with it.

- **Muse Code sessions.** The Muse Code pane has a **Sessions** tab: every
  conversation the CLI has on this machine, with its folder, model, age and
  size, filtered to the workspace you are in. **Open in terminal** resumes that
  exact conversation (`muse --resume <id>`) in its own folder. Read-only —
  PiCode lists and resumes, it never rewrites Muse's session store.
- Sessions for a CLI whose history exists before it has an adapter: the pane
  now shows the tab whenever the server reports a session source, instead of
  only for CLIs with activity reporting.

- **Autofill and passwords** (Settings ▸ Browser): Password manager and
  Contact info switches push straight into the work profile — every
  webview applies them at creation and live on change.

- **Open destinations** (Settings ▸ Browser): web links and popups, and
  local development sites, each open in PiCode (adopted as tabs) or in
  the system default browser. The work browser's shell hands external
  destinations over with a scheme-checked `btab_open_external`.

- **Show full URL** (Settings ▸ Browser ▸ Address bar): on — the address
  bar keeps the path, query and fragment (today's behavior); off — site
  origin only. Saved per machine, live on every open tab.

- HTML previews run on their own address (`<ticket>.localhost`), so a page can keep settings, register a worker and use its own storage — and neither **Save** nor **Reload** resets it.
- When the browser cannot open that address, or when PiCode is reached from another machine, the preview falls back to the sandboxed frame with one line above it saying so.

- **Browsing history (slice 3, first half)**: the work browser records one
  visit per navigation — a URL re-reported updates the newest row in place
  instead of piling up (typed flag sticks once set). Settings ▸ Browser
  gains a **Browsing history** section: newest-first list, per-row Remove,
  and a two-step **Clear history**. The address-bar dropdown reading this
  store is the next increment.

- **Muse Code and Antigravity in Agent CLIs.** The catalog lists Meta's `muse`
  and Google's `agy`: installed or not, Check setup, **New terminal**, which
  runs the CLI in a PiCode terminal with its own defaults, and **Check for
  updates** in the same ⋯ menu every other CLI uses — Muse Code against its
  release channel, Antigravity against the vendor's release manifest. There
  are no launch settings, no session list and no activity state for these two
  yet. Both wear their vendor's mark (Meta, Antigravity) wherever the CLI
  appears.

- HTML previews render unsaved editor changes too: the pane hands the buffer to the preview, and saving the file returns it to disk.

- **The grants editor lives in Settings ▸ Browser**: one row per managed
  agent with its effective access — Read (the tab on screen), Act or Full
  plus the hosts it may reach. Pasted entries are forgiven (scheme, path
  and port stripped); the change saves per row and is live for the
  agent's next command. Slice 4 of the desktop browser plan is complete.

- HTML previews reload themselves: editing the page or any file it loaded on disk refreshes the frame (served-asset watch over SSE), and a page too large for the text editor still previews from disk.

- HTML files preview as real pages — scripts run, styles and images load from their folder — in a sandbox that never reaches PiCode's session, with Reload and Open in browser.

- **The shell enforces the browser grant at navigation time**: once an
  act-tier command drives a tab, agent-caused loads outside the agent's
  domains — script redirects included, not just the `navigate` verb the
  daemon pre-checks — are cancelled. User-initiated navigation is
  untouched; navigating the pane from its own toolbar disarms the gate
  until the agent acts again.

- **A public Changelog on the docs site.** Newest first, including what is
  still Unreleased. Not a blog — the same Keep a Changelog file, readable
  next to the guides.

- **An MCP cookbook.** The MCP page is the index. Gmail and DeepWiki each
  have their own page: install, sign-in if any, and how to revoke.

- **A Configure overview.** Which screen edits which file, the one
  workflow that does not mix Preferences with pi Settings, and a worked
  example for a custom endpoint.

- **Agent browser split (ADR-0135)**: a terminal that hosts an agent — the
  agent's TUI or a CLI launch — can open the work browser beside it
  (right-click → Open browser), bound to that agent so the `browser` Pi tool
  acts on the pane on screen. The split resizes (drag), maximizes, and closes
  from the pane's own controls.
- **The split survives a relaunch**: the layout (which tabs host a pane, the
  dragged width, the maximized flag) and each pane's last url persist in
  localStorage. After the desktop app restarts, the panes come back at the
  pages they were on — the first time a host tab is shown, its webview is
  recreated where the pane is. Panes whose agent or terminal is gone are
  pruned on boot instead of reopening to nowhere.

- **Archiving and deleting snippets now work on the phone.** A snippet's page
  has **Archive** / **Unarchive** and **Delete** (with a confirm naming the
  snippet), and the list has an **Archived** view next to **Active** — so an
  archived snippet is visible and can come back instead of disappearing with no
  handle on the other side.

- **The Snippets guide now covers the whole editor**: what the slug line means
  while you type (free, or in use with Save off), the placeholder table
  (default, optional, and the choices list that becomes a dropdown when you
  send), where snippets come from (starters, saving a selection, importing a
  prompt, duplicating), and that a half-written snippet no longer dies with the
  tab.

- **The phone's snippet editor does what the desk's does.** The placeholder
  table is there — default, optional, and the list of offered choices per
  `{{name}}`, with a reserved name (`{{branch}}`, `{{cwd}}`…) saying where it
  comes from instead of pretending you can set it. Broken `{{placeholders}}`
  are named under the body while you type, Save waits until they are fixed,
  and an address already in use is a line under the **Slug** field rather than
  a refusal after the tap. Choices you add keep working when the snippet runs:
  a value outside the list is refused.

- **Custom endpoints: the base-URL field now explains itself.** A hint and an
  example follow the API type you pick — an OpenAI-style root usually ends in
  `/v1`, Google's in `/v1beta`, and Anthropic-compatible endpoints differ — so
  the field stops being guesswork.
- **More thinking formats, including the two that need an object.** The format
  picker now offers what pi actually supports: `openai` (the old
  `reasoning_effort` name was written back as `openai`), `deepseek`, `qwen`,
  `qwen-chat-template`, `openrouter`, `together`, `zai`, `ant-ling`,
  `string-thinking`, plus **chat-template** and **baseten**, which reveal a
  JSON editor for `chat_template_kwargs` / `chat_template_args` with pi's
  `$var` references. A typo is explained inline instead of being saved.
- **Verify a custom endpoint for real.** The row's **Verify with the endpoint
  (1 request)** sends one minimal completion (one word in, the smallest output
  ceiling the API accepts) and reports what the endpoint said — the model, how
  long it took and the tokens it counted. The dialog can do the same before
  saving. Built-in providers keep pi's free answer.

- **The snippet editor tells you when an address is already taken** — while you
  type, not after the save: the line under the **Slug** field says which
  snippet holds `/snip:…`, and Save stays off until you pick another one. On an
  existing snippet its own address is never reported as a clash.

- **Docs: browser tools for pi** — a new guide explains what an agent can read
  in the work browser, what a grant adds (`act`: `evaluate`, `navigate`), and
  who may call it: a managed agent has its own grant, a plain `pi` TUI reads
  the tab on screen.

- **The dashboard now says what is waiting on you.** One line above the
  numbers — “3 need you · 2 questions in Inbox · 1 terminal waiting” — with a
  single action that goes to the Inbox, or to the waiting terminal when the
  Inbox is empty. It appears only while something is blocked.

- Sidebar Apps tab: an APPS header and a live "Search apps" filter (Escape clears), with tiles sorted alphabetically — same pattern as the Pins tab.

- **Custom endpoints: load the model list instead of copying it.** The Add/Edit
  endpoint dialog has a **Load models** button under the ids: it asks the
  endpoint what it serves and adds the ids you are missing (your typed ones
  stay, nothing is reordered). It also fills context window and max output when
  every listed model reports the same number, and says so when they differ. It
  works for OpenAI-compatible gateways, Anthropic and Google endpoints.
- **The failure says what to fix.** A refused key, a URL that needs `/v1`, a
  host that does not resolve, or an account that cannot list: each is one line
  naming the next step, in both apps and in the same words.

- **The tmux app: every session on your tmux server, in one screen.** PiCode
  runs every terminal and interactive agent in tmux on your own server, and
  until now a session with no terminal or agent behind it was invisible in the
  product — you had to leave PiCode and run `tmux ls` to find it. Apps → tmux (on the phone, More → Apps) now shows the whole server: PiCode's sessions (with **Open**), sessions no
  longer in PiCode's records, and your own tmux sessions beside them, marked as
  not PiCode's and left alone. A row expands to the folder, command, uptime and
  attached clients; the **Server** tab carries the version, socket, client
  count and keyboard mode, plus the terminals whose sessions are gone
  (including the ones the flight recorder saw die in a restart).
- **Removing a leftover asks, and re-reads the session before it acts.** A
  session PiCode cannot attribute can be removed one row at a time, behind a
  confirmation and a receipt (session id, start time and pane process): if the
  session changed since the page was drawn, nothing happens. There is no bulk
  cleanup, because a session missing from PiCode's records may belong to
  another PiCode on the same machine — the row says so instead of guessing, and
  every removal is recorded as an audit event.

- **A grant can let an agent act.** With the `act` tier, two verbs join the
  browser tool: `evaluate` (run an expression in the page) and `navigate` (go
  to a URL). Without a grant both are refused by name, with the way to grant
  them.

- **Agents can read the page open in the work browser.** A new `browser` tool
  (`snapshot`, `screenshot`, `events`) reads the tab on screen in the desktop
  app and answers with its structure, a PNG path, or what the tab recorded.

- **Custom endpoints: pick how thinking travels on the wire.** The Advanced
  section of the Add/Edit endpoint dialog now names the thinking format —
  `reasoning_effort` (the OpenAI standard), DeepSeek, Qwen, OpenRouter,
  Together or Z.AI — instead of assuming every gateway speaks
  `reasoning_effort`. The choice is written as `compat.thinkingFormat` in
  `~/.pi/agent/models.json`; leaving it at the default writes nothing, so pi
  keeps choosing for that API type.

- **Starters in the snippets studio.** An empty page now offers six ready
  prompts — review a pull request, explain an error, write a commit message,
  summarize uncommitted changes, run the tests and fix failures, standup
  update — each opening in the editor with its placeholders, tags and address
  already filled in. Once you have snippets of your own the list collapses into
  one line you can reopen.
- **Duplicate, on a list row and in the snippet itself.** A near-miss becomes a
  copy you can edit: title gets “copy”, the address gets `-copy`, and the
  original is untouched.

- **Snippets are easier to write.** The studio editor now validates the
  body as you type — a broken `{{placeholder}}` shows a line naming the
  problem and **Save** says why it is off — and everything the grammar
  knows is editable from a table under the body: **Default**,
  **Optional** and **Enum** per placeholder, written back into the text.
  A **Try it** pane gives each placeholder a sample value (enums become a
  picker) and shows the expanded prompt, including what will be asked for
  at run time.
- **Save a prompt straight from the composer.** Select text and a **Save
  as snippet** button appears in the message bar; the same action is in
  the right-click menu as **Save selection as snippet**, so any text you
  select — in the composer or anywhere in PiCode — can become a snippet
  without retyping it in the studio. On the phone it is **Message
  options → Save as snippet**.
- **Import a prompt from another tool.** The snippets list gained
  **Import**: paste a prompt and PiCode finds its slots, offering
  `[BRACKETS]` and `UPPER_CASE` as `{{placeholders}}` — each suggestion
  can be switched off — then opens the editor with the result. Available
  on the phone too.

- Custom endpoint form (Add provider → Custom endpoint): a **Reasoning
  model** switch with the thinking levels the model answers on
  (`minimal`…`max`). The selection is written as pi's per-model
  `thinkingLevelMap`, so the levels a gateway really supports show up in the
  model picker — `max` included — instead of stopping at `high`. Edit
  prefills the same levels from the file.

- **Work browser: the shell now speaks CDP itself.** The desktop app carries
  a command bridge over the page host API, so a page's DevTools protocol is
  reachable without any open debug port. Read-tier page events (navigation,
  console, network, log) are recorded per tab for whoever polls them.

- **Continue in Pi now asks where it opens.** Pi is both a CLI and the
  platform's managed agent, so its "Continue in…" entry has a second
  menu level — *Pi agent · in the app* (a stopped agent, no installed
  CLI needed) or *Pi CLI · in a terminal*. The dialog's new **Where**
  section lets you change the choice before continuing; a brief to the
  agent lands as its first queued prompt.

- **Files: switch workspace from the tree's toolbar.** The folder line at the
  top of a Files tab is now the workspace that folder belongs to, opening as a
  picker of your other workspaces — every folder is offered, repositories or
  not, with the folder path as its second line and a check on the current one.
  Picking one re-points the tab: the same folder swaps the owner in place,
  another folder moves the tab to it (one tab per folder, as ever), and a pick
  that cannot resolve leaves the tree exactly as it was. A single workspace
  keeps the plain folder line.

- **Snippets.** Save a reusable prompt or a shell command under Tools or
  the command palette (`#/snippets`). Placeholders use `{{name}}`.
  Type `/snip:` in an agent composer to insert or send one, pick **Send
  snippet** in the command palette, right-click a running Agent CLI
  terminal and choose **Send to terminal…**, or **Run command…** on a
  shell pane — the confirm always shows the exact command before it runs.

- **Mobile Git: switch workspace from the screen.** The folder line at the top
  of a Git screen is now a control when there is more than one workspace to
  read through: it opens a sheet of your workspaces, each with the branch that
  folder is on and a check on the current one. Picking one opens the Git screen
  for that workspace, so Back returns where you were; folders that are not Git
  repositories are never offered, and a single workspace keeps the plain line.

- **Git graph: switch workspace from the toolbar.** The graph's first item is
  no longer the repository's name but the workspace its history is read
  through — a dropdown of your other workspaces, each with the branch that
  folder is on. Picking one retargets the reading folder without leaving the
  tab: sibling worktrees of one repository swap in place (the HEAD dot and the
  `this worktree` row move), and a pick into another repository moves the tab
  to it, so one tab per repository still holds. Folders that are not
  repositories are never offered, and a pick that cannot resolve leaves the
  graph exactly as it was.

- Providers: **Custom endpoint** in Add provider registers any OpenAI-compatible (or Anthropic-/Google-compatible) gateway for pi from the GUI — name, base URL, API key and model ids; no hand-editing `~/.pi/agent/models.json`. Definitions merge into pi's own models file, the key is stored with every other sign-in, and the roster gains a `custom` badge with Edit endpoint / Sign out / Remove endpoint actions.

- **Continue a terminal's conversation in another CLI.** The sidebar `⋯`
  menu, Agent CLIs → Terminals `⋯`, and the pane's right-click menu offer
  **Continue in…** for the conversation that terminal is running. The
  existing preview dialog (what travels, what is left behind) opens; the
  original terminal stays put. A memory tool that injects a “where you
  left off” note in the same folder is a separate layer, not this action.

- **Text on the canvas.** The toolbar has a **Text** element: draw a
  rectangle and a panel appears there ready to type in — a label beside a
  terminal, a note to whoever opens the canvas next. It saves as you pause
  and when you click away, and everyone looking at that canvas sees it.
  Two thousand characters; a pin is still what holds longer writing.

- **The zoom percentage is back, under the zoom buttons.** It says where the
  camera is, and clicking it returns to 100 % — the only zoom at which a
  click inside a terminal lands on the character it points at.

- Add an explicit, guarded “Activate now” action for open Agent CLI conversations that still need their first native event.

- Git guard: `main` can no longer be rewound — the reference-transaction
  hook refuses any move of `main` that is not a fast-forward, including a
  stale-ref fast-forward onto a divergent tip that silently erased merged
  work once. A deliberate rollback requires `PICODE_ALLOW_MAIN_REWIND=1`.
  The guard's functional tests live in the hooks selftest.

- Desktop shell: every window carries a dedicated app bar in the ChatGPT
  style — brand on the left, drag anywhere on the bar, double-click to
  maximize, and flat Windows caption buttons (close turns red) on the
  right. The bar belongs to the shell and works no matter which version
  of the web UI the daemon serves; the browser renders no bar.

- **Right-click the canvas.** The plane has its own menu now — the same one
  the `⋯` button opens, with **Show controls** on top.
- **Show controls hides the canvas chrome.** One switch takes away the zoom
  column, the toolbar and the minimap, so the plane is only the plane. The
  right-click menu still works with them gone, which is how you bring them
  back; the choice is remembered per browser.

- **Terminal attachments: sketch.** The desktop attach bar and the phone
  sheet gain a Sketch button — an Excalidraw pad (the dependency, not the
  pin studio) whose drawing leaves the composer as a PNG. The chip reopens
  for editing until Send.

- **The Inspector follows the panel you are on in a canvas.** Click a
  terminal or agent panel and the rail shows that one's files, changes and
  branch — until now a canvas left it on whatever it was showing before, or
  empty. Focusing a note, file or diff panel leaves the rail where it was.

- Desktop shell: the tray item is now **Management** — a window with three
  views over the WSL distro. **Disk** shows what the distro holds and runs
  the Give back flow with live progress; **Clean** measures the prunable
  caches (`picode clean --list`) and prunes the selected ones — Pi sessions
  and the PiCode database are never cleanable; **Config** edits
  `.wslconfig` (memory, processors, swap, sparse disk) with an automatic
  backup, leaving unknown settings untouched; changes apply at the next
  full WSL restart.
- `picode clean` — new subcommand that prunes the caches `picode disk`
  measures (`--list` to measure, `--apply id1,id2 --yes` to prune,
  `--json` for tools). Data directories are refused even when asked for by
  name.
- Desktop shell: sharper taskbar/window icon (the icon file's largest
  image now comes first) and the shell's local pages use the product's
  design tokens.

- **Packages: Configure now works for descriptor-declared packages.**
  pi-roles was the only package with a Configure button; any extension whose
  configuration is described (PiCode's catalog, or a `picode.config` manifest
  in the extension itself) now shows one, backed by a form that edits the
  package's own config file. Configurable today: pi-web-search (the model
  that backs native web search) and pi-compact (compaction policy — enabled,
  token/percent triggers, floor, cooldown, summarizer model, thinking and
  instructions). The files stay the packages' own source of truth: unknown
  keys survive saves, unset fields stay unset, and a broken file is reported
  and never silently replaced. No description for a package you installed?
  "Describe config…" lets you describe its configuration yourself — create,
  edit and delete the description at any time; PiCode stores it locally.
  Public docs (docs-site/guide/packages.md) now explain configuring and
  describing packages.

- **PiCode Desktop: give the held space back from the tray.** When the disk
  file is holding space the distro freed, the tray offers **Give back ≈92
  GB…**. It first asks PiCode whether anyone is working (the same interlock as
  a deploy — someone mid-turn is named, and it stops), then asks once in a
  dialog that names the cost, stops Ubuntu, converts the disk file to
  **sparse**, starts Ubuntu again and reports the before/after measured on the
  file. From then on WSL returns freed blocks on its own.
- **`picode-desktop disk-compact`** — the same flow from a terminal.
  `--dry-run` prints the plan and stops nothing; `--yes` runs it; `--force`
  overrides the working check; `--method optimize-vhd` compacts with Hyper-V's
  Optimize-VHD from an administrator terminal (Windows Home: upgrade WSL for
  the sparse path instead).

- **PiCode Desktop: the tray now tells you what the disk is doing.** One line
  under the status — `WSL 218 GB · ≈92 GB held by Windows · C: 27 GB free` —
  refreshed every five minutes, ending in `— low` under 20 GB free, with the
  tooltip carrying the same sentence and the command that shows the rest.
- **`picode-desktop disk` reports both halves of a WSL disk in one screen.**
  The Windows side (the VHDX path from WSL's own registry, the file's real
  size, whether it is sparse, free space on the volume) and the distro side
  (filesystem, the caches with the exact command that gives each back, the
  top-level breakdown of home). It shows **held for nothing**: the space the
  distro has already freed that the disk file still occupies — invisible in
  Explorer, and the reason a full `C:` stays full. Read-only.
- **`picode disk` lists what occupies the machine and what is safe to
  reclaim**, one line per item with `safe`, `redownload` or `data`, plus the
  paths it could not read named as such instead of folded into a category.
  `--json` feeds the tray and, later, the Storage app.

### Changed

- The tmux watcher integration test and the kill-server test address a server they own with `-S` instead of rebinding the process-wide `TMUX_TMPDIR`: a test that ends a server can no longer strand other tests — or a background goroutine — in the same binary on the server it dismantles. The kill test keeps asserting the directory guard; the watcher test asserts the suite's namespace is untouched.

- **No Activity reporting for Muse Code, stated plainly.** Its build
  offers no hook surface, so the toggle is replaced by a one-line note
  ("its terminals stay Open") and the server refuses `integration: true`
  with 400. The startup seed that switches Activity on for empty configs
  skips hookless CLIs.

- The work-browser tab no longer spends a line telling you it is a frame: the page starts directly under the address bar.
- The address bar's Open button fills the pill it shares with the field (its hover covered only a 28px strip) and uses the return glyph instead of a text `↵`, which sat high in its box.

- The setup tabs are placeholders until each CLI's editor exists; Pi keeps its
  real Providers, Settings, Keyboard, Packages and Connectors panes. No editor
  is implemented for another CLI before its launch settings match the others.

- **Autofill and passwords** rows now open **Manage** dialogs instead of
  hiding the switches on the page: Password manager offers "Offer to save
  passwords" and a two-step delete of everything the profile stored;
  Contact info offers "Save and fill addresses" and the same delete for
  saved form data. Each dialog says plainly that saved entries live in
  this machine's browser profile and cannot be listed or edited per entry
  — WebView2 exposes the switches and a wipe by kind, nothing that reads
  entries back.

- **Browsing history** now presents the data the way the reference does,
  inside the dialog: a search field, day groups that collapse (newest
  open, "Today"/"Yesterday" named), and a row per visit with its site
  icon, host, visit time and a per-row menu (Copy link, Remove from
  history). Rows can be selected and removed in one go, and "Clear
  browsing data" sits on the section header.

- Scratch instances (QA scratch, docs fixture) and their cleanup operate on
  their own tmux server, so their sessions no longer appear in — or collide
  with — your tmux.

- **Custom endpoint is now Custom provider.** Titles, buttons, menus,
  toasts and the Providers guide use the new name (matching ADR-0129);
  routes, API paths and behavior are unchanged.

- **Clear browsing data** now opens a dialog in the reference shape: a
  time-range picker (hour / 24 hours / 7 days / 4 weeks / all time) over
  a checklist of what to clear — browsing history, cookies and site
  data, cached images and files, download history, autofill form data
  and site settings — each showing what it will remove.

- Web: the editor tab strip (browser and desktop) now follows Chrome's CR23
  tab language — raised active tab, hover pill, idle separators — instead of
  the rectangular accent-underline tabs. Overflow, keys and the all-tabs list
  are unchanged.
- Header matches Chrome's tab strip: 41px tall with 35px tabs inset 6px
  from the top, 20px idle separators, 8px corner radius
  (`docs/benchmarks/2026-09-15-chrome-tab-header.md`).
- The selected tab now reads connected to the page on `/browser` and
  `/desktop`: it shares the toolbar background and no hairline cuts it
  off, while the sidebar head, tab strip and inspector head read as one
  header bar with a single hairline between them and the content and no
  pane verticals crossing it.

- **Settings ▸ Browser follows the reference layout**: a page title and
  lede, section headers, and cards of rows whose title and description
  sit left with the control pinned right, divided by hairlines. Switches
  are pills, selects and actions are outline buttons, and Browsing
  history opens in a dialog. The chrome around the page stays PiCode's.
- The master **Browser** switch is real: off refuses every browser verb
  for every agent server-side until it is turned back on.

- **AGENTS.md**: the scratch-tmux rule now names the measured mechanism —
  `$TMUX` outranks `TMUX_TMPDIR` (so a "isolated" client still talks to the
  server it was started in), `-L` overrides `$TMUX`, and `TMUX_TMPDIR` only
  counts when that directory exists.

- **Custom endpoint is a page, not a dialog.** Add provider → Custom
  endpoint (and Edit endpoint) opens `#/clis/pi/providers/custom` with the
  form in Identity / Connection / Models sections, Name + API type side by
  side on wide screens, and Verify key beside the field it checks. The
  how-to prose moved to the Providers guide; the page carries a Setup
  guide link instead.

- **Muse Code and Antigravity keep the same launch screens as every other
  CLI.** The New terminal screen shows the CLI row and the Launch preview
  (which command PiCode will run, what it adds), and the Launch tab shows the
  plan summary instead of a one-line notice. Nothing is editable for them yet
  — the customize checkbox and launch profiles stay absent — so the screen
  reads the same and says plainly that the CLI keeps its own defaults.

- Agent CLIs describe what a row can do as capabilities: a CLI can open a
  terminal without carrying activity, launch settings or sessions.

- Web: the active editor tab is the elevated surface (the white the sidebar
  reads as), so it no longer disappears into the strip gray.

- **A stopped CLI terminal now sits in a simulated terminal window.** The
  state — what stopped it, the conversation **Resume last session** would
  reopen, and the two actions — is drawn inside a framed terminal window
  (titlebar, traffic lights, the terminal's own name and folder as the
  title), centered on the page and hugging its content instead of filling
  the pane. The page keeps the app's theme; the window is a picture of a
  terminal and is dark in both themes. Same on the phone.

- **A stopped CLI terminal is a real empty state, not a sentence in the
  corner.** Its pane now shows a mark, what stopped it (a plain stop, a
  restart that ended it, or a failed last launch), the conversation
  **Resume last session** would reopen — CLI, name and age, with the opening
  words on hover — and the two actions. A terminal with nothing pinned says
  *no session to resume* instead of offering no button and no reason.
- **The empty terminal pane follows the app's theme** (owner call): a light
  app no longer shows a black well where a terminal used to be — the ground
  is the app's own and the message reads in the app's inks, in both themes.

- **`make desktop-restart` swaps the v2 shell too**: the target now builds
  `picode-shell.exe` (Tauri, ADR-0120) alongside the tray + native host, and
  `scripts/desktop-swap.sh` stops, copies and relaunches it when it was
  running. A stale shell refuses page commands with a Tauri ACL error — its
  capability list compiles into the exe (2026-09-14: `btab_cdp_call` refused
  for days after the CDP bridge landed). `DRY_RUN=1` prints the plan and
  touches nothing.

- **Tools docs open with Where and Not this.** Browser tools, Chrome
  extension, Docker, Integrations, tmux, CLI activity reporting and
  keyboard each say what the page is, where to click, and what it is not.

- **Inbox: "Clear all done" sits beside the filter** on the Done tab, right-aligned on the toolbar row, instead of trailing the list — the same place every list page keeps its bulk action. Any app that declares a list-pane actions row gets this for free; an item's own actions and empty-state actions are untouched.

- **Chat and work docs open with Where and Not this.** Canvas, session
  messages, Inbox tools, Checklist, Compact earlier and the diff panel
  each say what the page is, where to click, and what it is not.

- **Agents docs open with Where and Not this.** Agent CLIs, Packages, MCP,
  llama.cpp, Automations and Snippets each say what the page is, where to
  click, and what it is not.

- One right-click menu for every terminal pane (managed Pi in-terminal, Agent CLI, plain shell). Chat and the composer stay on the generic menu.
- **Send to terminal…** on an in-terminal Pi pastes into the TUI (palette Send snippet too); the sheet says “Sent to the terminal.”
- **Attach files…** / **Ask Pi about this** on an in-terminal Pi use `/api/agents/{id}/drop` and `/prompt`, not a terminal id.
- **Rename agent…** / **Remove agent…** (existing cleanup confirm, never “This stops the tmux session.”). **Terminal settings** from an agent pane opens global **Terminal defaults**.
- **Continue in…** on an in-terminal Pi uses that pane’s session file and asks to continue while the TUI is still writing.
- Close browser split actually appears when the split is open.

- **Getting started installs from a GitHub release**, not `make build`.
  Download the binary, run `picode install`, open the app. Building from
  source is a separate page.

- **Public docs sidebar is Start / Use / Run / Configure / Reference.** The
  old flat Guides list is gone. Settings and Providers sit under Configure;
  a Use catalog lists every capability in one place.

- **A snippet Command no longer runs into a repository another agent is
  writing in.** The confirm step still shows the exact command; running it is
  refused with a message naming who is at work there, the same rule the
  terminal's **Run command…** already followed.

- **Docker, Inbox and tmux now read like every other page.** Their surfaces keep the app's tab strip, head actions and filter, but inside the same page frame a system route uses: a 1240px card, the view's tabs as an underline nav with count chips, the filter in the card toolbar, and the list and detail panes inside the card (side by side, stacking on narrow windows). Empty, blocked and error states are one line plus one action instead of a full-page poster.

- **The custom endpoint dialog reads field by field.** Name, Base URL, API key
  and Model ids carry a label above the control and one line of help under it,
  with a clear gap between one field and the next — the placeholders were the
  only label before, so a filled field could not be told from an empty one.
  The helper prose that read as a lecture (a file path, a protocol note) is
  gone; what stayed says what the control does.
- **Limits belong to the model, not to the list.** Context window and max
  output are one row per model id (Advanced → Model limits), so Load models
  fills each model's own numbers — a gateway whose models disagree is no
  longer refused with a blank field and an explanation. A number typed by hand
  survives a load; a row follows its id into and out of the list.
- **Advanced is a section, not a wall.** The disclosure holds grouped sections
  — Request compatibility, Thinking, Model limits — each with a legend and a
  hairline, so it reads as structure once open and stays one line while
  closed.
- **Load models and Verify key report inside Model ids**, the field that owns
  them, so a refusal line ("The endpoint refused the key (401): invalid api
  key provided. Check the API key and try again.") sits next to the action that
  caused it and never pushes the alternative route below the fold.

- **Half-written snippets no longer die with the tab.** Drafts moved from the
  per-tab shelf to the browser's durable one, so closing the tab or reloading
  brings the text back (a capture or an import is still handed over as a new
  snippet rather than announced as "unsaved changes").

- **The Fleet tile counts agent CLI terminals, not just managed agents.**
  The sidebar's Claude Code, Codex, Grok, Hermes and OpenCode terminals were
  invisible to the dashboard, which could read `1 / 1 running` beside seven
  live terminals. The tile now shows `N live` (agents + agent CLIs), states
  `working` / `need you` / `idle` / `no signal`, names each one, opens its tab
  on click, and counts plain shells apart — `no signal` means a CLI is running
  but has not reported activity, never that it is idle.

- **The handoff board is an index now, not a ledger.** Next steps still render
  inline (they are bounded and they are what a session acts on), but debts are
  one line per topic — the count, the topic file and its plan. Rendering every
  debt made the board grow with the backlog, which is what filled it; it is
  about half the size and no longer trips its budget on one honest entry.
  `make handoff` now **fails, naming the file**, when a topic file lists bullets
  outside its `## Next` / `## Debts` headings — six snippet debts were invisible
  that way for a day.

- **Six snippet debts were invisible on the handoff board.** The board renders
  the bullets under a topic file's `## Next` / `## Debts` headings, and
  `docs/handoff/open/snippets.md` had neither — so its debts never reached the
  sessions that read the board. It has the heading now, consolidated to the
  three that carry the meaning; the roadmap lives in the plan.
- Also pruned there and elsewhere: bullets that only restated a plan file, a
  debt already carried by its owning topic, and the 0.2.0 release step (cut;
  tag `v0.2.0` exists).

- **Continue dialog uses the width of the screen.** How and How much sit
  as two columns; the brief preview is wider and taller instead of a
  520 px column in the middle of the pane.

- Web: the dashboard no longer offers a folder-scope filter — the "This machine | PiCode" chips are gone and `GET /api/sessions/stats` measures the whole machine only (ADR-0127). `Spend by workspace` keeps ranking every folder, naming claimed workspaces and leaving the rest as their own folder; the payload's `scope` field and the `?scope=` parameter are removed.

- Web: the inspector panel no longer carries its own hide button — closing it stays with the single toggle at the end of the editor tab strip (browser and desktop).

- **An empty canvas is empty.** The card explaining what a panel is no longer
  floats in the middle of a new canvas — the plane, its texture and the
  toolbar are the whole page.
- **The canvas switcher is as wide as the canvas's name.** It kept a fixed
  minimum width, which left a gap after a short name.

- **Panels are added from the canvas toolbar and nowhere else.** The
  **Add panel…** row left the `⋯` menu, and the empty canvas no longer
  carries its own button — pick an element below, then draw where it goes.

- **Surface split (routes)**: the responsive web app previously served at
  `/desktop/` now lives at `/browser/`, and `/desktop/` serves the new
  bundle composed for the Windows shell only (its entry wraps the app with
  the shell chrome; the browser never loads it). The Management page moved
  into that bundle — its design tokens are imported from the shared
  package, ending the hand-copied token block. The launcher and the docs
  screenshot machinery follow the new routes. ADR-0122.

- **You choose where a panel goes.** **Add panel** has left the toolbar. In
  its place, at the bottom of the canvas, there is one button per thing a
  canvas holds — agent, terminal, pin. Press one, draw the rectangle where
  you want it, and pick which agent or terminal goes there. The panel is
  created exactly in the rectangle, at the size you drew. `Esc`, or pressing
  the button again, cancels.
- **A new panel no longer appears off screen.** It used to be placed from the
  canvas origin outward, which is nowhere near what you are looking at once
  you have panned.
- **The canvas switcher and the `⋯` menu moved to the top-left corner.** The
  bottom edge is now for the hands: what you add in the middle, zoom and
  **Fit** on the left, the minimap on the right.
- **Files and changes are added from `⋯` → Add panel…** — they are opened
  from a tab rather than drawn on the plane.

- **The keyboard map is a pane of its own.** It was a *Keys* sub-tab under the
  Settings pane, which put two tab rows inside one view; it is now **Keyboard**,
  sitting between Settings and Packages in the Agent CLIs pane bar
  (`#/clis/pi/keyboard`). The Settings pane no longer has a sub-tab row, and the
  link keeps the agent and layer you came from, so going back lands where you
  left. A `?tab=keys` bookmark still opens it.

- **The sketch controls sit where a thumb is.** On the phone the tool row now
  sits at the bottom of the drawing with the colour/undo row above it, and the
  library/lock/hand column no longer crowds the pad; the pen is active on open,
  so drawing starts with the first touch.

- **Go tests are cached across worktrees.** `scripts/go-test.sh` builds with
  `-trimpath`, so the test cache no longer depends on the tree's absolute path:
  a package tested once is reused in every worktree and terminal (measured 10.4 s
  per package before, `(cached)` after). Sessions that touch several packages
  get those minutes back on every iteration.
- `make ci` keeps its full output in `var/ci-last.log` and says so when it
  fails. One full-matrix run failed unreproducibly with nothing but a bare
  `FAIL` left in the transcript; retries would hide that, evidence does not.

- The contributor docs point at the generated board: `CONTRIBUTING.md`, the
  `uiux-review`/`visual-review` skills and the affected plans now send a durable
  next-up or debt to `docs/handoff/open/<topic>.md` instead of telling the
  reader to edit `docs/handoff.md`, which `make handoff` renders (ADR-0123).

- Opening a sketch on the phone starts with the pen: draw immediately instead
  of tapping the pencil in the toolbar first.

- **The project board is generated.** `docs/handoff.md` is no longer a file
  every session edits (it was the repository's most-written file: 406 commits
  in ten days, 99.8% of its size cap, net losing text). `make handoff` renders
  it — what is in flight from git, next up and debts from
  `docs/handoff/open/<topic>.md` and the session notes — and it is not
  committed (ADR-0123). Contributors with an old clone see it disappear from
  `git status`; that is the change.
- `make worktree-status` shows what is actually open: each worktree's branch,
  commits ahead/behind `main`, dirty files, last commit and last green gate —
  and flags a tree that has stopped producing commits.
- `make close` reuses the green scoped run when a catch-up merge brought
  unrelated work (ADR-0124), instead of re-testing a tree whose tested content
  did not change; it prints which of the two cases applied and why.
- `docs-site/public/llms.txt` is generated by the docs build and the deploy
  instead of being committed (ADR-0125); `make docs` still writes it.
- `make dev`, `make ci-scoped` and `make close` print the worktree and branch
  they are running in.

- The floating-controls experiment and the frame handshake were replaced
  by the dedicated bar (ADR-0122 supersedes ADR-0121).

- **The canvas controls stand in three places.** Zoom and **Fit** are a
  column at the bottom-left, the canvas switcher and **Add panel** are
  centred, and the minimap keeps the right corner. The zoom percentage is
  gone from the row.
- **The canvas switcher is a PiCode menu, not the browser's.** It used to be
  a native dropdown, which opened an operating-system list over the plane in
  colours nothing else in the app uses.

- **The terminal message field is a textarea that grows.** One line tall at
  rest, four lines maximum, then it scrolls. Enter sends on the desktop and
  Shift+Enter breaks the line; on a phone Enter breaks the line (no Shift on
  a soft keyboard) and Send or Ctrl/⌘+Enter submits.

- **Agent CLIs → Settings edits one layer at a time.** A labelled switcher
  (**Edit: This machine / workspace / agent**) picks the layer, the line under
  it names the file that layer writes, and the body shows only that layer's
  rows — instead of stacking the same six knobs for the machine and the
  workspace and hiding the agent's below them. The chosen layer and the tab
  travel on the URL (`?layer=…&tab=…`), so a reload or a bookmark lands on the
  same view; an agent link opens that agent's layer.
- Rows say where their value comes from: **Set here** (with an accent bar) when
  this layer sets it, **Pi default** or **From This machine** when it inherits.
  A row this layer sets offers **Use inherited**, which hands the value back to
  the layer below instead of freezing a copy of it.
- **Keys** is a sub-tab of its own: the whole 89-row keyboard map keeps its
  filter and its Add-then-press-a-key flow, without sitting at the end of the
  settings scroll. The global settings pane went from ~6000 px of scroll to
  under 1000 px.

- **The Canvas toolbar is one button group.** The canvas and the camera were
  two groups with a gap; they are now a single row — the canvas switcher,
  **Add panel**, the zoom controls, and the `⋯` menu last. **Add panel** is
  no longer filled in the accent colour, and **Fit** wears React Flow's own
  four-corner glyph instead of diagonal arrows.
- **The canvas switcher is always a dropdown, with the app's icon.** With a
  single canvas it had become a plain label; it is a menu again, so the
  canvases you could have are visible from the one you are on.

- `picode-desktop` gained a `clean` subcommand that passes `picode clean`
  through to the distro, streaming its progress.

- **The Canvas chrome is now one toolbar on the bottom edge.** The canvas
  switcher, **Add panel** and the `⋯` menu left the top-left corner and
  joined the zoom controls in a single dock centred at the bottom, as two
  button groups: the canvas and its actions, then the camera. **Add panel**
  is the filled button there. The minimap keeps the bottom-right corner, and
  the top of the plane is now empty — nothing floats where you start
  reading. On a narrow pane the minimap steps aside so the two never
  overlap.

- **Agent CLIs → Connectors is a roster, not a catalog grid.** The pane opens
  with one primary action (**Add connector**), then the configured services:
  each row carries its state, the layer it lives in (title: the config file),
  the target, a switch and an overflow menu (**Sign out**, **Remove**, whose
  confirmation names the file). The catalog, the file picker and the target
  pills left the pane and became one dialog with a search field, a labelled
  **Save to** control, and two quiet secondary entries (*Custom server…*,
  *Import a file…*).
- The redundant workspace line under the Connectors tab bar is gone (it
  repeated the scope already shown per row), and every control in the pane is
  the 36 px control height instead of a 28 px pill beside a 36 px button.
- Live status is only claimed while an agent runs: with the agent stopped the
  pane says so once, instead of showing `Idle` on every row. A server that
  needs a login shows **Sign in** on its row even with the agent stopped.
- The install target is part of the route (`?scope=`), so it survives a
  reload, and it is stated where the write happens instead of above the list.

- **The Canvas camera controls moved to the bottom-left, as a column.** Zoom
  in, zoom out, the 100 % readout and **Fit** now stand at the far end of the
  bottom edge, with the minimap opposite them on the right instead of
  stacked above. **Fit** is an icon there; its keyboard shortcut (`0`) is
  unchanged.

- **A Canvas panel resizes from its bottom edge, its right edge and the
  corner between them.** The other five targets are gone: dragging one of
  them moved the panel's top-left corner while it resized, so the panel
  slid away from under the pointer.

- **Agent CLIs → Packages: the install target sits with the action it
  modifies.** The "This machine / workspace / agent" radios left the row they
  had under the source input — where they read as if they belonged to the
  Installed list below — and now sit in the install bar, labelled **Install
  to**, immediately before Install. Every control in that bar shares the 36 px
  control height, and the bar wraps as a unit (source, then target plus
  Install) when the pane is narrow.
- The redundant workspace line under the Packages tab bar is gone: it repeated
  the workspace the target pill already names, and it sat hard against the tab
  rule. The pane owns a 12 px gap under the tabs instead, on Packages and on
  the pi-roles settings sub-page alike.

- **Fullscreen hides PiCode, and leaves your browser window alone.** Turning
  it on no longer takes the browser fullscreen with it: your other tabs and
  your address bar stay where they are, and the mode is exactly what it says —
  the sidebar, the tab strip and the Inspector step aside, the edges bring
  them back, `Esc` closes an open panel and leaves on the next press. `F11`
  still works and now stacks on top of the mode, in either order.

- **The canvas switcher is a name until there is something to switch to.**
  With one canvas the top-left control opened a list holding the canvas you
  were already on. It is now simply that canvas's name; the moment a second
  canvas exists it becomes a picker again. **New canvas** is in the `⋯` menu
  either way, and the controls do not shift when the second one appears.
- **An empty canvas now offers its next step in the middle of the plane** —
  one line saying what a panel is, and **Add panel** — instead of a blank
  plane with the controls in the corner. The plane, its background and the
  minimap stay where they are, and the line goes the instant the first panel
  lands.
- **The plane no longer shows the React Flow credit.** The drawing library
  behind the canvas is MIT-licensed and unchanged; only its badge is hidden.
- **What's New opens on a fresh install too.** A released build used to wait
  until you had made a workspace, an agent or a terminal, which meant a brand
  new install saw nothing and looked broken. It now opens once on the first
  load, still closes for good when you dismiss it, and still waits while a
  dialog, a reconnect, a waiting agent, an Inbox item or a create/share flow
  needs you first.

- Agent CLIs strip is **CLIs | Messages**. Settings, Packages and Connectors
  (MCP) are panes of the selected CLI, next to Launch / Terminals / Sessions /
  Providers. Webhooks stay a PiCode page. Old Settings, Packages, Integrations
  connectors and `#/mcps` addresses rewrite.
- Providers pane dropped the “This machine” lecture; empty llama.cpp is one
  line plus Set up.

- **A Canvas panel's edges and corners are now something you can actually
  grab.** Every resize target is the same size to the hand however far in
  or out the plane is zoomed — a 20-pixel square at each corner, a
  12-pixel band along each edge — instead of shrinking with the camera
  until there was nothing left to aim at. What you *see* is the grid's old
  language back: a short bar at the middle of each edge and a corner mark,
  appearing when you hover, focus or select the panel.
- **The line between two linked panels is a curve**, and the line you drag
  while making the link is the same curve you end up with. Two panels
  sitting in the same row still join with a straight segment — that is what
  a symmetric curve between two aligned points is.
- **A link is easier to hit when zoomed out.** The invisible band along the
  line no longer thins with the camera.
- **The Canvas background texture can be seen.** Dots, Grid and Cross were
  drawn in the colour of a hairline — on the light theme the grid was at
  1.16:1 against the plane, which is to say invisible. All three now use a
  colour chosen for a texture, and the dot is two pixels rather than one.
  The spacing is unchanged, so nothing on a canvas moves.
- **A stopped agent's panel offers Run as the accented button** the agent
  tab uses for the same thing, instead of a grey chip that read as
  disabled. The same rule now runs through every panel placeholder: the
  action that *starts work* is accented, the action that only opens a tab
  or drops a dead binding is not.

- **The Canvas background is now chosen on the canvas.** `⋯` → **Background**
  opens a submenu with Plain, Dots, Grid and Cross, each showing a real
  sample of its texture and a tick on the one you are using. The rows stay
  open while you pick, so the plane behind the menu is the preview. Your
  choice is the same one you had — it is still remembered per browser, and
  nothing on a canvas moves when the texture changes.
- **Preferences → Appearance is PiCode's own chrome again**: the theme, and
  nothing else. The **Canvas background** group has left it, and so has the
  `⋯` menu's old `Background…` item, which existed only to send you there.
  An app's settings now live in the app — a project rule from today, not
  just a tidy-up (ADR-0109).
- **Messages' list of hand-drawn links reads as Messages'**, because that is
  whose it is: it is now **Granted contacts** — the pairs you allowed to
  message each other by drawing a link between two panels on a canvas. Same
  rows, same Remove, same guarantee that a link never lets one session read
  another's history.

- **A failed compact no longer leaves the machine without its distro.** The
  restart after `wsl --terminate` runs even when the conversion fails, and the
  tray re-arms the keepalive child the terminate killed — without it, WSL
  idles out sixty seconds after a compact.

- **Agent CLIs** opens from the last icon in the desktop sidebar header
  (stacked boxes — the catalog of CLIs), not from the user menu Tools
  list. Clicking it does not switch a rail tab. Package-update marks
  move to that icon. `Ctrl+K` and mobile **More** are unchanged.
- Launch **Customize** sits on the Launch pane heading again, not on the
  Launch / Terminals / Sessions / Providers tab row.

- A CLI's page hosts **Launch**, **Terminals**, **Sessions** and
  **Providers** as inner tabs. The catalog is the CLI picker. Pi shows the
  account roster; other CLIs explain that Providers is unavailable and offer
  Open Pi. `#/clis/<cli>/providers` is the list;
  `#/clis/<cli>/providers/new` opens Add provider. Old `#/clis/providers*`,
  `#/providers*` and `#/more/providers*` rewrite onto those addresses.
  OAuth still returns to the list in the same app.

- A VHDX that is not sparse is named in the report as the reason WSL cannot
  give freed blocks back, with the one command that changes it — it is not run,
  because converting or compacting the file stops the distro and its sessions.

- **Fullscreen: the exit control is a button again, not a sentence.** The
  label and the chord moved into its hint (`Leave fullscreen
  (Ctrl+Shift+Enter)`), and the glyph now sits in the same 28px square as the
  Inspector toggle beside it.

- **What's new is a two-column board on a desktop.** The release dialog takes
  880 px instead of 400 and lays the highlights out in two columns, so a whole
  release fits one screen: the nine highlights of v0.2.0 are read without
  scrolling, and a summary stops wrapping at ~40 characters. The dialog also
  centres itself at any window height — a short window scrolls inside the
  dialog instead of pushing it against the top of the screen.

- A CLI's page hosts **Launch**, **Terminals** and **Sessions** as inner
  tabs. The catalog is the CLI picker; switching CLI keeps the pane.
  `#/clis/<cli>/sessions` lists every folder; `#/clis/<cli>/sessions/<id>`
  lists one. Old `#/clis/sessions*`, `?cli=` and `#/sessions*` links rewrite
  onto those addresses. Dashboard top-session rows open that CLI's pane.

- **The Canvas has no bar across the top any more.** The plane runs from
  edge to edge and the controls float on it: which canvas, **Add panel**
  and a `⋯` menu sit in one small cluster top-left, the zoom controls and
  the minimap stay bottom-right. The bar's icon and title were the tab's
  own words, and its **Close** was the tab's ×.
- **New canvas, Tidy panels, Rename, Delete canvas and Close tab moved
  into that `⋯` menu.** Everything the bar carried is still one press
  away, by mouse or by keyboard — `Tab` reaches both clusters before it
  reaches the panels.
- **Fit now leaves the controls their corners**, so opening a canvas no
  longer parks a panel's header under them.
- **Maximizing a panel hides the floating controls** while it is up: they
  belong to the plane, and the plane is what the panel covers. `Esc`
  brings both back.
- **The Canvas plane's background is yours to pick.** Preferences →
  Appearance now offers **Plain**, **Dots** (what it drew before), **Grid**
  and **Cross**, each card showing the pattern rather than naming it. The
  choice is remembered per browser and applies the moment you make it, to
  every canvas already open. It follows the theme's colours, so switching
  between light and dark re-tints the ground without a reload.
- **Light mode is no longer pure white.** The page was white and the panels
  on it were grey, which is upside down: nothing lifted, and the app read as
  one flat sheet. The page is now a soft, slightly cool grey and the
  surfaces above it are near-white, with dialogs and menus in white above
  those. Text sits at 15.9:1 on the page and 17.3:1 on a panel, and links
  and primary buttons went one step darker to clear the readability bar they
  were under.

- **Providers: one row per account, columns that line up.** The roster is a
  dense table now — provider, account, identity, usage, 7-day spend and the
  actions sit in fixed columns instead of being pushed to the two edges of
  the card. The quota reading moved into its own column, so two accounts'
  bars can be compared without reading a label, and a long vendor reason
  ("Rate limited.") stays on one line instead of wrapping under the buttons.
  Ten providers plus the llama.cpp pointer and Recently used now fit one
  1000 px-tall window without scrolling.
- The search field and Add provider are one cluster instead of two distant
  corners, and a second account of the same provider repeats the provider's
  name, dimmed and indented, so a group reads on its own.
- Recently used is a line of chips (provider, Sign in, remove) instead of a
  full-height list of rows.

- Saving a terminal's launch settings — or opening one that was removed
  meanwhile — returns to that CLI's page instead of the machine-wide list.

- **The Matrix app is now called Canvas.** Same boards, same panels, same
  links; the name says what it is. Old bookmarks keep working —
  `#/app/matrix` and `#/app/matrix/<id>` land on the canvas — and a tab
  you had open reopens as the Canvas tab, with the canvas and the view
  you left it at.
- **Every board is the canvas plane.** The 12-column grid layout is gone,
  so there is one place a panel can be and one way to move it. A board
  still in grid mode was converted once, keeping the arrangement it had —
  a panel that was smaller than the canvas minimum comes out slightly
  larger.
- **The API and the feed were renamed with it**: `/api/matrices…` is now
  `/api/canvases…` and the `matrix.*` events are `canvas.*`. PiCode's own
  clients ship in the same binary and were changed together.
- **PiCode loads lighter for everyone.** Canvas is fetched the first time
  you open it instead of riding in every page load, and the two layout
  libraries the grid needed are gone: 18 KB gzip off the first load of the
  desktop, whether or not you use the app.

### Removed

- **The Go tray is deleted (ADR-0142).** `picode-shell.exe` is the only
  Windows resident — autostart, keepalive, health, disk line, Give-back,
  Restart, Logs — and `picode-desktop.exe` is its headless boundary tool
  (`doctor install disk disk-compact clean startup-check startup-repair
  update`). The systray dependency is gone, so a second tray cannot come
  back. A logon task that still points at the tray fails loud with the
  repair named; `install` registers the shell, staging it from the
  matching release when no sibling is beside the tool.

- The last `surface: terminal` catalog row. Every launchable CLI now
  carries an adapter entry; the terminal-only shape stays as the form a
  future CLI without one rejoins.

- **The user menu no longer carries a layout switch.** The `Layout` group
  (Desktop / Auto / Mobile) is gone from the menu in the browser, the desktop
  shell and the narrow/mobile layout — the surface follows the address you
  open (the launcher, `/browser/`, `/desktop/`, `/mobile/`). Theme
  (Light / System / Dark) stays in the menu.

- **From a Pi session** in the create dialog (desktop and mobile) and its
  API, `POST /api/clis/{cli}/sessions/adopt`. Agents are born only from
  new sessions (ADR-0126, superseding ADR-0021). Agents already created by
  adoption keep working; session files were never modified. The sessions
  surface keeps list, delete and auto-clean.

- **File and change panels can no longer be added.** They had no tool of
  their own and the old **Add panel** dialog was their only way in. Panels
  already on a canvas keep working.

- The sketch pad's canvas menu drops Load, Save to file, Export image and Save
  as image — the drawing is inserted into the composer, not saved from the pad.

- **Fullscreen no longer hands the browser's reserved keys to your agent.**
  Capturing `Ctrl+T`, `Ctrl+W` and the rest needs a page that asked the
  browser for fullscreen itself, which the mode no longer does; those keys
  stay with the browser in every window. Everything else still reaches the
  terminal untouched.

- **Agent CLIs: the general Providers tab is gone.** Provider accounts of a
  CLI are read in that CLI's **Providers** pane, so the tab strip no longer
  carries a second CLI picker.

- **Agent CLIs: the general Sessions tab is gone.** Sessions of a CLI are
  read in that CLI's **Sessions** pane, so the tab strip no longer carries a
  machine-wide list.

- **Agent CLIs: the general Terminals tab is gone.** A terminal is read in
  the Terminals section of the CLI that launches it, so the tab strip no
  longer carries a machine-wide list; an old `#/clis/terminals` link opens
  the CLI catalog.

### Fixed

- A terminal's folder is right from its first instant, for every reader. `#{pane_current_path}` answers with the tmux *server's* directory — the daemon's own working folder — for a pane tmux has not polled yet (`#{pane_pid}` is already set while it does: 33 of 40 creations in a loop), so anything reading a terminal's cwd in that window saw the wrong folder: the creation response, the file reader that resolves a relative path (`GET /api/terminals/{id}/text`, which answered 404 once under a sharded CI run), the Inspector's root. The tmux manager now remembers the folder it created each session in and answers with it for the first seconds, then follows the shell again.

- In the desktop app, Documentation and other external links now open in
  the system browser instead of dead-clicking.

- `scripts/qa-scratch.sh stop` ends the daemon **it** started (the pid the daemon itself wrote to `data/server.json`, with the remembered pid as fallback) and verifies the port afterwards, instead of killing whoever holds the recorded port: a stale port file used to make `stop` kill a neighbouring scratch's daemon and leave its own alive. If a different process holds that port, the stop now leaves it alone and says so; terminals are still removed through the API first, so no tmux session is orphaned.
- `start` refuses a port someone else holds instead of killing its holder, records the daemon's pid, and `status` prints port, pid, liveness and health.
- A tmux socket that cannot be created is an error, not silence: `NewWithSocket` creates the socket's directory, and tmux's `error creating …`/`error connecting to …` text (printed with exit status 0 by `new-session`) is promoted to an error. Before, a session on an unusable socket path "succeeded" with nothing behind it.

- A new terminal's folder is right from the first response. The creation answer read the pane's directory live, and that read races the pane's own process: tmux returns the *server's* directory — the daemon's own working folder — until the shell has spawned (measured: 12 of 30 creations in a loop), so the UI and the Inspector could see the wrong folder for a moment. The creation answer now names the folder the session was created in; every later poll still reads the live path, where a `cd` shows up.

- The app shell now really runs under the Content-Security-Policy: `/browser/`, `/desktop/` and `/mobile/` — the URLs the launcher and the desktop shell actually load — carried no policy at all, because the header matched only `/`, `/index.html` and `*.html`. The script hash is computed from the file each path serves, so each shell's inline theme bootstrap stays allowed.

- The Inspector's **Servers** tab is reachable with nothing selected: the tab row used to render only when an agent, terminal or workspace was the anchor, so the rail's empty state ("Open an agent or terminal to inspect its files.") had no way to the one panel that is about the machine rather than the selection.

- **The download switch (and every browser preference) survives a save
  now.** The page read the daemon's preferences twice — on load and after
  each save — with two copies of the same mapping, and the save path's
  copy had lost `askDownload`: the switch turned on, the answer came
  back, and the row snapped off, never to turn on again. One reader,
  `web/browser/src/lib/browserPrefs.js`, with the regression covered by
  tests.

- **The work browser's settings switches and actions now actually reach
  the app.** The shell's ACL manifest (`build.rs`) never listed the
  commands added after the first slice — `btab_set_prefs`,
  `btab_clear_data`, `btab_open_external` and the whole Downloads set —
  so every invoke was refused with "not allowed by ACL" and the page
  swallowed the error. The commands are in the manifest and the
  capability now, with their generated permission files.
- **The download folder is picked with the app's folder browser** (the
  same one the Add workspace dialog uses) instead of a typed path, and
  the picked place is translated from the daemon's tree to the Windows
  drive the browser writes on (`/mnt/c/...` → `C:\...`). A folder
  outside a Windows drive is refused with a line saying why.

- **One PiCode can no longer remove another PiCode's terminals.** Every
  session PiCode creates now carries `PICODE_INSTANCE`, the data directory of
  the instance that created it, and the tmux inspector reads it: a session
  from a different PiCode on the same machine is labelled *another PiCode
  instance's work* — with the text saying which one — instead of being
  offered as a removable leftover. The removal itself refuses it, so the rule
  holds even if the request does not come from the screen. Sessions created
  before this version are told apart the same way, by the loopback port their
  session environment names.

- The work-browser tab took the whole window down: `WebTabSurface` named the "Show full URL" preference in an effect's dependency array above the `useState` that declares it, so rendering any browser tab threw `ReferenceError: Cannot access … before initialization` and React unmounted the app (blank page, every tab gone). A deploy from main would have carried it; a plain-browser session on main reproduces it in one click on **New browser tab**.

- The work-browser tab crashed the whole shell: an address-bar preference was read by an effect before its own state existed (`ReferenceError: Cannot access … before initialization`, blank window on every work-browser tab). The tab strip also never received the tabs' addresses, so every web tab read "New tab".

- **A long session preview no longer breaks the row.** In the Sessions tabs the
  second line is the CLI's own prompt text — a paragraph, or a quoted Windows
  path. It sized the line to its whole text and pushed **Open in terminal** and
  the row menu outside the card; at the same time the model name (a secondary,
  single-line attribute) refused to shrink, squeezed the session name to
  nothing and painted over the age/size/count. A session row is now two
  deterministic lines: identity + facts, then the preview with its buttons,
  each clamped with `…`. Seen on Muse Code and Grok; every CLI's Sessions tab
  uses the same row.

- Terminal settings: a failed refresh of the guard no longer leaves the old
  state on screen without a word.

- Clearing site data no longer touches saved passwords or autofill; the
  earlier one-click version included general autofill in its mask and
  missed WebView2's own browsing history.

- The server test that launches a terminal for the change feed no longer leaves
  the tmux session behind: cleanup ran with `t.Context()`, which is canceled
  before cleanups start, so every tmux call it made failed and each test run
  left one session in the shared tmux server.

- **`make ci` no longer leaves orphan shells on your tmux server.** The test
  suites that start real tmux servers (server, tmux, term) now run them on a
  private namespace that is torn down with the run — after a 2026-09-15 CI
  run left twelve orphan shells behind. The harness also refuses, by
  construction, to end a server outside its own directory.

- **Terminal settings say what is wrong when there is no tmux server to
  read.** With every terminal closed, the terminal settings catalog now
  answers "Start a terminal to read the tmux option list." instead of an
  internal error — a state an isolated test suite exposed on 2026-09-15.

- A guard-style wrapper finding its real binary no longer depends on
  `dirname(1)`: under a minimal PATH it could resolve to itself and loop.

- **README and architecture docs corrected.** The Agent CLIs list now names
  Muse Code and Antigravity as onboarding terminal-only entries, and the
  architecture pages match the current routes (`/browser/`,
  `/api/canvases`), the Go MCP broker, and the `web/browser` component
  paths.

- **Muse Code and Antigravity terminals wear their own mark in the sidebar,
  the tabs and the phone.** A terminal launched with a CLI that reports no
  activity (both of them today) fell back to the plain-shell icon, the
  *Shell session* subtitle and the *Terminal open* status. Every surface now
  resolves the identity from the runtime CLI, otherwise from the CLI the
  terminal was launched with, so the row reads **Antigravity · Open** or
  **Muse Code · Open** like any other CLI. It also lands in the dashboard's
  CLI bucket instead of counting as a plain shell.
- **A terminal created while the page is open stops looking like a shell.**
  The creation only announced the bare record, so a new Muse Code or
  Antigravity terminal kept the default icon until a reload; the launch view
  now travels on the feed with it.

- **Canvas: the plane fills the pane again, and *No canvas yet* sits in the
  middle of it.** The page-frame change of 2026-09-14 dropped the surface's
  own flex column, so the stage collapsed to its content: an open canvas
  measured 0px tall (an invisible plane) and the empty state hugged the top
  of the tab. A native surface (Canvas, the QA demo) keeps that column now.

- **Create an agent in the grants empty state now opens the New agent
  form.** It used to navigate home and nothing else (the handler called a
  helper that fell through to the default route). The copy also says what
  is true: grants apply to managed agents — sidebar sessions always read
  the tab on screen.

- **Grok 1.0.30 suggestions no longer block message delivery.** The new
  build restyled the unaccepted suggestion in its composer, so automated
  prompts silently stayed pending. Both captured suggestion styles are now
  recognized, with the same strict frame, cursor and footer requirements.

- **Inbox replies reach legacy terminal questions.** An `ask_human` item
  from pi in an Agent CLI terminal that predates per-question session
  stamping can now be answered from the Inbox when the terminal's pinned
  session and the conversation its receiver is showing agree; the item
  still stays open with an honest refusal when they disagree.

- **The agent split's browser pane stayed painted over other tabs**: the
  native WebView2 kept its last bounds when the hosting tab was switched
  away (the pane mounts only for the active tab, and unmount never told the
  shell to hide). Switching tabs — or closing the pane — now parks the
  view; its page state survives and remounting brings it back.

- Resuming a stopped CLI terminal that fails now says why on the pane
  ("Codex was not found. Check its executable or PATH.") instead of nothing
  changing when the button is pressed.

- Sidebar: each tab (Workspaces, Agents, Terminals, Apps, Pins) keeps its header and search bar pinned at the top; scrolling now moves only the list, not the tab's controls.

- **Communication delivery survives slow TUI redraws.** A native CLI
  (Grok and the others) that renders the received prompt late under load no
  longer ends a delivery as uncertain: the post-paste composer check is
  sampled inside a bounded window, and a refusal log names which guard
  refused without exposing screen content.

- Destructive buttons on the phone (Delete) read destructive **before** the
  tap. They only had a hover style, and a phone has no hover.

- Opening a link to an inbox item that is gone (answered or removed elsewhere) says so — "This item is no longer in the list" with **Back to the list** — instead of `not found` with a *Try again* that could never work.
- The tmux app's Sessions tab shows its session count instead of a `21 · 21 unclaimed` sentence, which no tab badge in the product had room for.
- The split view's empty detail pane says "Pick an item from the list" — on a narrow window the list is above, not to the left.

- **Communication: OpenCode now receives messages in narrow windows with deep project folders.** The delivery check no longer mistakes the wrapped folder path in OpenCode's footer for typed text; it accepts exactly the path rows the width implies and still refuses anything else.

- Desktop: clicking a work-browser tab in the tab strip now selects it (it
  silently did nothing — `openTab` had no web branch and fell through to the
  agent lookup), and a selected browser tab no longer snaps back to the
  previous tab on the next route reconcile. Work-browser tabs own a
  `#/web/<id>` address the router writes and honors.

- **The guide was showing `&#123;&#123;name&#125;&#125;` instead of `{{name}}`.**
  Every placeholder on the public Snippets page was written as an HTML entity,
  which a code block renders as literal text. The braces are real now, on that
  page and everywhere the guide shows one.

- **The Add/Edit endpoint buttons stay in reach.** A tall custom endpoint form
  scrolled its **Back** / **Add endpoint** buttons off the bottom of the dialog,
  so saving meant scrolling to find them. The action row is now pinned to the
  dialog's (or the phone sheet's) bottom edge and the form scrolls under it.
- **Verify says what it covers instead of a status it never had.** An endpoint
  whose API type PiCode cannot speak (a hand-edited `models.json`) answered
  "rejected the request (0)"; it now names the type and the four API shapes
  verification can address, and reports that nothing was spent.

- **A wrong key on a custom endpoint no longer reads as verified.** Verify used
  to answer from credential presence, which stayed green for a bogus key; a
  gateway that cannot list, a refused key, an exhausted account and an unknown
  model each now say which one happened.

- **Quota readings no longer outlive their window.** `Limits` now marks a
  reading whose window has already reset as `stale · reset 2d ago` instead of
  counting down to a past instant, and every row carries how old the reading
  is (hover). A panel fed by one CLI also says so: `Covers Codex only.`

- **A gateway's error message can no longer leak your key.** If an endpoint
  echoes the API key back in an error, PiCode redacts it before the message
  reaches the browser.

- **The `Today` range drew one bar stretched across the whole chart.** A
  one-day window was one calendar day, so the panel showed a single
  full-width block under a range picker that promised a chart. `Today` is now
  bucketed by hour (24 bars, labelled `midnight` … `11pm`, panel titled
  `Hourly`), and every longer range keeps one bar per day.

- The `events` verb now honours `since`: the sequence number the agent last saw
  travelled at the top level of the request, where the channel never read it,
  so every poll replayed the whole ring.

- **A hand-edited `compat` key no longer hides the provider from the GUI.** A
  string value in `compat` (such as `"thinkingFormat": "deepseek"`) made the
  whole entry fail to decode, so the provider silently vanished from the
  roster, Edit and the catalog while pi kept using it. The file is now read
  key by key and unknown keys are left untouched.

- Right-clicking a selection **inside a text field** now offers Copy and
  **Save selection as snippet** enabled: the menu reads the field's own
  selection, which the page's selection never saw.

- `make desktop-shell` actually builds the Windows shell again — the target
  was shadowed by the `desktop-shell/` directory and reported "up to date".

- Agent CLIs: the page no longer sticks on its loading skeletons when the terminal list is slow — `GET /api/terminals` now serves a shared snapshot (singleflight + 1s TTL) computed by a bounded worker pool, and the page keeps the last result that landed instead of discarding every response older than the newest request.
- Agent CLIs: web-tab ids (`w:<n>`) no longer reach `/api/agents/{id}/slash` and `/role-state` as agent ids (console 404s).

- **A file, folder, git or app tab no longer asks the API for an agent's role.**
  Selecting one of those tabs fetched `/role-state` and `/slash` with a tab id
  the server does not know as an agent — two 404s per selection, a chat
  composer asking a repository its role. Both fetches now ask `isAgentTab`
  (a bare id, not a tagged surface), so they run where an agent is on screen
  and nowhere else; the composer's role chip and slash list are unchanged on
  an agent tab.

- **Removing a workspace no longer takes the server down.** The git watcher
  killed the whole daemon when it noticed a watched folder was gone: that path
  had left the watch set, and the pass still read the group it no longer had.
  A removal — a workspace, its last agent, or a folder that stopped being a
  repository — is now a quiet pass; the removal event is what reconciles the
  sidebar, as it always did.

- **Private connection setup no longer lingers on disk.** PiCode removes a conversation's private setup files once its connection is revoked or the owner is deleted; the credential stopped working before, but the 0600 files used to stay behind.

- **Continue: a session too large for Native can still use Brief.** The
  dialog no longer leaves Continue disabled; Native is offered only when
  the conversation fits, and Brief uses the recent turns.

- **Grok terminals show Needs you while a question card waits.** The
  `ask_user_question` card notified an `elicitation_dialog` notification that
  the hook map ignored, so the row stayed on Working for the whole wait. The
  question is now reported as a held attention, and it survives the sibling
  tool completions Grok runs in the same parallel batch — the hold ends only
  when the question tool itself completes.

- **Terminal side padding is even.** xterm's fit floors the column count, so
  unused pixels collected on the right of every pane (desktop app, browser,
  and phone). The leftover is now split so both sides match the terminal
  padding preference.

- **Right-clicking a canvas opens the canvas's own menu again.** PiCode's
  generic menu (Copy, Paste, Reload PiCode…) was answering instead, which
  also meant that once you hid the canvas controls there was no way to bring
  them back. Right-clicking inside a terminal panel still opens that
  terminal's menu.

- Keep a newly opened Claude Code terminal running when communication is enabled before its conversation is saved; connect after its first message creates a resumable conversation.
- Keep terminal Pi message notifications pending while its editor contains an unsent draft, allowing delivery once the draft is cleared.
- Explicitly ask native agents to execute the communication diagnostic after reading it.

- Clicking the keyboard map no longer silently returns to Settings: the pane
  rewrote the route to carry the selected agent and dropped the sub-tab with
  it (the new pane cannot lose a tab it does not have, and the layer survives
  the same rewrite).

- Deliver first-message notifications to an idle Grok welcome screen while preserving draft, identity and exact-submit safeguards.

- Recognize an empty OpenCode editor with its sidebar visible and working directory wrapped below the footer. Communication now measures pointer fit against the editor width while retaining draft and post-paste edit protection.

- **The sketch pad on the phone fits the screen.** The drawing surface now
  fills the usable area — no header under the status bar, no dead strip under
  the drawing — and its Cancel/Insert buttons are thumb-sized (44px). It lives
  inside the shell's viewport, so the software keyboard shrinks it correctly.
- **The sketch canvas is dark in dark mode.** The pad asked Excalidraw for a
  near-black canvas, which its dark-theme filter inverted back to light; the
  canvas is plain white paper now, so the screen is dark and the PNG attached
  to the terminal stays a white sheet.

- `make worktree-status` no longer reports a finished branch as a stalled one:
  a worktree whose branch `main` already contains reads *merged — `make
  worktree-gc` can remove it*, and the handoff board's **In flight** section
  lists only trees with unfinished work (a dirty tree is never called merged).

- A settings value can now be *un-set*: `PUT /api/pi-settings` accepts
  `patch.reset[]` and removes exactly those keys from the layer's file
  (other keys, including ones PiCode does not know, stay). An unknown name is
  refused rather than reported as inherited. After a reset, a running agent
  adopts the effective compaction/steering/follow-up values instead of keeping
  the override that just left the file.
- A malformed `~/.pi/agent/settings.json` no longer disables the agent's own
  Model/Tools/Checklist fields on the mobile quick sheet.

- web: a TUI booting inside a terminal (OpenCode boots straight into it) could
  freeze the pane at its first painted line until a reload. esbuild's minifier
  dropped the `let` declaration of xterm 6's `requestMode` enum while keeping
  the `(n = {})` assignment, so the first DECRQM query threw
  `ReferenceError: n is not defined` inside the parser and stalled xterm's
  write pipeline. The desktop and mobile builds now pin `@xterm/xterm` to its
  UMD build, which ships the enum pre-compiled.

- Recover observed native conversations and activity after a PiCode server restart on Linux/WSL, preserving running terminals, drafts and pending messages (ADR-0112).
- Show activity, connection status and completed communication tests separately on desktop and mobile; keep selected participants visible while reconnecting.
- Renew Pi receiver presence without restarting its terminal, and bind Hermes message commands to the native conversation when a background review changes its environment.
- Deliver pending messages when Grok's empty composer shows an unaccepted native suggestion, preserving typed drafts and native approval guards.
- Keep live native hooks working on hosts without Linux process metadata; distinguish temporary recorder writes from permanent recording failures.
- Show a failed connection with a Terminal controls action when state recording is blocked, and record bounded delivery failure reasons without logging message content.
- Bind native message commands to the launched CLI even when PiCode inherits another agent's environment; refuse a missing native identity instead of selecting the parent conversation.

- Mobile rendered the Connectors pane without its stylesheet (the CSS was only
  imported by the webhooks view, which mobile loads lazily). The pane imports
  its own styles now.
- Removing the unreachable second "MCPs" view (`#/mcps` already rewrote to the
  Connectors pane) leaves one rendering path for both apps.

- **Zooming out no longer turns a terminal panel white.** Below 75 % a panel
  shows its last screen as text instead of a live terminal, and that text
  was landing on the panel's own light ground — the same words, suddenly
  looking like something other than a terminal. The still now keeps the
  terminal's background and foreground, following your terminal theme.

- **Deploying from a PiCode terminal no longer refuses because of that
  terminal.** The interlock that protects other people's turns counted the
  pane running `picode deploy` — which is working, by the act of asking —
  so a deploy started from inside PiCode could be told to wait for itself.
  It now ignores the caller's own pane; `--force` still means ending
  someone else's turn.

- Opening **Packages**, **Connectors** or **Settings** from Agent CLIs while a
  workspace agent is selected (sidebar icon, then the pane tab) puts that
  agent and folder on the URL, so **This workspace** / **This agent** appear
  next to **This machine**.

- **Canvas links now attach to the sides of the panels that face each
  other.** A link used to leave from one point at the top of each card
  wherever the two cards sat, so it could run out of a corner, cut across
  the panel it belonged to, or — when a panel carried several links — leave
  every one of them from the same spot. A link to a panel on the right now
  joins right edge to left edge, one below joins bottom to top, and a
  diagonal neighbour is met on the side it actually lies beyond. The line
  leaves and enters at a right angle to the border it touches, and follows
  the panels as you drag them.
- **Several links out of one panel fan out instead of stacking.** Each one
  leaves from the point on that side which faces its own target, and two
  that would land on top of each other are pushed apart just far enough to
  read as two lines — in the order of their targets, so they do not cross.
- **The line you drag is the line you get.** While you drag a new link, the
  preview leaves the same border the finished link will, and jumps to the
  other panel's facing border as soon as you are over a panel the link can
  land on.

- Connectors opened from the composer, user menu, palette or More keep the
  selected agent and workspace in the URL, so “This agent” and reload after
  Add still work. Free agents resolve the same way as Packages.
- `#/packages` and `#/settings` without a query adopt the selected pane
  again once the fleet is ready. `#/clis/connectors` rewrites to Pi.
  Extra path on Settings is blocked instead of opening the editor.

- **The terminal shows one scrollbar, never two.** The web terminal is a tmux
  client and tmux attaches on the alternate screen, so xterm has no scrollback
  of its own — and yet it painted two one-pixel artefacts at the right edge of
  every terminal: its own scrollbar (one pixel wide, because
  `overviewRuler.width` is also its `verticalScrollbarSize`) and the overview
  ruler's white outline. Both are hidden now, on desktop and mobile, together
  with the eight-pixel scrollbar gutter the viewport reserved for a bar nobody
  could see (Chromium only removed it on touch platforms; the terminal surface
  painted over it). The fit keeps reserving the one pixel, so the text grid is
  unchanged, and the wheel still scrolls what it always scrolled — tmux's
  history or the TUI's own viewport. The scrollbar a reader sees is the one
  that belongs to whoever holds the scrollback.

- **The Inspector's branch chip is readable again.** A branch name and a
  worktree name were sharing one chip's width, each with its own ellipsis,
  and together they read as `· fe… ·…`. The row shows the branch, which is
  what it is for; the worktree is in the tooltip with the rest, and on a
  narrow rail the *unpublished* / *detached* word steps aside for the name
  instead of the other way round.

- **Fullscreen: the Inspector button in the top strip shows the panel.** It
  only flipped the rail's dock preference while fullscreen keeps the rail off
  screen by itself, so the click changed an icon and nothing on the page.
  Inside the mode it now lays the rail over the page at once — opening the
  rail when it was closed — and hiding it there takes only the overlay down,
  never the dock you set outside the mode. A panel opened that way stays until
  the same button, Escape or the mode: it no longer slides shut behind a
  pointer that never entered it. The right edge still works, and the rail can
  be turned on from the strip with the mode already running.

- **Approving a permission prompt no longer leaves the row saying "Needs
  you" while the agent works.** Grok, Claude Code and Codex report a waiting
  prompt as `needs-you`, but none of them emits a "permission resolved"
  event, and none of them registered a tool-activity hook to contradict it —
  so the row stayed `Needs you` for the rest of the turn. A Grok plan-mode
  approval showed it for 10+ minutes after the plan had been approved. The
  tool lifecycle is now the resume signal (`PostToolUse`, plus
  `PostToolUseFailure` where the CLI has it), so the first tool that runs
  after the answer returns the terminal to `Working`.
- A Grok notification that is not a permission prompt (`task_complete`, …)
  no longer turns the row into a false `Needs you`; only a permission UI
  that is actually waiting does.

- The release dialog was 400 px wide, not the 560 px it asked for: `.dlg` is
  declared later in the stylesheet at the same specificity, so its `width` —
  and its `padding`, which kept the header and footer hairlines inset from the
  dialog's edge — won silently.
- The matrix highlight of v0.2.0 wore a generic sparkle: the note's icon name
  had no glyph in either shell. Desktop draws it with the Canvas glyph, the
  phone with the panel grid, and `web/tools/release-note-icons.test.mjs` now
  fails when a published icon name is missing.
- A build with no release notes showed the changelog link twice — once as the
  empty state's action and again in the footer.

- Column headings are only drawn when there are rows, so an empty search no
  longer leaves a header over nothing, and the empty roster shows one action
  instead of two identical Add provider buttons.
- With no providers connected the page no longer renders an empty toolbar
  row.

- **Dashboard: Grok's tokens, cost, turns, tool calls and timings now show
  up.** The Grok coverage row said "prompt history only" and rendered `—`
  beside a CLI that has been writing a `summary.json`, an `events.jsonl`
  turn/tool timeline and a `usage.json` per-turn count into every session
  directory. Turns, tools and durations come from `events.jsonl`; the model
  from `summary.json`; tokens and cost from `usage.json`, which Grok only
  began writing in 1.0.x. Because older sessions have no `usage.json`,
  tokens and cost report as *partial* with both counts — the coverage row
  says exactly how many turns are priced — instead of a total that quietly
  omits them.

- **Inbox: ignoring a question from a pi in an Agent CLI terminal now
  closes it.** Ignore sends nothing, so it no longer needs that terminal to
  be live and on the same session — it never claims "no reply" and then
  refuses to be dismissed.
- **Inbox: a reply to a terminal question fails with the truth.** A pi in
  that terminal that had not opened a conversation (a nested `pi -p`, a
  print-mode run) could take the reply file and answer "the terminal is
  showing a different session". The daemon now refuses before parking when
  the receiver names no session, each reply file is addressed to the
  process whose hello was accepted, and a sessionless hello can no longer
  take that address over — so another pi in the same terminal can neither
  consume nor blind the reply.

- **Codex terminals no longer try to resume a sub-agent conversation.** Codex multi-agent v2 sub-agents (helper threads like "Gibbs") could overwrite the conversation a terminal remembers, and Codex refuses to resume them (`Process exited (1)`). Sub-agent threads are now ignored for resume, in live hook reports and in the session list — resume always brings back the conversation you were in.

- An old `#/clis/terminals?…` link now rewrites the address to `#/clis`
  like the plain address always did, instead of keeping the stale URL while
  showing the catalog.

### Security

- `frame-src` (with the dev-server preview) allows PiCode's own browser surface to frame this machine's `localhost`/`127.0.0.1` pages over http(s) and nothing else; the framed page remains a separate origin with its own policy, and `frame-ancestors 'self'` keeps PiCode itself unframed by others.

- Read is the default and the ceiling: an agent with no grant reads the tab the
  human has on screen, and cannot click, type or navigate. Anything more needs
  an explicit per-agent grant (Settings ▸ Browser, next).

- **The desktop debug port is off by default.** It was opened on loopback in
  every launch; it is now an explicit `PICODE_CDP_PORT` opt-in for external
  tooling. The agent access tiers (`read` / `act` / `full`) are enforced
  against a named command catalog, and an unnamed command is refused at every
  tier.

## [0.2.0] - 2026-09-11

### Added

- **Matrix: a managed agent's panel is now its live conversation.** Where a managed agent's panel said *Managed agent — open to read.*, it shows what the agent is saying as it says it — the same turns, tool cards, diffs and markdown its tab shows, scrolling itself, in a read-only layout with no composer and no queue controls. When the agent is waiting on a person the chip says **Needs you**, the body shows the question and the choices it offers, and a line across the bottom of the panel carries it with **Open**, which takes you to the tab where you answer. The chip comes from the fleet, not from the panel's connection, so a panel that is asleep, paused or zoomed down to a name-plate still tells you which agent is blocked.

- **Link two Matrix panels, and the two sessions can message each other
  (ADR-0116).** On a matrix canvas, hover a panel and drag the connector in
  its header onto another panel. The line grants exactly one thing: those
  two sessions gain each other as a contact in PiCode's existing messaging
  (ADR-0104). One can send the other a message and read the replies — that
  is all it does.

- **A link never grants a transcript.** It does not let one session read the
  other's history, scrollback, session file or anything typed into it. The
  supported way to get context out of another session is to ask it and let
  it answer in its own words. The guide says so in plain words.

- **Two sessions in different project folders can now be paired**, per pair,
  by hand. The workspace rule is unchanged and there is no owner-wide
  switch: the blast radius of a link is two named sessions you drew a line
  between.

- **Nothing is connected quietly.** Drawing a line to a session that is not
  connected offers the existing connection in one line, with what it grants;
  drawing across two project folders asks again, separately, naming both
  folders. Cancel at either point writes nothing at all — no link, no
  connection.

- **Removing the line removes the permission**, with nothing left over:
  PiCode derives who may message whom from the links that exist right now,
  so deleting the line, either panel, or the matrix revokes it immediately.

- **A link that grants nothing reads broken**, in amber and marked
  *Broken*, with the reason when you point at it — the connection was
  revoked, the session moved, the target is gone, or it was never connected.
  It is never a faded version of a working link.

- **Matrix links, in the Messages view.** Every link you have drawn,
  anywhere: both ends, the matrix it lives on, whether it works right now,
  and a Remove that revokes it exactly as the canvas does. It is not
  filtered by the folder picker, because links across folders are the ones
  worth seeing. Grid mode, which has no plane to draw on, shows a per-panel
  link count in the header and sends you here.

- **Matrix: a panel can be a pinned note.** Add one from **Add panel** →
  **Pins**: the panel shows that pin's markdown, read-only, follows it as
  you edit it elsewhere, and its header opens Pin Studio. A note whose pin
  was deleted says so and offers Remove, the way a deleted terminal does.

- **Matrix: a panel can be a file.** **Add panel** → **Open files** offers
  the files you already have open in a tab; the panel is the same editor,
  with Save. It says *Unsaved* while you have changes, keeps them when you
  maximize it or switch layout, and never goes to sleep holding them.

- **Matrix: a panel can be the changes to a file.** The same list offers
  each open file as **Changes to an open file**: the panel shows that
  path's diff and keeps it current as the file changes, with **Open file**
  to jump to the editor. One file can be on a matrix twice, as the file and
  as its diff.

- **The Matrix canvas — your panels on a plane you pan and zoom.** A matrix header now has a **Grid | Canvas** switch. On a canvas a panel goes anywhere: drag it by its header, resize it from any edge or corner, drag a box around several and move them together, pan with the middle button or by holding Space, zoom with the wheel or the `+` / `−` buttons, and `0` fits everything on screen. A minimap in the corner shows the whole plane. The arrow keys still move between panels — an off-screen one is brought into view — Enter still starts typing in that panel's terminal, and Delete still removes it with an Undo. Switching to the canvas converts the whole matrix in one step and switching back packs it into the 12 columns again (it asks first, since the plane positions are not kept). Where you left the camera is remembered per browser, so two people looking at the same matrix never yank each other's view.

- **Zoomed out, a panel shows its last screen instead of a live one.** Below about 75 % a panel stops being a connected terminal and shows the text of the screen it last had — sharp, free, and stamped with its age in the header if the panel has done something since; below about 40 % it becomes a name-plate with the face, name and status, sized so it stays readable however far out you are. Zoom back in and the same terminals reconnect where they left off. That is what lets a canvas hold hundreds of panels without hundreds of connections.

- **A terminal takes the mouse at 100 % only, and says so.** A terminal panel is readable and typeable from 80 % up, but a *click* is only accurate at 100 %: below it the terminal would put your click on the wrong cell, and with mouse reporting on that wrong cell reaches the program running inside. So away from 100 % the panel body does not take clicks — clicking it zooms the canvas back to 100 % first, and then your click lands where you aimed it. The zoom percentage sits next to the minimap and takes you back to 100 % in one press.

- **Tidy.** The matrix menu gains **Tidy panels** on a canvas: it lays them out again in reading order, keeping each panel's size, and saves once.

- **Mobile: Back from an agent or terminal lands where the work lives.** A workspace's agent or terminal returns to Workspaces focused on that workspace's group; a free one returns to its own Agents or Terminals list — never a flat section it was never in.

- **A matrix can be a canvas, from the API.** Every matrix now has a layout mode: the 12-column grid it has always had, or a `canvas` where panels sit on a plane with no columns and no bottom — coordinates in 8 px units that may run negative. `PATCH /api/matrices/{id}` takes `mode` and answers with the matrix plus every panel it moved: switching converts the whole matrix in one step (a grid cell becomes 8 units across and 3 down; coming back divides, rounds and tidies the panels so none overlap and none is lost), and switching to the mode it already has changes nothing. Matrices made before this keep the grid. **There is no canvas to look at yet** — the panels, the mode switch and the pan-and-zoom surface are the next step; this release is the model underneath, exercised by tests.

- **Fullscreen hands the browser's keys to your agent.** In a normal window the browser keeps its reserved shortcuts (`Ctrl+T` new tab, `Ctrl+W` close tab, `Ctrl+N` new window) before PiCode can see them — so a CLI chord like Codex's `Ctrl+T` opened a browser tab instead. Fullscreen mode (`Ctrl+Shift+Enter`) now also locks the keyboard (Chrome/Edge/Opera): every key reaches the page, terminals get their chords back, `Escape` still leaves the mode outside a terminal, and holding `Escape` for about two seconds is always an exit. Firefox and Safari keep the previous behavior until they support the API.

- **Fullscreen: any tab can take the whole window.** Right-click anywhere in the desktop app — a terminal, an app, a file, the conversation — and choose **Fullscreen** (`Ctrl+Shift+Enter`, `Cmd+Shift+Enter`, or the command palette). The sidebar, the tab strip and the Inspector rail step out of the way and the tab you are on fills the screen. Nothing is closed: touch the **left** edge with the pointer and the sidebar slides back over the page, the **top** edge brings the tabs back — with a **Leave fullscreen** button at their right end — and the **right** edge brings the Inspector back if it was open when you started. Each edge waits a moment before opening, so crossing it on the way somewhere else does not flash it, and it stays while the pointer is on it. Esc leaves. The mode is remembered per browser, so a reload comes back into it, and it is offered on every tab except the git graph's own commit menu.

- **The browser goes fullscreen too.** Turning it on also asks the browser for real fullscreen, so the app is the whole screen and not just the whole page; leaving gives the window back, and if you exit fullscreen yourself (F11, Esc) the mode ends with it. A browser that refuses fullscreen still hides the app's own chrome. A reload cannot ask for fullscreen — there is no click to ask with — so it returns to the in-app mode, with the top edge still there to leave from.

- **Matrix — a live grid of your agents and terminals.** The Apps tab has a Matrix tile: create a matrix, add panels from a picker of your agents and terminals, drag them by the header and resize them by the corner or an edge. Each panel shows the real terminal or agent screen — the same one its tab shows — with the sidebar's Working / Needs you / Ready word in its header. Only the panels near what you are looking at stay connected, so a matrix can hold hundreds; layouts save as you go and every browser sees the same matrices (`#/app/matrix/<id>`). The guide is [Matrix](https://cfpperche.github.io/picode/guide/matrix).

- **Keyboard across a matrix.** Click a panel or tab into one, then move with the arrow keys (the target scrolls into view and wakes up), Home and End for the first and last, Enter to start typing in that panel's terminal, Shift+Esc to come back out to the panel, Delete to remove it with an Undo. A plain Esc still belongs to the program in the terminal.

- **Maximize a panel.** The ⤢ button gives one panel the whole surface — the terminal resizes to the space, the matrix keeps its layout underneath, and the panel's slot says *Shown maximized*. Esc or the button puts it back.

- Workspace Communication on desktop and mobile: select managed Pi agents and native terminals, apply their connections, and follow preparation and activity in one view.

- Run a connection test through the participants' native tools. A test passes only after a message, correlated reply and both acknowledgments; pending and unconfirmed results remain visible.

- Agent CLIs → Messages: native Grok and Hermes communication through `picode messages contacts|send|read|ack`, sharing the existing MCP mailbox, authorization and history.

- Stored messages can notify an already open conversation through Pi's native receiver or a guarded TUI pointer. History distinguishes pending, notified, unconfirmed and acknowledged messages.

- **Matrix API and feed events (ADR-0108).** Matrices — named 12-column grids of agent and terminal panels — are stored, shared across browsers and announced on the change feed: `GET`/`POST /api/matrices`, `GET`/`PATCH`/`DELETE /api/matrices/{id}`, `PATCH …/layout` (the changed subset, 409 when stale), `POST …/panels`, `DELETE …/panels/{panelId}`, and six `matrix.*` events. Limits refuse with the limit named (64 matrices, 500 panels each, 80-character names, panels of at least 4×8 cells). No surface yet — the Matrix app itself is phase 3.

- **`picode version` (`--version`, `-v`).** Prints the build identity and exits. The install verifier already ran `picode --version`; until now the flag silently started a server instead.

- Messages can save private setup for recorded Pi, Claude Code, Codex and OpenCode conversations and attach it when they resume. Desktop and mobile show the next action, including installing the Pi adapter when needed.

- Automatic local HTTPS setup supplies verified public CA material to the launched client while preserving existing configured CA bundles.

- Direct session messages (ADR-0104): an embedded HTTP MCP endpoint with same-workspace opt-in, scoped credentials, durable receipts, retry deduplication and explicit acknowledgements. Agent CLIs → Messages on desktop/mobile manages connections and history. Client setup is manual per conversation; no automatic agent turn or terminal migration.

- **Automations: several schedules per automation.** The editor's
  Schedule block is now a list: each rule has its own preset (Hourly,
  Daily, Weekdays, Weekly, Custom), an optional label, its own switch and
  Remove, plus *Add a schedule* — "weekdays at 09:00 and Saturday at noon"
  no longer needs two automations. Each rule is its own clock (own jitter,
  own catch-up, saved in the browser's time zone); the list and the detail
  summarise every rule; the Runs table names the rule that fired. The API
  returns `schedules` on every automation (each with `nextFireAt`) and
  takes `schedules` on create and PATCH; `cron` stays as the one-rule
  shorthand. Editing a rule's time starts it over instead of catching up
  a slot it was never asked for. Migration 038 moves existing schedules
  into rows with their last fire intact. (ADR-0045 amendment 2026-09-09.)

- **Pins on the phone: create and edit.** More → Pins lists your pins
  (starred first, search over title, tags and note) with a **+** that
  opens a form — title, tags, the note as markdown — and the pin screen
  gains Edit. Drafts are retained and a stale save answers with the
  conflict message, as on the desk. Files, sketches and reminders stay
  with the desktop studio.

- **Git graph: per-commit +/- in the listing.** Every commit row shows
  the lines its own diff adds and removes (`+178 −80`, green/red, blank
  when a commit changes no text — empty or binary-only commits), so the
  shape of history reads without opening each commit. The numbers are
  git's own shortstat totals over the commit's first-parent diff — the
  same diff the opened commit detail shows per file (they reconcile
  exactly, merges included).

- **Pins: search, keep on top, archive.** The Pins tab has a search box
  over title, tags and note (every word must match; archived pins are
  found too and say so). A star keeps a pin on top of the list; Archive
  takes it out of the sidebar into an "N archived" view one click away,
  pauses its reminder and closes an open reminder item, and Unarchive
  brings it back (a slot missed meanwhile fires once). The studio has
  Archive and Keep on top beside Delete; the phone's pin screen shows
  "on top" / "archived". `GET /api/pins?q=` and `?archived=1`,
  `POST /api/pins/{id}/starred` and `/archived` (migration 037).

- **Pins: reminders on screen (ADR-0100, slice 3).** The studio has a
  "Remind me" chip: presets first (in 1 h, in 3 h, tomorrow morning, next
  Monday morning, every day, every weekday — day presets at a morning hour
  you can change), then every N hours with "count from when I close it",
  a date-and-time picker and a cron field; the chip and the sidebar card
  read the rule back in words with its next fire ("every day at 09:00 ·
  next tomorrow 09:00"). When a reminder fires, a card with the pin's
  glyph stays on screen until you act — on the desk and on the phone: X
  closes it (the Inbox item goes done everywhere), Snooze hides it for an
  hour, Open lands on the pin. A reminder owed is shown on load too; more
  than three open reminders collapse into one "N reminders" card that
  opens the Inbox. Preferences → Notifications gains "When a pin reminder
  is due" for the card, and the push switches gain the same for the
  phone. The Inbox app's reminder rows say "Pin" and their "Open pin" goes
  to the pin. On the phone `#/pins/<id>` opens a read-only pin (title,
  tags, reminder line, pictures, note) instead of landing on Settings.

- **Pins: reminders, server side (ADR-0100).** `PUT /api/pins/{id}/reminder`
  sets one cadence per pin — `once` at a date and time, `interval` every
  N minutes (at least 5) counted from the schedule or from the moment you
  close the reminder, or `cron` (five fields) evaluated in the IANA zone
  the browser sends — and `DELETE` removes it; the rule rides
  `GET /api/pins/{id}` and the list summary with its next fire and a
  plain-language label ("every day at 09:00", "every 3 h after you close
  it"). A one-minute engine (`internal/remind`, same shape as
  Automations) files each fire as an Inbox item of the new kind
  `reminder` (source `pin`): the item is the acknowledgement — unread
  means owed, snoozed means snoozed, done means closed — and the feed
  announces `pin.reminded`. A fire while the previous item is still open
  re-raises that item instead of piling up; a snoozed item stays quiet and
  is re-raised once the snooze ends; a slot missed while PiCode was down
  fires once with "Was due …" in the body; deleting the pin or the rule
  closes its open item. Phones with push get a `reminder:<item>` push
  under a new `reminders` preference (on by default), kept on screen where
  the platform honours `requireInteraction`. The Inbox app shows the kind
  with an "Open pin" action. `GET /api/inbox?kind=` filters. The in-app
  sticky card and the picker are the next slice.

- **Packages: view, edit and reset package configuration (pi-roles first,
  ADR-0099).** Installed packages render as the familiar card grid, now filterable,
  with **Configure** beside Update/Remove on packages with a known config
  adapter. The pi-roles editor (`#/packages/config/pi-roles`)
  edits the same files the extension reads — the workspace
  `.pi/roles.json` and the per-agent overlay `.pi/roles/<agentId>.json` —
  and shows the effective merge: inherited slots are visible (and can be
  overridden for one agent), overrides can fall back to the workspace
  value with one click, and each layer can be cleared with a confirmed
  "Clear file…". A file the parser rejects is reported, never silently
  overwritten (explicit replace). Saves write atomically, preserve
  unknown keys, announce on the change feed, and say when they apply
  ("on the agent's next message"). The page follows the house control
  recipe throughout — selects and inputs share the system's height, radius
  and focus language in light and dark — and a thinking level cannot be set
  without a model (it would have been silently dropped on save).

- **Mobile: Preferences → Layout — fit the shell to your screen.** Two
  dials with platform-smart defaults: **Bottom bar buttons** (Auto / Low /
  Screen edge) and **Bottom bar height** (48/56/64px). "Auto" anchors the
  buttons low only when the bootstrap measured a letterboxed standalone
  install (WebKit 313800/317153 — the behavior varies by iOS generation and
  icon install date); "Screen edge" is the former strip-probe experiment as
  a user choice, honest about the clipping risk in its own label. Applied
  before first paint, persisted per device like the theme. Side fix: the
  Preferences tabs kept their choice in the desktop hash space
  (`#/preferences/<tab>`), which the mobile router can't express — every
  tab click silently fell back to Appearance; tabs are now component state.

- **Public docs: working inside a terminal.** The Agent CLIs guide gains
  *Inside the terminal pane* — the right-click menu, the `Shift` bypass to
  the browser's own menu, the message bar that opens from it, find with its
  three toggles, and a table of every terminal key. The Attach paragraph no
  longer describes a bar that stands under the pane on desktop; it opens
  from the menu now.
  [Agent CLIs](https://cfpperche.github.io/picode/guide/agent-clis).

- **The git graph's actions on the phone.** A long press on a history row —
  or **Actions** on a commit or worktree view — opens a sheet with the same
  vocabulary the desktop graph offers, in the same tiers, through the same
  doors: prepared in a terminal, run when nobody is working there, or asked
  of an agent or pi terminal living in that worktree. The exact command is
  composed by the server and shown before anything is sent; a tier C action
  through the run door asks you to type a word first.

- **The worktree removal form names its field.** "Name" over a worktree folder
  read as nothing; it says *Worktree folder name* now, like the create form.

- **A clean sibling worktree can be removed from its branch pill.** A checkout
  with nothing uncommitted draws no row of its own, so the graph had no place
  to offer *Remove this worktree* for it; the branch pill of a branch checked
  out elsewhere now carries the worktree's rows.

- **"Ask Pi" reaches pi running as a terminal.** A pi launched from Agent
  CLIs now carries the same receiver an interactive agent does, so the git
  graph lists it beside the agents of its worktree, offers **Open Pi** into
  its pane, and can ask it to do a git action in its own turn — delivered
  through the receiver with the session row as proof, never pasted. Without
  a live receiver or a named session the option is absent and the request
  says why. Other CLIs (Claude Code, Codex…) are unchanged: no receiver, no
  ask.

- **The git graph acts on what you point at.** Thirty-odd git actions bound
  to the row or pill under the cursor — create a branch, tag or worktree,
  check out, merge, rebase, cherry-pick, revert, reset, pull, push, delete —
  each delivered through a door that already existed: prepared in your
  terminal for you to press Enter, run there when nobody else is working in
  the repository, or asked of an agent that lives in that folder. PiCode
  still never runs git itself. Actions carry a risk tier: the ones that
  publish or destroy work are marked, and when PiCode is the one pressing
  Enter they ask you to type a word first. The exact command is composed on
  the server and shown before anything is sent, so the preview and the
  command are the same string.

- **A worktree and the agent that lives in it, in one gesture.** "Create a
  worktree…" on any commit or branch can also start an agent in it once the
  folder exists.

- **Undo, where there is an honest one.** After an action that only moved the
  branch — merge, rebase, pull, reset, commit — the graph offers to put it
  back where it was, and says plainly that it prepares the command rather
  than undoing anything by itself. A push or a `git clean` gets no such
  offer, because there is no inverse to give.

- **The git graph answers a right-click.** Every row and pill now carries a
  menu naming what you pointed at: a commit offers its hash and subject, a
  branch its name, a worktree row its path and the agents living there. A
  local branch says which checkout holds it — and when that is a sibling
  worktree, the menu offers **Open <agent>** instead of a checkout git would
  refuse. Branch pills show how far they have drifted from their upstream
  (`↑182`). The pills on a row are also reachable from the row's own menu, so
  a keyboard reaches everything a right-click does. This is the read-only
  first phase of ADR-0096; nothing here runs git yet.

- **Find inside a terminal** (`Ctrl+Shift+F`, or **Find…** in the pane's
  right-click menu): a field floats over the terminal — searching never
  resizes the pane — highlights every match, counts them (`3/14`), and walks
  them with Enter and Shift+Enter; Escape closes it and gives the keyboard
  back to the terminal. **Match case**, **whole word** and **regular
  expression** are three toggles in the field, kept for the life of the page;
  a pattern that does not compile yet says *Invalid pattern* rather than
  pretending there is nothing to find. The plain `Ctrl+F` still belongs to whatever runs in the
  pane (`less`, `vim`, readline), and the chord is rebindable in
  Settings → Shortcuts.

- **A waiting agent says so on the screen you are on, whatever agent it
  is:** when any managed agent stops for a question, a card names it —
  `Atlas · needs you · QA`, the question, its detail and an **Answer**
  button that opens the conversation. Unlike the finished card this one
  covers the whole fleet, not just the agent you have open: the runtime
  already publishes every dialog edge on the change feed. The card has no
  timer — it goes away when the question is answered, or when you open the
  conversation that answers it — and a reload never replays a backlog the
  sidebar badge already shows. On the phone the Now screen stays quiet,
  because that screen *is* the queue.

- **Notification preferences say what to announce, not where the box
  sits:** *When an agent needs me* and *When a run finishes*, worded like
  the push switches above them. Turning one off silences that
  announcement only — the sidebar badge, the Inbox and the phone are
  untouched. "Close position" and "Rich colors" are gone: the card draws
  its own close control, and a level already reads from its glyph and
  border. Position, duration, visible-at-once, expand and close button are
  unchanged, and an old stored value is simply ignored.

- **Right-click inside a terminal now opens PiCode's own menu**, built from
  what that pane can actually do: copy, paste and select all; on a running
  Agent CLI, **Ask <CLI> about this** and **Attach files…**; **Open** for the
  file or link under the cursor; go to the end and text size; **Clear** on a
  bare shell only, because a TUI owns its screen; rename, terminal settings,
  the folder in Files, close the tab and remove the terminal — the last group
  never on an agent's own TUI pane. Holding the bypass modifier (Shift by
  default, `Settings → Context menu`) still hands the click to the browser's
  own menu. The right button no longer reaches tmux, which used to answer it
  with a second menu drawn inside the terminal.

- **The terminal's message bar is no longer permanent chrome.** It opens from
  that menu — empty, or already carrying the selection: one line lands in the
  message, anything longer is attached as `selection.txt` so the CLI reads a
  whole file instead of a mangled paste. A close button (or Escape) gives the
  pane the full height of the editor back.

- **Toasts are notice cards, and a finished agent turn is one of them:**
  an announcement now carries who spoke, how long they worked, what
  changed and one way out — `claude · finished · worked for 7s`, the
  agent's own last sentence, `1 file +46 −1`, and an **Open** pill —
  instead of a bare string. When a turn settles while you are looking at
  another tab, another view, or another application, that card is what
  tells you; it stays quiet when the agent's own conversation is the
  focused surface, and a second finish from the same agent replaces its
  card instead of stacking a new one. Errors now live 12–30 s instead of
  4, because an error nobody was looking at was a lost error. The 317
  existing one-line toasts keep their wording and gain a level glyph.
  Position, duration, visible-at-once, expand, close button, close
  position and rich colours all keep working. Adapted from the Superset
  study in `docs/benchmarks/2026-09-07-superset-notifications.md`.

- **Public docs: agent terminals over SSH:** a new guide page shows how to
  reach PiCode's tmux sessions over SSH (`tmux attach`, read-only
  supervision with `-r`, Tailscale SSH on a server box) and the one rule:
  never kill sessions by name pattern.
  [Agent terminals over SSH](https://cfpperche.github.io/picode/guide/ssh-terminals).

- **Public docs for moving a conversation between agents:** the Agent CLIs
  guide now explains the two ways a session arrives (a native session, or
  the vendor's own `import` for the SQLite-backed CLIs), the choices in the
  dialog, and what stays behind; the home page names the capability.

- **A workspace reaches its own files and history with nobody in it:**
  **Files** and **Git graph** on the workspace card's menu open the project's
  own file tree and commit graph — no agent and no terminal required. The
  graph, the commit details and the uncommitted changes read through
  `/api/workspaces/{id}/…`, which the server has answered since ADR-0030;
  a workspace-owned graph tab now also survives a reload and a deep link
  (`#/git/w/<id>`) instead of reporting "That agent is gone".

- **Handoff dialog polish**, from the visual pass that was owed: a session
  with no title of its own is no longer named after the handoff note, the
  dialog names it by a short id instead of a full UUID, and the summary
  line counts a CLI's internal record types instead of spelling them out
  ("bridge session left out (4)" meant nothing to a reader).

- **Repository picker in the clone form:** the New-workspace "Clone
  repository" URL field is now a filterable combobox fed by the GitHub
  repositories behind the machine's own `gh` login (`gh repo list` via the
  new `GET /api/github/repos`, cached 5 min). Typing filters `owner/repo`
  and descriptions, grouped by owner, with a lock on private repositories;
  picking one fills URL, name and destination at once. Pasting any URL
  still works first-class — a pasted URL shows a "Use this URL" row and
  Enter submits it as before. Missing `gh` / not-logged-in machines show
  one line with the one action that fixes it (install guide / copy the
  `gh auth login` command); mobile gets the same list as a tap-to-open
  picker next to the URL field.

- **Mobile Files and Git:** full-screen file browsing, search, editing and previews
  with unsaved-change and file-conflict recovery; Git changes, history, branches,
  worktree/commit details, PR status and existing terminal/agent action channels.
  Workspace history now works without an agent; supplied folder preconditions
  are checked for history and commits across workspaces, agents and terminals.

- **Mobile tools:** Agent CLIs now includes Sessions, and More includes
  Automations with list, editor, run history and recoverable loading errors.

- **Every Agent CLI can now receive a session handoff** (ADR-0094). OpenCode
  joins as a source and a target, Grok gains a native session (a live probe
  showed `summary.json` and `chat_history.jsonl` are all `--resume` needs),
  and Hermes becomes a target through `hermes sessions import`. For the two
  SQLite-backed CLIs PiCode hands the conversation to the vendor's own
  import command instead of writing their store, then reads the session
  back. A written session records the model of the installation that will
  answer next, never the source's, which Grok refused and Claude Code could
  not restore; the source model rides in the handoff note instead.

- **llama.cpp installation cleanup:** desktop and mobile can review and remove
  verified older installations while retaining the current version, restoration
  version and unknown files.

- **Install a missing CLI** (ADR-0093): the Agent CLIs surface offers an
  **Install** action when a catalogued CLI is not installed — npm-backed for
  pi, Codex and Claude Code through the same durable job lane; Grok and
  Hermes Agent show their vendor's install guide instead. Installing an
  installed CLI is refused.

- **Managed local llama.cpp service**: desktop/mobile Local service page with
  CPU presets, saved settings, verified installation, reviewed start/stop/
  restart/update/rollback, durable progress, ownership-aware cache cleanup and
  redacted diagnostics. External routers are never adopted.

- **Attach a photo or file to an Agent CLI terminal** (ADR-0089). On a
  running Claude Code, Codex, Grok, Hermes, OpenCode or Pi CLI terminal, an attach
  bar (desktop) or header paperclip (phone) stages the file in the
  terminal folder and types the path into the TUI. Caps: four files, 4 MB
  each. Inspector Git type/run still does not type into a CLI.

- **CLI lifecycle management** (ADR-0087): update checks, update, reinstall
  and uninstall for catalogued agent CLIs. The Agent CLIs surface shows an
  update badge with the latest version (npm registry or the vendor's own
  `--check` command) and runs each CLI's native update/reinstall/uninstall
  command as a durable job with streamed output, terminal guards and typed
  uninstall confirmation. Install methods PiCode cannot manage (Homebrew,
  manual checkouts; Grok and native Claude Code uninstalls) show their
  official guide instead of controls. Jobs survive daemon restarts as
  `interrupted` — never replayed.

- **OpenCode in Agent CLIs**: the catalog launches the installed `opencode`
  command in a terminal, lists top-level sessions from
  `~/.local/share/opencode/opencode.db` (or `$XDG_DATA_HOME/opencode/opencode.db`)
  read-only, and resumes with `opencode --session <id>`. Activity reporting
  is a presence lease plus a session-only plugin injected through
  `OPENCODE_CONFIG` (session busy/idle, permission and question prompts) —
  not a data-dir overlay and not by writing `~/.config/opencode`.
  Maintenance subcommands skip the plugin.
  Update/reinstall/uninstall use `opencode upgrade` and
  `opencode uninstall --keep-config --keep-data --force` (the vendor
  commands, including bun-global installs classified as npm by path).
  OpenCode lists sessions for the Sessions tab, and gives and receives a
  handoff (see below).

- **Continue a session in another Agent CLI** (ADR-0088): the Sessions tab
  offers "Continue in <CLI>…" for every CLI the server says can receive the
  session. A preview shows what travels and what is left behind; the
  handoff writes a new native session for Claude Code or Codex, adopts a
  new Pi agent, or starts a CLI from a deterministic brief. Grok sessions
  now list their real transcripts (title, model, size) and Hermes
  sessions can be read. Lineage shows on both rows.

- **Hermes Agent in Agent CLIs**: the catalog launches the installed `hermes`
  command in a terminal, lists cli/tui sessions from `~/.hermes/state.db`
  read-only, and resumes with `hermes --resume <id>`. Activity reporting is
  a presence lease plus session-only activity hooks (LLM start/end and
  approval prompts) injected through `PYTHONPATH` sitecustomize — not a
  `HERMES_HOME` overlay and not by writing `config.yaml`. Hermes may still
  record PiCode's hook command in its own `shell-hooks-allowlist.json` when
  auto-accepting. Check setup keeps the first `--version` line so Hermes'
  install dump does not wrap the heading.

- **Deploy refuses while agents work** (ADR-0086). `picode deploy` asks the
  running daemon who is mid-turn and stops before touching the installed
  binary when anyone is — the refusal names each agent or terminal and why.
  `picode deploy --force` (or `PICODE_DEPLOY_FORCE=1 make deploy`) is the
  deliberate override. `make deploy-batch` and the `picode-deploy.timer`
  (12:00, 18:00, 23:00; `make timers` installs it) ship `main` in batches,
  refreshing stale public screenshots first. New loopback-only route
  `GET /api/deploy/readiness`.

- **Inspector: ask a running agent to do a Git action** (ADR-0078). For every
  agent running in the rail's repository — managed or in its own terminal —
  the Git menu now lists an "Ask &lt;name&gt;" submenu with the same actions.
  Asking sends the agent a plain-language message through the channel that
  already carries its prompts: a queued turn for a managed agent (delivered
  at once, or as a follow-up once its current turn ends) or the Inbox
  reply's own door into a TUI (the receiver extension, or a bracketed
  paste). The agent decides how and runs it in its own turn; PiCode never
  runs git here. The commit form's message becomes optional when asking —
  left empty, the agent writes one from the changes. A stopped agent, or a
  terminal hosting a coding CLI, is not offered.

- **Inspector: run Git actions when nobody is working** (ADR-0078). The Git
  menu gains a per-viewer checkbox, "Run when no agent is working here". With
  it on, PiCode types the command into your terminal and presses Enter itself,
  but only when no agent in that repository is mid-turn, no automation is
  running there, and no other terminal there is working or holding a program;
  otherwise the command is prepared as before and a note says who is busy.
  Git still runs in your own shell with your credentials and hooks. Typing
  now also refuses a terminal that moved away from the folder or whose pane
  is not at a shell prompt, and takes a fresh terminal instead. Right-hand
  toasts step left of the rail while it is open, so a note never covers its
  buttons.

- **Session forensics** (ADR-0085): the daemon now records which tmux
  sessions were alive at a graceful shutdown and reports at boot exactly
  which ones did not survive (log + `var/restart-report-*.json`), stopped
  CLI terminals say "PiCode restarted while this terminal was running"
  when that applies, launch scripts ignore SIGHUP like interactive shells
  always did (explicit Stop escalates to SIGTERM so stopping still
  stops), and every `picode deploy` appends who/what/where to
  `var/deploy-log.jsonl`. The next session-loss incident self-reports.

- **CLI terminal session recovery** (ADR-0084): every CLI terminal now pins
  the native conversation it is running (claude, codex, grok, pi) and a
  stopped terminal offers "Resume last session" — one click relaunches the
  CLI with its verified resume arguments. A deploy, crash or daemon restart
  no longer costs the conversation; the pin is stored (`terminal.last_session`
  event) so it survives the process. Plain start is unchanged; nothing
  auto-restarts. Desktop and mobile terminal surfaces both carry the button.

- **Contextual llama.cpp model guidance**: GGUF choices show file size,
  estimated runtime memory and a hardware-oriented starting recommendation.

- **Checklist disclosure on sidebar cards** (`docs/plans/sidebar-checklist-
  expand.md`). Clicking a card's plan line expands the full list in place:
  ☑ on finished steps, the braille spinner on the one being executed, ☐ on
  the rest. The items are already in client state and arrive over the feed,
  so opening costs no fetch and updates render live while open. Works on
  agent and terminal cards; the line is a real button (`aria-expanded`,
  Enter opens, Escape closes) and never navigates. No server change.

- **llama.cpp operation activity** (ADR-0083): load, unload and download now
  run as persistent jobs. Activity shows per-file progress, reconnect outcomes
  and supported download cancellation on desktop/mobile. Duplicate requests
  are deduplicated; conflicting model operations are refused. Unknown results
  remain visible rather than being reported as success.

### Changed

- **Matrix: what a conversation panel costs, and what bounds it.** A conversation holds a WebSocket and a transcript, so it is live only where it can be read: in the band and at 40 % zoom or more. Below 40 % a panel is a name-plate and its connection closes; outside the band it unmounts after the same five seconds every other body gets, and reconnects with its conversation when it comes back. At most twelve conversations are live at once — the twelve you scrolled to most recently — and the rest say *Paused* until a slot frees. Measured with twenty managed agents on one matrix: nine connections at rest, twelve at the cap with six paused mid-scroll, never thirteen.

- Participant selection carries forward to future conversations in the selected workspace. Each conversation keeps a separate credential; clearing a selection revokes its connections atomically.

- Normal setup offers Apply and connect, Open and connect and repair actions, with per-conversation configuration under Advanced.

- Agent CLI terminal menus are one menu everywhere: the sidebar terminal rows and the Agent CLIs terminal list render the same rows — Rename…, Launch settings, Terminal settings, Start/Restart/Stop terminal (per running state) and Remove terminal. The sidebar gains Start/Restart/Stop and Launch settings; the Agent CLIs list gains Rename… and Terminal settings.

- Grok/Hermes integration uses native hooks/plugins with ownership receipts and preserves their native executable, home and unrelated configuration.

- **Apps host: a first-party app may declare a native surface (ADR-0109).** A manifest can say `surface: "native"`: the app's body is then a component compiled into the desktop shell instead of a primitives view, opened from the same tile, tab and `#/app/<id>` route. The phone lists such an app as *Desktop only* and answers its link with one line and Back; a desktop build that lacks the surface dims the tile as needing a newer PiCode. No shipped app uses it yet — the only native app is the hidden QA demo behind `PICODE_DEMO_APP=1`, which shows a live terminal; a terminal shown there and in its own tab follows whichever is visible.

- Workspace cards: the plan (checklist) line no longer shows a chevron — the whole line stays clickable to expand the plan, and the line is now italic, matching the plan's voice.

- Document verified native conversation resumes and acknowledged message roundtrips for Claude Code/Codex and OpenCode/Claude Code, including OpenCode `zai/glm-5.3-flash` with variant `max`; runtime behavior is unchanged.

- **Development process (ADR-0105).** Deploy happens only when the owner runs
  `make deploy`; the deploy timer and `make deploy-batch` are gone. The
  changelog is assembled from `docs/changelog.d/` fragments, the handoff
  keeps no shipped-work prose, `docs/architecture.md` is an index over
  `docs/architecture/`, and `make adr` seeds a decision record with a
  boundary line. Gates: `internal/server` tests run sharded across four
  processes, `make ci` runs its gates in parallel, `make web` is a no-op when
  nothing under `web/` changed, and `make close` reuses a green
  `ci-scoped` for an unchanged tree.

- **Sidebar: selection without the blue bar.** A selected agent or
  terminal card no longer paints the accent bar that read as a blue
  border around the card — selection is the tinted background plus the
  quiet border (the bar stays on *Needs you*, where it marks attention).
  The card's checklist chevron joins the card grid: the `>` now sits on
  the same column as the folder/branch icons below it (supersedes the
  2026-09-06 own-gutter refinement), its text on the folder/branch label
  column, and expanded plan steps follow the same two columns.

- **User menu: llama.cpp in Tools.** The desktop user menu's Tools group
  now lists **llama.cpp** (models, server connection and local service),
  matching the mobile More screen; it is searchable like every row.
  Previously the manager was reachable only through Providers or by URL.

- **Sidebar: PiCode without the build number.** The version string
  (`v0.1.0+…`) no longer sits next to the name in the sidebar header;
  it still lives in the user menu.

- **User menu: one Tools group.** After Providers, Settings and Packages
  moved under Agent CLIs, the leftover *Agents and connections* heading
  (Integrations alone) is gone. The desktop user menu and the phone's More
  screen list Integrations with Agent CLIs and Automations under Tools.

- **Desktop: one panel width for every page.** All routes now share the
  Agent CLIs panel geometry — fluid up to 1240px, centred, same gutters.
  System, Integrations, MCPs, Devices, Preferences and Pins no longer use
  narrower centred cards (680px), and Automations, llama.cpp and Terminal
  defaults stop at 1240px instead of 1080px. Workspace surfaces (agents,
  terminals, files, git, apps) are unchanged.

- **Providers under Agent CLIs.** Pi accounts, keys, quotas and verification
  now live in Agent CLIs → Providers, with a CLI selector and explicit machine
  scope. Old links and login shortcuts redirect; OAuth returns to the same
  desktop/mobile app. Failed catalog refreshes keep the current roster.

- **Packages under Agent CLIs.** Pi package management and desktop package
  configuration now have explicit CLI and workspace/agent URLs. Old links
  redirect with their context, update indicators lead to Agent CLIs, and
  failed context reads preserve drafts while blocking stale writes.

- **Pins: the reminder picker is a form, not a menu of presets.** Once —
  a date and a time; Repeat — every N hours, or every N days at a time of
  day (one day at a time is a wall-clock rule; several days count as a
  duration and say so), optionally counted from when you close the card.
  One Set, Remove beside it; opening starts from the rule that is set. The
  "morning hour" preference went with the presets. An interval may now
  name its first fire (`at` on `PUT /api/pins/{id}/reminder`), and its
  label says the time of day ("every 2 days at 18:30").

- **Mobile Work: workspace favicons match agent and terminal marks.** The group head uses the same 24px slot and 22px glyph as the rows under it, so a project icon is no longer smaller than the CLI faces in the same list.

- **Mobile Work empty states sit in the well.** Agents, terminals and workspaces with nothing to list center one line and a primary create button in the remaining space (muted icon, no essay); a search miss stays at the top with Clear search. Copy drops "free".

- **Sending a message to the terminal no longer toasts "Sent to the
  terminal."** The user is looking at the pane and sees the message land —
  the confirmation was chrome talking to itself. The message bar/sheet
  closes over the visible delivery instead; toasts stay reserved for
  failures (send errors still surface via `toastError`). Both surfaces
  changed together: desktop bar (`TermAttachBar.jsx`) and mobile sheet
  (`TermAttachSheet.jsx`).

- **Settings now lives under Agent CLIs → Settings → Pi (ADR-0101).** Desktop and mobile retain global, trusted workspace, agent and key settings. Old links redirect; contextual URLs preserve the agent on reload. Native settings load independently of terminal setup, and failed saves retain edits.

- **Agent CLIs: one CLI combobox with favicons.** Settings, Sessions and the new-terminal form share a `--ctl-h` combobox that shows each CLI's mark instead of an unstyled native select.

- **Development: the public docs site moved from `www/` to `docs-site/`** —
  same VitePress build and GitHub Pages URL (`cfpperche.github.io/picode/`);
  the `www/` name is reserved for the future product website. Make targets
  (`DOCS_STAMP`), CI workflows, scripts and living docs updated in the same
  commit; ADR-0086 amended in place (owner-approved).

- **The dashboard counts every agent CLI, not just Pi.** `GET
  /api/sessions/stats` now aggregates all six agent CLIs — Pi, Claude Code,
  Codex, OpenCode, Hermes Agent and Grok — so spend, activity, tokens, tools and turns
  describe the machine rather than one CLI. On the machine this was built
  on, the same 7-day window went from **$517.04** to **$1,916.35** — the old
  number was 27% of the truth, with nothing on screen to say so. Two new
  breakdowns ride along: `byCli` (which CLI the money went to) and
  `coverage` (what each CLI can and cannot report), plus code impact,
  timings and quota windows where a CLI records them. A metric a CLI never
  writes is reported as *not reported*, never as `0` — Codex is never given
  an invented price, and shows the quota window it does record instead
  (81% of a weekly limit, resetting Saturday, on the machine this was built
  on). Parsing is cached per file, so the minute-by-minute refresh costs
  46 ms rather than the 2.4 s a full re-read would. A new
  `?scope=machine|picode` narrows the window to folders a PiCode workspace
  claims; the default counts everything, as before. (ADR-0097)

- **The dashboard shows which CLI the money went to.** A **By CLI** card
  ranks spend per agent CLI with a billing badge (`api` / `sub`), and four
  new panels read what the guest CLIs record and Pi never did: **Code
  impact** (lines added/removed, cost per line), **Agent time** (waiting on
  models vs. running tools — agent time, not elapsed, since sessions run at
  once), **Limits** (a quota window's headroom and reset), and
  **Efficiency** (cache hit, cost per turn). A **What each CLI reports**
  matrix shows the blind spots as data. A **This machine / PiCode** control
  narrows the window to claimed workspaces.

- **The dashboard never prints `$0.00` for spend it could not measure.** A
  day whose sessions are all still running has no price on disk yet; it now
  reads `—` with "not priced by any CLI that ran" instead of a zero that
  looked like "free". Sub-cent spend reads `<$0.01`, a plan-covered CLI
  reads "not priced", and a period with no activity says so instead of
  ranking six CLIs at zero.

- **Spend now lands on the day it was earned.** A compaction marker is
  counted on its own timestamp instead of the session file's modification
  time, so a long-running session no longer dumps its whole compaction
  history onto the day it was last touched.

- **The desktop user menu follows the mobile v2 pattern:** rows are grouped
  (Tools / Agents and connections / PiCode / Continue) and each carries a
  one-line description; a search field filters the menu with the same
  matcher as the phone's More screen; a no-match search says so with a
  Clear search action. Theme and layout radios stay in the menu; the
  install button and version hide while searching.
  Cites `docs/benchmarks/2026-09-07-mobile-v2.md`.

- **The workspace card's actions are two, not five:** a **New** menu (agent,
  shell terminal, Agent CLI terminal) and one overflow menu holding Files,
  Git graph, Sessions and Remove workspace. Both triggers stay visible at
  rest — the old strip appeared only on hover, out of reach of touch — and
  the destructive action no longer sits beside the creative one. The menu
  hides Git graph on a folder that is not a repository and Sessions while
  the workspace has no agents, where the route answers 409. The empty state
  gained its two actions: "Empty — *add an agent* or *a terminal*". The card
  header stays one line: the rows under it already carry path and branch, so
  repeating them on the workspace was noise (owner's call, 2026-09-07).
  See `docs/benchmarks/2026-09-07-workspace-card-toolbar.md`.

- **Mobile v2:** compact conversation and terminal controls, searchable Work
  views and grouped tools give more space to active work. Each agent keeps
  its unsent draft and attachments while navigating; message options open
  in a sheet and Enter adds a new line. Failed sends offer Retry.

- **Pi terminals no longer show a checklist strip above the pane.** The
  TUI already draws the plan; the sidebar card still carries the one-line
  step. The pane is just the terminal.

- **Composer attaches a photo from this device.** An image button next to
  the paperclip opens Photos, the camera, or a file picker. The paperclip
  still attaches a file from the agent's folder on the PiCode machine.
  Same caps as paste/drop: four images, 4 MB each.

- **Mobile extra-keys row no longer shows a vertical overlay scrollbar.**
  Dragging the row sideways is `pan-x` only; the terminal screen, the
  row and xterm hide overlay scrollbars so iOS cannot paint a gray strip
  down the right edge.

- **Mobile extra keys no longer leave a black strip, and hide the TUI
  behind them.** The phone shell fills the screen while the keyboard is
  closed. Opening it shrinks the terminal (and refits xterm) so the
  prompt stays above the extra-keys row; that row is opaque. Safari's
  undo/Done pill above the keyboard is the system IME — a web app cannot
  remove it.

- **Mobile extra keys sit above the phone keyboard.** The terminal key
  bar is one horizontally scrolling row (esc, tab, ctrl, alt, arrows,
  Ctrl+C, then Home/End/pages and `| ~ / -`) that opens and closes with
  the software keyboard instead of a two-row Termux grid the IME could
  cover. The phone shell sizes itself to the visual viewport so the chat
  composer stays visible while typing. Sticky Ctrl/Alt are unchanged.
  The same row is available on an agent's Terminal view.

- **Loopback browser sessions end when the access ends** (ADR-0049
  amendment 2026-09-06). An auto-minted loopback browser session — the
  silent mint every browser on this machine gets, the headless QA fleet
  included — is revoked by a minute housekeeping sweep once its last
  authenticated request is 10 minutes old: a closed browser stops
  refreshing `last_seen_at`, an open one keeps the row alive even with
  Chrome's once-a-minute background-timer throttle. Each revocation is a
  `session.revoked` event, so open Devices views drop the row live; the
  daily prune deletes the rows a week later. Paired devices (phones,
  paired loopbacks) and the install token are never touched. The pile of
  offline "Headless browser" rows self-clears on the first sweep after
  upgrade.

- **Development process** (ADR-0086, owner-approved after the 2026-09-06
  cost review): `make ci-scoped` runs only the gates a branch's diff can
  break; `make close` ends a worktree session (scoped gates, regenerated
  OpenAPI/llms/captures, fast-forward check, closing summary); `make
  worktree NAME=x` hardlinks `node_modules` instead of `npm ci`; `make
  worktree-gc` removes merged, clean, idle trees; `make docs-check` warns
  on stale capture fingerprints instead of failing (`--strict` for the
  old gate); `docs/handoff.md` is capped at 100 lines by the pre-commit
  hook and per-session notes move to `docs/handoff/`; `docs/screenshots/`
  is frozen (evidence stays in `var/screenshots/`).

- **Sidebar text column corrected for the smaller identity mark.** The
  runtime-favicon resize shrank agent/terminal identity marks from 24px to
  16px without updating the folder/branch line's (and, transitively, the
  checklist line's) left inset, so every row's `.ws-context` sub-line sat
  8px right of its own title in production. Found while re-verifying the
  checklist chevron fix below against the deployed favicon change; the
  inset is now 23px (16px mark + 7px row gap) everywhere it is used.

- **Checklist chevron moved to its own gutter** (owner refinement, follow-up
  to the compact-view alignment below). The disclosure's chevron sat in the
  same 31px column as the title/folder text, but a rendered icon's ink
  rarely touches its own bounding box the way plain text does — the owner
  spotted the chevron reading as offset from the title above it even though
  the boxes lined up. The chevron now sits in a dedicated gutter *before*
  that column (the tree-view convention: a leading disclosure mark, then
  the label), so the **text** of the checklist line and its expanded steps
  lands on the exact same column as the title, subtitle and folder/branch
  line, independent of the chevron icon's own inset. No JSX change.

- **Checklist compact-view alignment** (owner refinement, `docs/plans/
  sidebar-checklist-expand.md`). The counter drops its parentheses (`5/8`,
  not `(5/8)`) and moves to a fixed column at the row's end, the Linear/
  GitHub sub-issue idiom, instead of prefixing the step text. The line and
  its expanded list now sit in the same 31px text column as the card's
  folder/branch line below it — previously flush left, out of the card's
  grid. The disclosure line's focus ring matches the row's own selection
  style (`box-shadow`) instead of a text-field-like inset outline. A
  finished plan (every step completed) dims to the same weight as a
  completed step, so it stops reading as open activity. The plan line
  moves above the folder/branch line on agent and terminal cards, matching
  identity → activity → location. Mobile's sub-line drops its parens too
  (`5/8 · text`). No server or domain change.

- **Identity favicons match the workspace favicon**. Agent faces (sidebar
  rows, collapsed strips, editor tabs, inspector) and terminal CLI badges
  now render at the workspace favicon's 16px box with its 3px radius —
  natural image shape, no forced circular crop. Letter/glyph fallbacks keep
  the white plate for sidebar contrast; image faces stay full-bleed. Row
  indent (`ws-meta`) re-aligned to the smaller identity mark.

- **Checklist line refinements on cards** (ADR-0092). The `(x/n)` counter
  no longer wears the accent color — the operator line is one muted line.
  An **absent** checklist (the task required a plan and none was written)
  now renders as silence: no line, no "No checklist", on agent cards,
  terminal cards, the terminal pane strip and mobile rows. The data plane
  is unchanged — the absent marker is still published and stored.

### Removed

- Mobile: the "Running" section is gone from the Now home; agents and terminals running stay visible with their state chips in the Work tab.

### Fixed

- **Reloading no longer halves fullscreen mode.** A reload always brought the mode's layout back but not the browser part — no gesture, so no fullscreen and no keyboard lock — and you had to toggle it off and on to get the keys back. Now the first click or keypress after the reload completes it on its own: the browser goes fullscreen again, the terminal keeps every key, nothing else changes. If the first gesture is Escape or the fullscreen chord, it does what it means (leaves the mode); synthetic input never triggers it.

- Mobile: the Work tab no longer scrolls horizontally — the per-workspace action strip (Communication, Files, Git, +Agent, +Terminal) grew wider than a phone screen and pushed the whole page sideways; the actions now live in an "…" menu on each workspace row.

- Prevent OpenCode from hanging when communication reconnects its native conversation; validate resumed readiness without overriding newer activity or permission requests.

- Connect identified Codex conversations without restarting, and preserve their native identity when older launchers emit auxiliary completion notifications.

- Preserve terminal dimensions during communication setup and recognize Claude Code's compact footer without requiring the browser panel to resize.

- Communication now recognizes Grok 1.0.25's bordered input when delivering message and connection-test prompts. Drafts, unfamiliar layouts and prompts that would wrap remain protected.

- A panel whose terminal or agent is deleted lets go of its screen at once instead of holding it until the cache trims, and keeps the name it was showing, so the row reads *shell · Gone — That terminal is gone.*

- Pi connection setup shares its receiver registration, preserves native input and rejects stale process or conversation identity.

- Communication preparation resumes the exact idle terminal conversation only after verifying the previous writer has exited; an unverified shutdown blocks replacement.

- Managed Pi reconnect checks pending delivery, native commands and approvals before stopping. A stale stop cannot remove its replacement.

- Sidebar: the plan (checklist) line's text starts on the card's left edge — same column as the title, subtitle and folder/branch icons — instead of floating ~16px right on a ghost indent.

- Native conversation identity no longer falls back to the most recent session for enrolled terminals; daemon restart recovery verifies the current pane/process.

- Prevent duplicate conversation addresses across `/new` and resume, stale activity reports, multiline draft submission, and attention starvation behind another recipient's backlog.

- An explicit successful Messages refresh clears stale action errors while preserving history.

- Codex lifecycle hooks remain active on resume/fork by placing their scoped overrides in the native subcommand.

- Desktop: the default right-click menu (Copy, Paste, Reload PiCode, Toggle theme) again renders rows like the terminal context menu — icon and label together on the left, instead of the label pushed to the menu's right edge.

- Creating two terminals at once on a machine with no tmux server running
  no longer fails one of them with "server exited unexpectedly": the daemon
  retries the tmux startup race for `has-session` and `new-session`.

- **The HTTP docs pair shells the way the server actually pairs.** `docs/api` now shows `picode pair` (the old example called an endpoint that never existed) and no longer claims `PICODE_INSECURE=1` skips pairing — that is the **Who must pair: Off** setting (`PICODE_AUTH_MODE=off`).

- **The architecture index renders as one table again** on GitHub and the docs site (a stray blank line had broken it at ADR-0090).

- **Stale file path in the routes doc** (`web/src/lib/fileDocument.js` → `web/desktop/…`, mirrored in `web/mobile`).

- **Development process (ADR-0105 follow-up).** The pre-commit hook lets the
  release cut through (a `CHANGELOG.md` edit that adds a `## [x.y.z]`
  heading) and parses staged changelog fragments so a malformed one fails at
  commit time, not at release. `make deploy` commits only the refreshed
  captures, never whatever else was staged. `make adr` allocates the number
  across every worktree even when run from inside one. The split
  architecture files link to `docs/plans`, `docs/design` and
  `docs/benchmarks` again. Capture tolerance is an absolute 128 px budget
  with the count printed per surface.

- Pi terminal resumes use the recorded conversation file, including older session pins without resume arguments.

- Communication setup preserves unrelated OpenCode inline JSON settings and MCP servers; malformed inline configuration blocks launch with an actionable error.

- **Sidebar cards: folder/git icons no longer crush on long paths.** A
  deep workspace path flex-shrank the row's folder or branch SVG to a
  sliver; the label ellipsizes now and the icon keeps its 12px slot.

- **Sidebar checklist: no more `undefined/undefined`.** Creating a plan in
  the same turn as a refused edit/bash posted an empty `blocked` marker that
  could land after the real list; the disclosure then concatenated missing
  numbers. Absent plans render as silence again (ADR-0092); a parallel
  mutator no longer overwrites a plan this task already wrote.

- **Stable Agent CLIs layout.** All tabs share the same page width, card
  padding and tab alignment. Switching to Packages no longer narrows the
  page; desktop scrolling keeps its horizontal space reserved. Sessions and
  CLI setup actions wrap within narrow screens.

- **Native settings recovery.** Keep unsaved model patterns across temporary refresh failures, block stale writes until retry succeeds, and leave mobile agent settings and global key bindings usable when Pi defaults cannot be read. Desktop tool-mode restart failures stay visible as partial success and stop the sequence. Tools and Checklist menus now fit within the desktop viewport.

- **Pins: opening a pin with a heading or a list no longer restores an
  "unsaved" draft nobody typed.** The editor normalizes the markdown it
  loads and announced that as an edit; loading is silent now.

- **Mobile: the Work list wears the workspace's project favicon.** The
  workspace groups on the phone's Work screen showed a plain folder icon
  even when the project has a favicon the desktop sidebar displays. The
  group head now uses the same `hasFavicon` advertisement and
  `/api/workspaces/{id}/favicon` endpoint, falling back to the folder
  icon when the project has none or the file fails to load.

- **Schedules in a zone with daylight-saving time no longer hang the
  daemon.** `internal/cron`'s next-match search stepped the wall clock by
  hour and looped forever across the spring-forward gap (02:00 does not
  exist, so the step went backwards); it now steps by duration through the
  gap. Automations never hit it because the host zone has no DST; pin
  reminders in a named zone would have.

- **Web: CLI favicons no longer vanish on the dark theme.** The dark-mode
  invert for transparent monochrome vendor marks was scoped to the Agent
  CLIs view only, so terminal rows in the mobile Work list and the desktop
  sidebar, tabs and inspector rendered black-on-dark. The rule now applies
  on every surface; Pi and colored raster fallbacks keep their native
  colors.

- **Pins: limits refuse instead of truncating, large screenshots can be
  annotated, an edited sketch shows its new picture, and two editors no
  longer overwrite each other.** A title or note over the limit answers
  400 with the limit spelled out (the studio counts the note near 100 KB)
  instead of a byte-sliced text that left invalid UTF-8 behind; tags are
  capped at 40 characters and whitespace folds to one `-`. Annotating an
  image keeps the picture by reference, so the drawing's 2 MB cap is about
  the drawing and a 6 MB screenshot annotates fine; the old "scene is
  required" message for that case is gone. Preview URLs carry the file's
  version, so re-saving a sketch changes its thumbnail and the picture in
  the note at once. Save sends the version it loaded and a 409 offers
  Reload instead of losing the other writer's edit; the studio retains an
  unsaved draft across navigation and reload ("Unsaved changes restored",
  Discard). A file dropped on a new pin names the pin after the file, never
  "Untitled", and Cancel offers to delete a pin created that way. The
  sidebar follows the change feed instead of refetching on every
  navigation, and the list no longer carries note bodies (nor do
  `pin.created` / `pin.updated`, which now share one summary shape).
  Sketch bytes are written atomically; a boot sweep removes attachment
  directories whose pin is gone; downloads keep accented names
  (RFC 6266). Pins v2 review: `docs/plans/pins-v2.md`.

- **Inbox: replies to `ask_human` questions asked from a terminal pi now
  actually reach the terminal.** A pi running as an Agent CLI terminal
  filed questions as `pi (unmanaged)`; the reply path only delivers to
  agent-sourced items, so replying marked the item done with a "Reply
  sent" toast while nothing was ever delivered. Terminal-sourced items
  now carry the terminal's identity and the reply rides that terminal's
  receiver back into the exact session that asked (ADR-0037 amendment,
  ADR-0089's door in reverse); failures reopen the item with the reply
  preserved. A blocking question from a source with no channel at all is
  now refused visibly and stays open — it can no longer be closed while
  nothing was sent. Requires pi-inbox 0.2.0 installed in the pi session
  (`pi install -l …/packages/pi-inbox`); items filed by 0.1.x as
  `pi (unmanaged)` must still be answered by hand.

- **Mobile: the installed web app's tab buttons sit as low as the platform
  allows.** In standalone the tab content was centered in a 56px bar that
  ends at the (shortened) layout viewport, leaving ~11px of dead nav below
  the labels on top of WebKit's unreachable strip; the content and its pill
  now anchor to the bar's bottom edge there. The standalone detection also
  **measures instead of assuming** (screen height vs. visual viewport;
  WebKit 317153 reports the behavior varies with the iOS generation and the
  icon's install date), so edge-to-edge installs keep their real insets.
  An opt-in probe, opening the app with `?strip-probe=1`, extends the shell
  into the unreachable strip to test whether element painting survives —
  if labels survive at the screen's physical bottom, that offset can become
  the default; if they are clipped, the answer is no.
  Preferences → Layout now exposes that extension as the user's own
  **Screen edge** choice (clipping risk stated in the option label).

- **Mobile: the active-tab pill covers the label, and the installed web
  app no longer shows a dead band above the home indicator.** The active
  pill used to hug the icon only (28px tall) and its bottom edge cut
  through the label's first two pixels; it now sits behind icon + label as
  one surface. On iOS home-screen installs (standalone), WebKit parks a
  status-bar-sized strip *below* the layout viewport that no element can
  reach and still reports `env(safe-area-inset-bottom)` inside it — a
  double count that wasted ~81pt under the tab bar (WebKit bugs 313800,
  254868). The shell now flags iOS standalone at bootstrap
  (`navigator.standalone`), treats the bottom inset as already reserved
  (Safari and Android keep real insets), and paints the canvas so the
  unreachable strip continues the surface it sits under — the tab bar's
  panel on tab screens, plain content on pushed screens — instead of a
  black band. Recovers the 34pt of doubled inset for content.

- **The dashboard's reliability numbers were wrong for every guest CLI, and
  said otherwise.** Claude Code showed 0 errors against 409 real ones (a
  tool's failure comes back on the *user* turn, which the counter skipped),
  45 aborts that never happened (a normal `stop_sequence` was read as one)
  and none of its 4 refusals; Codex showed 620 prompts against 455 (its own
  AGENTS.md injections were counted as the person's); a week of Claude Code
  showed 42 sessions where 30 ran (subagent transcripts counted as their
  own). The "What each CLI reports" matrix now derives from what the parser
  actually saw instead of a per-CLI claim, and the tests parse real lines
  from each CLI's store. (ADR-0097)

- **A prepared command no longer swallows the next one.** A git command typed
  into a terminal and never submitted stayed on the prompt; the next one
  landed glued to its end and the shell answered `fatal: only one reference
  expected`. The line is cleared before anything is typed — for the Inspector
  rail's Git menu as well as the graph's. Stated plainly: anything you had
  typed at that prompt and not submitted is discarded, where before it was
  corrupted.

- **Undo never composes `reset --hard` over work you just made.** Undoing a
  commit uncommits it and leaves the changes staged; undoing a merge, rebase,
  pull, cherry-pick or revert moves the branch back with `--keep`, which
  refuses rather than lose a local change. A hard reset is offered only to
  undo a hard reset.

- **A git action that could not be sent no longer reads as sent.** A busy
  repository or a moved terminal keeps the form open with the reason; the
  graph does not wait for a change that was never asked for.

- **A worktree created from inside a worktree is a sibling, not a child.**
  The command names `<repository>/.worktrees/<name>` absolutely instead of a
  path relative to wherever the terminal sat.

- **Branches of a second remote work.** `upstream/feat-x` is fetched from,
  pulled from and deleted on `upstream`, not refused as "not a branch".

- **Keyboard focus stays inside a git action form** that has no field of its
  own (merge, fetch, rebase…); it landed on the page behind before.

- **The graph's "waiting" line no longer spends its patience while the tab is
  hidden**, and when it does give up it shows the repository as it is now.

- **Browser-tab icon is back on `/desktop/` and `/mobile/`.** Since the
  desktop/mobile split (2026-09-05) the app pages linked the favicon, Apple
  touch icon and PWA manifest under the application path (`/desktop/favicon.svg`),
  where nothing is served — tabs fell back to the browser's placeholder icon
  and the manifest 404'd. The three brand links now keep pointing at the
  site-root files (as ADR-0072 intends), and the build enforces it.

- **Reloading no longer announces every agent that was already waiting:**
  the needs-you pass could not tell "nothing is waiting" from "nothing has
  been read yet", so it seeded from the first render's empty fleet and the
  first real answer looked like a fresh arrival — a reload with three
  blocked agents raised three sticky cards on top of the badges that
  already said so. The pass now waits for the fleet to have been read
  once. A question that arrives while you are looking still announces
  itself, which is the whole point of the card.

- **Agent CLI attachments no longer dirty the project's `.gitignore` or
  pile up forever:** attaching an image or file to a Claude Code/Codex/
  Grok/Hermes/Pi terminal (ADR-0089) used to append a silent, uncommitted
  `.picode/drop/` line to the project's own tracked `.gitignore` on the
  first attach — showing up as an unexplained `M .gitignore` in `git
  status` — and did nothing at all in a project with no root `.gitignore`,
  leaving every staged attachment fully untracked. A nested
  `.picode/.gitignore` now covers the folder unconditionally, without
  touching the project's file either way. Staged attachments older than
  7 days are also swept the next time anything is attached in that
  project — nothing removed them before.

- **Terminals reattach after a phone lock:** locking the phone (or any
  network drop) no longer leaves a dead "— detached —" terminal that
  must be exited and reopened, losing the reader's place. The browser
  reattaches automatically — backoff 1 s → 10 s for up to ~95 s per
  burst, an immediate retry when the app becomes visible or the network
  returns — keeping the same xterm pane; tmux preserves the copy-mode
  scroll position across attaches. A tmux session that is truly gone
  says "Session ended. Reopen the terminal." once, and heals by itself
  if the session comes back.

- **Mobile recovery:** partial fleet failures preserve prior results; stale
  session responses cannot replace the newly selected CLI. A CLI diagnostic
  such as "No models available" no longer appears as a provider/model.
  Automation actions reject duplicate taps and restore rejected toggles;
  session handoff keeps controls locked through a confirmed retry and shows
  execution failures inside its sheet.

- **New Agent CLIs get Activity reporting on.** A CLI added to the catalog
  no longer imports with the switch off because it was missing from
  `enabled.json`. OpenCode rows frozen that way are turned on once at boot.
  Opening an OpenCode terminal prints `Starting OpenCode...` until the TUI
  draws, so the pane is not a blank cursor.

- **llama.cpp setup recovery:** failed or interrupted initial setup removes
  only its recorded empty folders, preserving existing or unexpected content.
  Unreadable ownership proof retains the recovery record until access is restored.

- **Phone terminal: extra keys stay off until you open the keyboard.**
  Opening a terminal no longer focuses xterm on attach, so iOS does not
  show the extra-keys row without the phone keyboard. The header keyboard
  button lets that tap summon the IME. Pinning the shell to the visual
  viewport requires a real shrink against the unfocused baseline, so a
  rest-state gap no longer leaves a black strip under the TUI.

- **Update checks failed on real installs**: install-method detection now
  resolves symlinks (`~/.local/bin/claude` → the native versions dir) and
  reads wrapper scripts (Hermes Agent), so update/reinstall/uninstall
  controls appear for actual installs instead of "No managed lifecycle".

- **Owned llama.cpp model cleanup** now recognizes verified Hugging Face cache
  downloads even when the router omits file paths. It records the exact blob
  and snapshot link, validates their hashes and refuses shared or changed files.

- **GitHub CI on macOS**: the worktree test compares symlink-resolved
  paths (`/var` → `/private/var`), and the terminal run/type routes
  validate the request (404/409) before asking for tmux (503), so the
  matrix is green without tmux installed.

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

[Unreleased]: https://github.com/cfpperche/picode/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/cfpperche/picode/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/cfpperche/picode/releases/tag/v0.1.0
