# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

**Agent contract:** this file is assembled, never typed into (ADR-0105). A
commit with a user-visible change writes one fragment,
`docs/changelog.d/<branch-slug>.md`, with Keep a Changelog sections (`### Added`,
`### Fixed`, …) inside it; `make changelog` folds every fragment into
`[Unreleased]` on `main` before a release. The pre-commit hook refuses a direct
edit to the entries here, so the fragment is not a convention — it is the only
way in. The repository's official language is English (see `AGENTS.md`);
changelog entries included.

## [Unreleased]

## [0.4.0] - 2026-09-21

### Added

- **The key-map API is documented for every agent CLI**: `GET /api/cli-keys?cli=`
  answers one shape — the map's file and catalog, the host's platform, when the
  CLI picks an edit up — and `PUT` writes it. Pi answers today; the other eight
  answer with their state (and a refusal that names why: *"Grok does not allow
  its keys to be remapped"* is not the same fact as *"PiCode cannot write Codex's
  key map yet"*). See it in [the API reference](/api/).

- **Keyboard → Find by key**: press a chord and the map narrows to the actions
  that answer to it — the fastest way to answer "what is `Ctrl+T` doing?".
- **Keyboard → Reset all**: hands every key this pane changed back to pi's
  default in one write, and leaves keys PiCode does not know where they are.
- A **warning on the chords a browser keeps**. Five of pi's own defaults
  (`Ctrl+T`, `Ctrl+W`, `Ctrl+N`, `Ctrl+PgUp`/`PgDn`) never reach the terminal
  inside PiCode; the rows that use them now say so instead of silently doing
  nothing in a browser tab.
- `app.thinking.save` (`Ctrl+S`) is on the map: the pane listed 89 of pi's 90
  actions.

- Observe project deliveries in desktop and mobile Git views, with branch inclusion, recorded checks, agent associations, evidence details and explicit incomplete or stale states.
- Record best-effort scoped-check, full-check and integration receipts; `picode delivery show` and MCP report the same observed integration and validation facts. Execution queues and deployment observation remain separate follow-ups.

- Register revision-bound delivery declarations and request review with `picode delivery` or the optional MCP delivery tool. Both share launch identity, durable retry receipts and version checks; integration queues and deployment remain unavailable.

- **Pins, Backup and Dev servers have guides.** Three shipped surfaces had no
  user documentation at all: Pins is announced in Getting started as a sidebar
  tab and was never explained, Backup had seven API routes and not one
  sentence, and the Inspector's Servers tab existed only in the changelog.
  [Pins and reminders](/guide/pins), [Backup and restore](/guide/backup) and
  [Dev servers](/guide/dev-servers) now cover them, including the two things
  a backup user has to know before they need them: a snapshot carries the
  credential vault but never the key that opens it, and what a restore
  replaces.

- **Installed plugins group by where they came from.** A list that mixes origins
  (Hermes' bundled plugins beside yours, Claude Code's claude.ai-managed ones
  beside the machine's) now carries a header per group — `Installed by you`,
  `Ships with the CLI`, `From a marketplace`, `Managed by claude.ai` — keyed on
  the vendor's own provenance word. A single-origin list gets no header.
- **The plugin filter says what matched.** The run of text a filter matched is
  highlighted in the name and in the description, instead of the card merely
  surviving the filter.
- **A marketplace chip filters the catalog.** Clicking a marketplace name — on a
  catalog card or in the sources row — narrows the list to that marketplace, and
  clicking it again clears the filter.
- **An empty plugin list points at the CLI's own catalog.** Where the CLI can
  install and its catalog was read, the empty state offers up to three real
  entries with Install, instead of only a link to the Marketplace.
- **The toolbar rides along.** The filter and the update check stay pinned while
  a catalog of hundreds of rows scrolls under them.

- **"Check for updates" for the other agent CLIs' plugins.** PiCode compares
  what each CLI has installed with what that CLI's own catalog offers and marks
  the plugins that are behind (`→ 0.3.0`), which is where **Update** appears.
  Before the check the pane offers the check itself instead of an Update button
  on every row, and a CLI whose catalog cannot be read says so rather than
  reporting that everything is current. Codex, OpenCode and Antigravity keep no
  Update action: those CLIs have no update command of their own.

- **Desktop: Ctrl+R and F5 reload the page on screen.** A work-browser or web-app tab reloads that site; everywhere else reloads PiCode. A terminal keeps Ctrl+R for reverse search.

- **Paste files copied in Explorer straight into an agent terminal.**
  Screenshots already pasted as bytes, but Explorer file copies arrive as
  references the browser never exposes — the attach never opened. In the
  desktop app an empty paste now asks the Windows-native shell, which
  reads the clipboard files and stages them through the same drop door
  (4 files, 4 MB each). Real browsers are untouched.

- **Desktop/Web:** removing an agent now offers **Undo** in the confirmation
  toast. Undo re-creates the agent in the same workspace with the same name,
  CLI and config, re-attaches its session history, and opens its tab again.
  The old id is gone for good (automations pointing at it stay broken), and
  sessions only come back if the dialog's "Also delete sessions" was not
  picked. Per-agent package selection is the one setting Undo cannot carry
  over.

- **Deploys and desktop restarts can no longer race.** `make deploy`, `make desktop-restart` and a direct `picode deploy` serialize on one lock; a CLI deploy caught behind it says so and waits. The desktop swap refuses to kill the shell while the WSL keepalive is not alive (the state that loses every session to the distro's idle reclaim) and ends with a real daemon-health verdict.

- **A refusal that needs you comes with the command to run.** When a CLI turns
  a plugin action down because only a person in a terminal can answer it — Grok
  asks you to trust the source, a marketplace can ask for a command to be
  confirmed — the pane now shows the CLI's own words *and* the exact command,
  with **Copy command** and "Run it in a terminal." It works for the actions
  that answer immediately and for a background install or removal that failed.

- Right-click on the globe offers **Open browser** / **Close browser split**
  and **Open in new tab**, instead of the generic window menu.

- **`picode inbox notify|ask` — the Inbox door for any CLI, no MCP config
  needed.** Any guest CLI (Claude Code, Codex, a plain script) can file an
  FYI into your Inbox with one command, or ask you a blocking question:
  `picode inbox ask --question "…" --wait` polls until you answer in the
  Inbox and prints your reply on stdout, hours if needed. Same daemon
  discovery as `picode mcp` (`--url`, `PICODE_URL`, or server.json), same
  items, same app.

- **Packages works for the other eight agent CLIs.** Claude Code, Codex, Grok,
  Hermes Agent, OpenCode, Muse Code, Antigravity and Omp each get the Packages
  pane on their own CLI page, driven by the vendor's own plugin commands:
  installed list, install, remove, enable/disable, and the CLI's marketplace
  where it has one. Pi's pane is unchanged.
- **Each pane offers only what its CLI has.** Where a CLI exposes no verb the
  pane says so in one line instead of showing a control that does nothing —
  Codex has no plugin enable/disable, OpenCode has no plugin marketplace,
  Antigravity has no plugin update.
- **Install and remove run as jobs.** They keep running with the browser
  closed, refuse while that CLI has a live terminal (a plugin loads on the next
  start), and never replay after a PiCode restart. Antigravity, Muse Code and
  Omp got their plugin stores read here for the first time; OpenCode's plugin
  list is edited in its own config, with every other byte of the file kept.

- **Device toolbar ("Responsive width") in the work browser — the native
  half.** The ⋮ menu gains the reference's "Show device toolbar" row; the
  strip above the page offers width presets (390/768/1024/1280), zoom −/+ and
  a reset. Picking a width narrows the page to a centered preset-width rect of
  the pane (WebView2 bounds arithmetic, no CDP — mobile UA/touch/DPR would be
  the emulation half and needs an ADR); zoom rides the controller's
  `ZoomFactor`, clamped to WebView2's 0.25–5.0. State is per tab in the shell
  and rides `btab_meta`, so the strip survives tab switches and route returns;
  hiding the toolbar resets the tab (width, zoom, full pane bounds). The
  bounds/reset/zoom-clamp math lives in pure `desktop-shell/src/responsive.rs`
  (host-tested) and the preset/label/zoom ladder in
  `web/browser/src/lib/responsive.js`; the on-screen resize/zoom behavior is
  pending the owner's live Windows acceptance.

- **Codex's Memory pane shows its version history.** Codex keeps that folder
  under git, so the pane now says how many revisions it holds, when the last
  one was written, and whether any file has changed since — and each row's
  Modified date comes from the revision that actually changed the file instead
  of the filesystem's timestamp, which moves every time Codex rewrites a file
  with the same contents. Any memory store the CLI keeps under version control
  gets the same line.

- **Paste screenshots and files straight into an agent terminal.**
  Ctrl+V / Ctrl+Shift+V with an image or file on the clipboard opens the
  attach bar with the files staged, delivered through the same drop door
  as Attach files… — no more copying paths by hand. Text pastes keep the
  terminal's native paste; plain shells answer that attach is for Agent
  CLI terminals.

- **Mobile: the Inspector.** The desktop rail's review surface, re-shaped
  for the phone: on the agent screen, a header button (the desktop's
  inspector toggle) opens it as a drawer over the conversation — Changes
  (folder-grouped with sums, an `All | This agent` scope that follows what
  the session's tools touched, and worktree following), Files and PR — and
  Back returns to the agent. From the Work tab, a workspace's changes chip
  opens the same screen pushed full-width. Git actions keep the three
  doors: prepare in the terminal, run when idle, ask the agent.

- **The Memory pane is a table you can audit with.** Beside every memory it now
  shows how many other memories cite it, its size, when it last changed, and
  the things worth acting on: a memory the index does not name, and a link that
  points at a file that is gone. Any column sorts, the kind chips filter with
  their counts, and the file the CLI loads every session stays pinned at the
  top. On a phone the same numbers ride under the title with a Sort control.
- **The index budget.** Claude Code reads only the first 200 lines or 25 KB of
  `MEMORY.md` at the start of a session and drops the rest in silence. The pane
  now shows how close the index is to that limit, and names any row in it that
  points at a file that no longer exists.
- **Delete several memories at once,** with a confirm that names what it
  breaks: how many links in other memories will point at nothing, and how many
  of them the index still names.

- **Use: run a CLI on a saved account.** Each guest CLI's Providers pane can
  now put a saved login into that CLI's own login file — the same thing **Use**
  does for Pi — so Claude Code, Codex, Grok, Hermes, OpenCode, Muse or
  Antigravity runs on the account you pick. The account in use is marked on the
  row, and it is read back from the CLI's own file, so a login made in the
  CLI's TUI shows up here too.

- **One credential vault for every agent CLI.** Claude Code, Codex, OpenCode,
  Grok, Hermes, Muse, Antigravity and Omp now have a Providers pane of their
  own: see which accounts the machine holds, import the login a CLI already
  has, add an API key, verify it, pause it or sign out — one account per
  provider or ten.

- **Automated coverage for the work-browser capture path.** The decisions of
  `capture_png`/`btab_preview` (capture completed / failed / timed out / tab
  gone mid-capture) and the read-back gate now live in a pure module
  (`desktop-shell/src/preview.rs`) with host tests — a still is served only
  with non-empty pixels; a zero-byte capture ("the page has not painted") is
  refused. The legacy hide/restore decision of the options menu
  (`coverDecision` in `previewStill.js`) carries the same table tests. The
  COM capture and native hide themselves stay owner-verified on Windows.

- **Pi's Settings pane gained five of its own settings** on the This machine
  layer: Theme, Hide thinking, Quiet startup, New folders (pi's trust
  default) and Shell. Pi persists about forty keys and the pane showed eight,
  so a machine with a theme set saw no theme row at all. Each name and its
  allowed values were read out of the installed pi build before being
  declared, and a key pi keeps for the machine is refused by name if it is
  sent to the workspace or agent layer.

- The desktop Pi view now attaches interactive sessions through the canonical Agent CLIs terminal binding and exposes the Chat/Terminal switch in the agent toolbar.

- Mobile agent views now keep Chat and Terminal on one agent route, with the
  view switch in the contextual toolbar and a durable terminal deep link.
- Non-Pi agent rows now attach their mobile Terminal view to the bound
  Agent CLIs terminal surface, reusing the canonical terminal session path.

- **Settings for every agent CLI.** Claude Code, Codex, Grok, Hermes Agent,
  OpenCode, Muse Code, Antigravity and Omp now have a real Settings pane at
  `#/clis/<cli>/settings` that edits the CLI's own config file — model,
  approvals, sandbox, reasoning effort, memory switches and more, each row
  named with the vendor's own key. A CLI with one config file shows one layer;
  one with a workspace file shows both, with the file it writes named under the
  switcher and provenance on every row (ADR-0163).
- **A Memory pane for every agent CLI** at `#/clis/<cli>/memory`, showing what
  the CLI has remembered between sessions. Claude Code, Hermes, Muse Code and
  Omp are read, edited and pruned from the browser wherever they keep plain
  markdown; Grok and Codex are read here and cleared with their own command,
  because they generate those files; Pi and OpenCode say in one line that they
  keep no memory, and Antigravity says PiCode cannot confirm one (ADR-0163).
  Omp writes nothing until its memory backend is turned on, and PiCode reads
  its folder from two paths the vendor does not document, so that pane is
  usually empty.

- Document the execution plan and acceptance gates for the remaining work-browser items.

- Scope Omp agent `/resume` sessions to a durable private directory per workspace agent.

- Require an explicit agent terminal destination before sending annotations from an independent browser tab.

- **Inspector Changes follows dirty worktrees.** When the anchored folder is
  clean but a linked worktree of the same repository has uncommitted
  changes, the rail shows that checkout behind a Following pill (Back
  returns to the anchor); several dirty checkouts render one branch-headed
  group each. The branch chip, Git menu and commit dialog address the
  followed checkout, and its files open in the center through the same
  file tab. Works for agents, terminals and workspaces, with any CLI.

- **Windows setup finishes on a clean machine.** `picode-desktop install`
  now delivers the Linux binary and the runtime itself: an `install-picode`
  stage (the release matching the tool, verified, placed where provisioning
  resolves it) and an `install-runtime` stage (tmux, git, curl, Node.js from
  the NodeSource repository, pi) between account creation and provisioning.
  Ubuntu only; anything already present is left alone; a distro PiCode did
  not register asks before apt and npm run (`--yes` skips the question);
  the install ends with the shell running in the tray.
- **Per-account installs.** `install --user <name>` puts the picode binary
  in that account's `~/.local/bin` and installs pi for it alone (its own
  npm prefix, linked where the login shell finds it). Without the flag pi
  stays a system-wide root install. An unknown account fails fast naming it.
- **Install failures stay visible.** The launcher waits for the elevated run
  and reports its exit code instead of exiting 0; the last error line lands
  in `%ProgramData%\PiCode Desktop\install.log`; the elevated window pauses
  for Enter on failure rather than closing with the evidence.

- The address bar now suggests the pages you have already visited — the ones
  you typed first — with ↑/↓ and Enter to open one. The pane that runs without
  the desktop app offers only the servers it can actually frame.

- Sending annotations has no limit on how many: the whole set travels as one
  note (every pin's sentence, element, HTML, styles and screenshot) plus one
  crop per pin, and the crops that do not fit the paste are staged beside the
  note, which names every one of them.

- `docs/architecture/work-browser.md` — the work browser's own architecture
  file (who decides what, the command channel, tabs, the "a layer never paints
  over a WebView2" rule, permissions, annotations, and the traps that are
  architectural rather than dated).

- The annotation card now offers the element's own styles — text color,
  background, opacity, font, size and weight — and every change is a live
  preview on the page. What reaches the agent is the pair: what the page had
  and what you proposed, so a preview is never mistaken for the page.

- Marketplace connector cards now have a **Docs** link to the server's
  website or repository (seed cards with a cookbook page go there). Cards
  without a URL hide the link.

- `find_webview`: one resolver for "the webview of this tab", used by the
  preview, the annotate mode, the trash and the state pull.

- A workspace can create an Agent CLI from its **New** menu (Claude, Codex,
  and the others that are installed). You no longer have to go through
  Agent CLIs first.

- Binding a Claude Code, Codex or OpenCode principal turns on PiCode's
  tools for that terminal (computer, browser, Inbox, checklist). Other
  CLIs still pick them up from Connectors. Grants stay in Settings.

- A managed CLI that reports **needs you** files one Inbox item with
  **Open terminal** (and a push, if you allow actions). Ordinary shells
  still only show the chip on the row.

- A workspace can bind an Agent CLI terminal as a **managed principal**
  (`term:<id>`): the same identity Computer and Browser grants already use,
  without creating a Pi agent or opening a chat. Create with
  `POST /api/workspaces/{id}/principals`. Removing the binding leaves the
  terminal and the vendor's files alone.

- Annotate mode mirrors the page: pressing Esc in the page turns the strip
  off too (the pull says "off"), and a page with no script yet (a navigation
  in flight) is told apart from an off page.

- **Inbox and Checklist for Claude Code, Codex and the other agent CLIs.**
  `picode mcp inbox` gives them `notify_human` and `ask_human` — a question
  waits for your answer in the Inbox and returns it to the agent — and
  `picode mcp checklist` shows their plan as the current step on the
  terminal's card. Two more cards in Connectors and two more switches under
  PiCode tools in Launch settings.

- The annotate harness doubles as the navigation check: after a real
  navigation the script is gone, and one re-injection arms the new document
  again and re-reports its (empty) state.

- **`type` can paste.** `mode: "paste"` sends the text through the
  clipboard and one Ctrl+V, for long text or apps that garble synthetic
  keystrokes (Windows 11 Notepad when busy). The clipboard's text is put
  back afterwards.

- **Computer lab: Embed into PiCode (spike).** Two lab-only buttons reparent
  a chosen window into PiCode's main window and put it back, so the owner
  can measure whether Windows apps could get a pane like the work browser.
  Nothing in the product uses it.

- **Apps: clear one web app's data.** The tile menu gains Clear data — signs you out of that app and deletes what it stored inside PiCode, without touching other apps or the work browser. Removing a tile now clears its storage too.

- **Annotate mode matches the reference.** The strip replaces the URL bar
  while annotating (exit, discard-all, undo, crops toggle, hints, Send N);
  notes accumulate as numbered pins, each edited in an anchored card
  (Cancel/Save) and collapsed to a chip (text, options, remove); Send ships
  the whole set as one context to the agent terminal.

- **Connectors tolerate vendor JSONC configs on read.** Trailing commas and
  comments that OpenCode and friends hand-write into their JSON configs no
  longer break the Connectors pane; every save still lands strict JSON. A
  blocked config file shows one line naming it, with **Open** and **Retry**;
  other config layers keep listing.

- **Apps: multiple accounts of the same service.** The tile menu gains **Add another account** — a second, independent copy of a web app you already have (its own logins, its own data). Trying to add an address that is already installed now offers the same choice instead of just refusing.

- **PiCode tools for Claude Code, Codex and the other agent CLIs.** `picode
  mcp computer` and `picode mcp browser` serve the computer and browser tools
  over MCP with the same names, parameters and answers as the pi packages.
  Turn them on per launch in a CLI's Launch settings (Claude Code, Codex,
  OpenCode), or add the "PiCode · Computer" and "PiCode · Browser" cards from
  the Connectors pane at workspace or machine scope. The switches in
  Settings ▸ Computer and Settings ▸ Browser apply to the terminal the CLI
  runs in.

- **Apps: each web app keeps its own logins.** Installed web apps now live in their own storage inside the desktop app — two accounts of the same service work side by side, and clearing the browser data no longer signs every app out. Apps installed before this change keep the old shared storage until you remove and add them again (they will ask to sign in once).

- **Connectors for Muse Code and Hermes Agent — all nine CLIs covered.** The Connectors pane now manages Muse's `~/.config/muse/settings.json` (`mcp_servers` block, `streamable_http` transport, per-server on/off; `schema_version` and `mode` preserved) and Hermes's `~/.hermes/config.yaml` (`mcp_servers` block edited as a YAML node tree so comments, key order and `${VAR}` placeholders survive; malformed files refuse without writing). Both CLIs keep a single config file, so "this folder" scope points at it instead of inventing one.

- **Apps: Refresh keeps a web app tile truthful.** The tile menu gains Refresh — PiCode re-checks the site and updates its icon, colors and start page (your name for it and its address stay; if the site is down, nothing changes).

- **Computer use, visible.** Each step an agent takes on the computer shows
  its capture in the chat as the last capture, the dashboard counts the
  steps and the time an agent spent acting on the desktop, and the public
  guide explains what the switch in Settings ▸ Computer means.

- **Computer use for agents.** Install the `pi-computer` package and switch
  an agent (or a CLI in a PiCode terminal) on in Settings ▸ Computer: it can
  then see the screen, click, type, use the clipboard and open programs
  through the desktop app, with your own permissions. Every call, allowed
  or refused, is listed on that page. Off by default.

- **Connectors for OpenCode and Grok.** The same Connectors pane now manages their native MCP configs: OpenCode's `opencode.json` (`mcp` block, command arrays, per-server on/off; `XDG_CONFIG_HOME` respected) and Grok's `.grok/config.toml` (`[mcp_servers.*]` tables edited with a surgical splice that keeps comments and `${VAR}` placeholders byte-for-byte; Grok has no per-server switch, so rows show none).

- **Computer lab** in the desktop app's tray: a page that drives the new
  computer actuator by hand — list displays and windows, take a screenshot
  of a monitor or a window, click and type where you point on the image,
  read a window's accessibility tree, use the clipboard, open a program. It
  is the acceptance surface for computer use (ADR-0148) before any agent is
  wired to it; nothing changes for agents yet.

- **Connectors for Omp and Antigravity.** Manage their MCP servers from the same pane: Omp in `~/.omp/agent/mcp.json` (and your workspace's `.omp/mcp.json`), Antigravity in `~/.gemini/config/mcp_config.json` (and your workspace's `.agents/mcp_config.json`). Each CLI keeps its own settings, servers switch on and off per entry, and sign-in follows the vendor — the Omp TUI command, or Antigravity's Agent Settings.

- **Agent CLIs: additional folders control.** Claude Code, Codex and Omp launch editors offer one-per-line extra workspace directories, written as repeatable `--add-dir` flags alongside the other quick controls.

- Connectors (MCP) management for Claude Code and Codex (ADR-0150 phase 1): `#/clis/claude-code|codex/connectors` manages each CLI's native MCP config — Claude Code's workspace `.mcp.json` plus its vendor CLI for user scope, Codex's `config.toml` with comment-preserving surgical edits — behind the same pane and API shape as Pi, with honest "configured" status and per-CLI toggle capabilities.

- **Windows Hello and passkeys** (Settings ▸ Browser ▸ Password manager): a
  row that opens Windows' own Sign-in options. PiCode does not store or
  unlock passwords and passkeys — Windows does — so the row points at the
  screen that manages them instead of imitating it. It appears only in the
  desktop app, and it is the first target class besides `http(s)` that the
  shell will hand to the operating system.

- **Dashboard covers Omp, Muse Code and Antigravity.** "What each CLI reports" now has nine rows: Omp reports spend, tokens, turns, tools and agent time; Muse Code reports prompts, turns, tools and model (no tokens or cost in its logs); Antigravity reports step counts as turns from its conversation index.

- **Apps: install web apps as desktop shortcuts.** The Apps tab gains **+**: type a web app address, PiCode checks it answers and reads its name and icon, and the tile opens the site inside the desktop app with its own tab and login. Rename or remove from the tile's menu; unread counts in the page title show on the tile. In a plain browser the tile explains it needs PiCode Desktop.

- **Agent CLIs: quick launch settings.** The launch editor offers verified model, approvals, reasoning/thinking and sandbox controls above the advanced argument fields, with a one-line warning on dangerous picks. Covered: Pi, Claude Code, Codex, Grok, Hermes Agent, OpenCode and Omp.
- **Agent CLIs: one-click YOLO flags.** Codex and Hermes expose their vendors' `--yolo`, Grok its "Auto-approve tools"; on Codex picking YOLO clears the conflicting sandbox/approvals choices (and vice versa), so the launch never composes a combination the CLI refuses.

- **The Servers panel names what a port is, and can stop it.** Each row now
  carries what the port actually answered (`page`, `api`, or nothing yet), the
  scheme that answered, the process behind it and how long it has been up —
  and a menu with **Copy address**, **Show terminal**, **Stop server…** and
  **Hide from the list**. *Open* only appears where the URL leads somewhere a
  browser can show: an agent CLI's internal HTTPS control channel is no longer
  offered as a page it could never be (measured: plain HTTP to that port answers
  `400 Client sent an HTTP request to an HTTPS server`). A port that only
  answers an API, started outside PiCode, waits behind **Show N more**.
  Stop signals only a process PiCode can still re-derive inside one of its own
  panes, with the pid and the process's start token checked at that moment
  (ADR-0151); a port PiCode cannot attribute is never stoppable. Hiding is keyed
  by the listener — port, pid, start token — so a new server on the same port is
  born visible.

- **Forget unused sites** (Settings ▸ Browser ▸ Site settings): one action
  drops the permission entries for sites you have not visited in the last 90
  days. Your every-site policies and anything you visited recently are left
  alone, and the action says what it forgot — or that there was nothing to
  forget.

- **Agent CLIs: Omp installs and updates per install method.** A missing
  Omp now offers **Install** through npm (like pi, Codex and Claude Code).
  Update checks follow the install: npm installs read the npm registry,
  native installs run omp's own `omp update --check` and parse its answer.
  A bun-global Omp is honestly reported as unmanaged — measured, npm
  would miss the bun copy and omp's updater resolves by PATH — so no
  update or uninstall buttons propose an action that would hit the wrong
  installation.

- **Agent CLIs: "Continue in Omp…" works both ways.** Omp's own sessions
  move to any other CLI (it reads its on-disk transcript), and other CLIs'
  conversations arrive as native omp sessions you resume with
  `--resume <id>` — written at omp's own minimal session shape, verified
  live against omp 18.2.4 before shipping. Omp joins pi, Claude Code,
  Codex, Grok, Hermes Agent, OpenCode, Muse Code and Antigravity as a full
  handoff source and target.

- **Agent history access** (Settings ▸ Browser): an agent may read where you
  have been — off unless you allow it, its own permission rather than a side
  effect of a grant. The `browser` tool gained the `history` verb (a search
  over url and title, newest first, capped at 200 rows) and answers with url,
  title and time only: no page content, no sessions. ADR-0146.

- **Agent CLIs: Omp is a full citizen.** Editable launch settings, the PATH
  wrapper with a presence lease (maintenance subcommands and protocol modes
  stay out of it), an **Activity reporting** toggle through omp's own
  extension API (it reports Ready and Working like pi), and **Check for
  updates** — `omp update` on native installs, npm on npm installs. omp
  refuses to start when launch arguments carry `--trusted-extension` next
  to PiCode's activity extension, so that combination is named in the
  launch preview instead of starting a broken terminal. Still needs
  Bun ≥ 1.3.14 on PATH.

- **Agent CLIs: Omp lists its sessions.** The Omp pane gains a Sessions
  tab: its own on-disk sessions under `~/.omp/agent/sessions`, grouped by
  folder, searchable, and resumable — **Open in terminal** starts `omp` on
  that conversation with `--resume <id>`. Launch settings, activity and
  managed updates are still without an adapter, and Omp still needs
  Bun 1.3.14 or newer on PATH.

- **Agent CLIs: Omp joins the catalog.** Omp (oh-my-pi, omp.sh) opens a
  terminal of its own — detection, Check setup (`omp --version`) and New
  terminal, with its own card (mark `Om`, vendor mark). Launch settings,
  Sessions, Activity and managed updates have no adapter yet: the Launch
  tab is a read-only preview, its terminals read Open, and install is
  guided through omp's own docs. Omp needs Bun 1.3.14 or newer on PATH;
  Check setup reports an older Bun's bundle error verbatim.

- **Add a site exception by hand** in Settings ▸ Browser ▸ Site settings: pick a
  site (or pattern, or `*`), a permission kind and the decision to remember, and
  the row lands saved — the same rows the Ask prompt writes, without waiting for
  a site to ask.

- **Continue in… (and resume) for Muse Code and Antigravity terminals.**
  Stopping one now pins the latest conversation the vendor store holds
  for that folder, so the sidebar menu offers Continue in… and the
  launch can resume it — the same pin wrapper CLIs get from their
  runtime. Sessions predating the terminal never pin.

- **`make land BRANCH=<name>`** — the landing rite from the root: it verifies
  the branch, refuses when the root's local changes overlap it, fast-forwards
  `main`, and runs `make ci`. It never commits, so it cannot sweep another
  session's staged work the way a `git add -A` in the shared checkout once did.

- **`*` in a browser grant means any site.** The domains field now takes the
  one token for "this agent may open anything" (still http and https only),
  and everything else keeps working as before: exact hosts, `*.example.com`
  for a domain and its subdomains, ports ignored.

- **Developer mode — full CDP access** in Settings ▸ Browser, off by default
  and labeled Elevated risk: an agent at the Full tier can name any Chrome
  DevTools Protocol method, not only the curated ones, and every call it
  makes — allowed or refused — is listed with the method, the agent and the
  outcome. Everything keeps working with it off.

- **Update and Reinstall for Muse Code and Antigravity.** Both pages now
  show the blue Update button when an update is available, and the `···`
  menu offers Check for updates and Reinstall like every other CLI —
  running the vendors' own commands (`agy update`; Muse's launcher as its
  own updater). Neither ships an uninstaller, so uninstall stays a guided
  link to the vendor docs, same as Claude Code.

- **Browser options menu, matching the reference.** The work-browser ⋮ menu
  now carries Find in page, Print, Zoom (− / 100% / + / reset), Take a
  screenshot, Passwords and autofill ›, Downloads, History, Clear browsing
  data and Browser settings.

- **Antigravity reports activity.** Working while it thinks, uses tools
  or starts up; Ready when idle — via a reporter PiCode installs as the
  CLI's title command (merged into your settings, removed on toggle-off;
  a foreign title command is refused, never replaced). No `hooks.json`
  decision hook ships, so nothing can gate your tools.

- **Browser permissions: Ask.** A permission kind can be set to Ask in
  Settings ▸ Browser ▸ Site settings. When a site asks, the prompt appears in
  that browser tab with Allow, Block, and Always allow — the last remembers
  the answer for the site. An unanswered prompt denies itself after a minute
  instead of leaving the site waiting.

### Changed

- **Git ▸ Delivery now reads as a page.** The view sits centred in one card —
  the same geometry Agent CLIs uses — with the target, filter, change list,
  detail and its empty, blocked and error states inside it. Git ▸ History is
  unchanged.

- **An agent can open the browser beside its session and drive that tab.** `open` launches the split; navigate, click, type and press run there, and you see the same page. Closing the split stops it. This is not a headless browser — unattended browsing stays on the runtime's own tool. Raw protocol still needs Developer mode and the Full tier.

- **Keyboard** is rebuilt (Agent CLIs → a CLI → **Keyboard**). One row per
  action, the chord drawn as a keycap (mono, 24px — it used to be a 36px box
  the height of a form control), the row's own **Add key** and **Reset** in a
  fixed column at the right edge, and a toolbar with the filter, a **Key**
  filter and facets that count: **Changed**, **Shared**, **Off**. A key two
  actions share is reported, not called a conflict — 52 of pi's 90 actions
  share one on purpose.
- **The row is the action, its chords, and what they mean, on one line.** The
  label is bounded and the keycaps follow it (they used to sit at the far edge
  of a 1240px card, hundreds of pixels from the action they belong to), the
  note about a shared or browser-kept key sits between them and the row's own
  actions, and a row is 32px — 48 with a note. 90 rows are ~3 200px of list,
  not the six screens of air an earlier build drew with the label, the keycap
  and the note each on its own line.
- **The toolbar holds one line.** Only the filter field gives up width when the
  pane narrows; the facets keep their labels and the row count and **Reset all**
  stay on that line. On a phone the bar wraps instead — filter beside the
  button, facets on their own row — and the count is left to the chips, which
  already carry it.
- A row that differs from pi's default wears the accent bar the settings rows
  already use, and the pane names the file it writes
  (`~/.pi/agent/keybindings.json`).
- The key map's own chords read as keys (`Ctrl+B`, `Page ↑`) rather than the
  file's spelling.

- **Apps in the sidebar can be rearranged.** Drag a tile anywhere in the
  grid. The order is saved and stays after a reload. A search still filters
  the grid and does not change the saved order. New apps land at the end.

- **Dragging a sidebar row shows where it will land.** The row in the list
  becomes a quiet dashed slot and slides into place. A lifted copy follows
  the pointer, and the neighbours ease over in a short move.

- **The sidebar keeps the order you leave it in.** Drag a workspace by its
  name, or an agent or a terminal by its row, to move it within that list.
  Agents and terminals stay in their own blocks. The same move is **Move up**
  and **Move down** in the row's menu. A new workspace or terminal lands at
  the bottom instead of sliding back into alphabetical order. The browser,
  the desktop app and the phone show that same order. A drag does not move
  an agent into another folder.

- **The tray menu is product-first.** `New browser tab` (redundant with the
  work browser's own new-tab button) and `Test notification` (the Phase 1
  notification spike) are gone. `Management…` moved up beside the actions it
  belongs to, and the owner-acceptance spikes — **Browser lab** and
  **Computer lab** — now sit under a single **Labs ▸** submenu instead of the
  main list.

- **The tmux guard moved to Agent CLIs.** The switch left Terminal defaults:
  that page is font, color and tmux options a terminal inherits, and the
  guard is none of those. It now sits above the CLI catalog on Agent CLIs
  (desktop and phone), and still applies to every terminal opened from
  then on — not to the CLI selected below it.

- **Providers** is one pane for every agent CLI: pi's roster moves onto it (Add provider, custom endpoints, Use, Pause, Sign out all still there) and the other eight gain what only pi had — **Usage** windows per account with a one-call **Check**, and **7d spend** from your own sessions. Same columns, same words, one place per account, whichever CLI you open.

- **Plugins are cards now, two per row.** The installed and marketplace lists
  of the other agent CLIs read as a grid of cards like Pi's Packages: name,
  version and the `→ 0.3.0` marker when the CLI's own catalog has a newer
  release, the plugin's description, its scope and origin, and its actions on a
  footer that lines up across the row. One column on a narrow window or a phone.
- A plugin the CLI reports as turned off keeps its buttons at full strength —
  **Enable** is the way out, so it no longer looks disabled too.

- **The attach bar follows the terminal's theme.** Under a dark terminal
  in a light app (or the reverse) it rendered in the app's colors; it now
  reads the terminal's Light/Dark choice and flips with it, live.
- **Terminal defaults moved into the user menu.** The gear is gone from
  the Terminals header; search "terminal" in the bottom-left menu (or pick
  Terminal defaults) to open the same global page. Per-terminal settings
  in pane menus are unchanged.

- The header globe opens a browser bound to the selected agent (or agent-CLI
  terminal). Shift+click, or **Open in new tab**, still creates a tab with
  no agent.

- **Providers**: a guided **Sign in** starts a CLI's own login (its binary, its browser or device-code flow) from the pane — finish it, press **Check now**, and the account lands in the vault. Muse's subscription login is read and written correctly (its own file shape), a second subscription of a store that names no account can be kept by naming it, and Codex, Grok, Hermes and Antigravity keep two subscriptions apart automatically from the account their own files carry. Vault rows a CLI's file cannot name are matched by the token they were saved with — no network call in a pane load.

- The Memory pane no longer sits flush against the tab bar, and its toolbar is
  one cluster on the left: the store switcher and the filter sit together
  instead of the filter floating alone against the right edge. The row is gone
  entirely when there is nothing to switch and nothing to filter.
- A read-only CLI now names its clear command inside the sentence, with a plain
  **Copy** button — the button used to read "Copy grok memory clear".

- The old mobile `#/changes` screen retired into the Inspector's Changes
  segment; existing links redirect.

- The first time PiCode writes a CLI's login file it keeps a copy of what was
  in it (`~/.picode/credfiles/<cli>-<time>.bak`) and tells you where.
- Nothing about a CLI's home changes: no new folder, no `HOME`-style variable,
  so settings, sessions and memory stay exactly where the tool expects them.

- **A setting added to Pi's pane now takes one edit instead of two.** The list
  that decides which value each layer shows is derived from the same table
  that declares the rows, so the two cannot disagree — a key added to only one
  of them used to render an empty control beside a row that said the value was
  set here.

- **Pi's Settings rows come from the same table as every other CLI's.** All
  eleven are declared in one place with their group and order, so the pane
  reads as one product with the other eight and a setting pi gains is a line
  rather than a component edit. The three rows that are not simple values —
  the model defaults, scoped models and the tool grid — keep the controls they
  always had instead of being flattened into something they are not.

- Pi's provider accounts moved into an encrypted vault
  (`credentials.json` beside its key, both readable only by you). Existing
  accounts are carried over the first time PiCode starts after the update;
  the old file is left where it was.
- A backup with secrets carries the vault but never the key that opens it: a
  snapshot restored on this machine keeps working, and a copy taken to another
  machine cannot be decrypted. That is the point of storing the keys
  encrypted, and it is the one thing to know before moving a backup around.

- **A setting that is on or off looks the same everywhere.** Every pane now
  uses one switch for a boolean instead of a switch in Pi's and a checkbox in
  the other eight. A switch cannot show "nobody set this", so a row nobody set
  is drawn at the CLI's own default and the line beneath it says whose default
  that is.

- A capture whose tab closes mid-flight now reports "the tab closed while the
  capture was in flight" instead of "capture timed out" — the same refusal,
  naming the right failure.

- Mobile agent cards now match the terminal cards: the status chip reads the agent's real state (working with an activity age, needs you, open, stopped) and the row menu carries the full lifecycle — start/restart/stop, open chat or terminal, launch settings, continue in another CLI, rename and remove.

- Route bound interactive agents through the shared terminal identity on mobile and expose terminal attachments in the agent view.
- Keep the desktop agent Chat/Terminal switch icon-only with accessible labels.

- Legacy unbound Pi interactive sessions retain their existing agent endpoint until restart; new sessions use the bound terminal identity.
- Agent and workspace cleanup now closes bound terminal panes through the canonical terminal identity.

- Saving a guest CLI's setting keeps the rest of the file byte-for-byte:
  comments, key order and every key PiCode does not know survive, and a file
  that changed on disk since it was read is refused instead of overwritten.

- Other Agent CLIs keep their native session-storage behavior. PiCode does not emulate per-agent `/resume` isolation for runtimes without a dedicated session-directory boundary.

- **The terminal settings report the tmux that is actually serving your
  terminals.** After a tmux upgrade the new program can talk to the older
  running server; the version shown on the System page (desktop and mobile)
  and in the tmux app's server view is now the server's, not the new
  program's.

- The prompt door now verifies deliveries for CLIs with a measured input
  reader: the composer must read empty before and after, and refusals
  name themselves (working, needs-you, occupied, busy). The response
  carries a delivery receipt; CLIs without a reader answer "unverified".
- Automations can target CLI agents: the prompt is delivered through the
  agent's launch terminal via the door, and the receipt becomes the run's
  outcome (done / skipped / failed with the door's named reason).
- The graph and pane ask on a non-pi CLI terminal is delivered through
  the door (fire and forget, provenance recorded).

- Pi agents reach full row parity with the other agents: the face wears
  pi's favicon (same art terminal rows use), and the menu offers
  Continue in… from the agent's own pinned session — the same submenu the
  other agent rows have.

- A CLI agent row offers Continue in… (ADR-0088): a conversation pinned on
  its terminal continues in another CLI, the same submenu terminal rows
  have.

- The address bar says "Enter a URL". It promised a search ("Search or enter a
  URL") while typing a phrase produced an invalid-URL error; search from the
  bar is its own decision and is not built.

- Naming: CLI runtime agents are called CLI agents — "guest" is gone from
  code, copy and docs (ADR-0160: PiCode runs multiple agents; Pi is one
  runtime among several).

- A guest agent's launch now carries `PICODE_AGENT_ID`, so `picode mcp`
  and the computer/browser tools resolve the agent as the caller — grants
  given to the agent in Settings match without per-terminal setup.
- Inbox "needs you" items for a guest are filed for the agent (not the
  terminal), close when the agent or its terminal is deleted, and their
  Open terminal action follows the agent's bound terminal.

- A guest agent row (and the collapsed faces strip) now wears the CLI's
  favicon — the same identity terminal rows use — instead of a letter
  fallback.

- Guest agent rows (Claude Code, Codex, …) name their CLI instead of
  "Pi agent", show the terminal's status, and carry Launch settings and
  Start/Restart/Stop terminal in their row menu. Mobile starts and stops
  guests through the same terminal launch.

- **Connectors show real status for Claude Code, OpenCode and Hermes.** Rows in those Connectors panes now report Live or Failed from each CLI's own status report — no more guessing whether a service actually answers. The other CLIs keep their honest "configured" state (they expose no status signal to read).

- Cancelling an annotation (or removing it, clearing the set, or leaving the
  mode) puts the page back exactly as it was found.

- Guest CLI workspace instances are agents only. Leftover managed-CLI
  bindings become agent rows; the extra table is gone.

- **New → Agent** in a workspace picks Pi or any installed CLI. Claude and
  the others are agents now; Agent CLIs stays the place to install and
  configure runtimes.

- An agent names which CLI it is (Pi by default). Play/managed start still
  only works for Pi; Claude and the others stay in the terminal for now.

- **`ask_human` over MCP waits for you.** The call now stays open until
  you answer in the Inbox (hours if needed), keeping the CLI's tool call
  alive with progress reports, instead of handing the wait back to the
  agent every 90 seconds.

- **Connectors get a Marketplace, like Packages.** The pane now has **Installed | Marketplace** tabs: search a curated catalog, pick **Save to**, press **Add** — the dialog is gone. PiCode's own connectors stay pinned at the top, and every agent CLI's pane works the same way.
- **The catalog goes beyond the hand-picked few.** Marketplace search covers a curated slice of the official MCP Registry (verified, remote servers), synced in the background and cached — it answers even when the registry is unreachable.

- **The computer tool acts only in the window the agent last saw.** A
  click, a keystroke or typed text is refused (`foreground_changed`) when
  the window in front is no longer the one the agent last captured or
  focused, so the agent looks again instead of typing into what you are
  using (ADR-0156, the first refinement of ADR-0148).

- **Connectors are now verified against the real vendor CLIs.** Every agent CLI's Connectors pane — Claude Code, Codex, Omp, Antigravity, OpenCode, Grok, Muse Code, Hermes Agent — was checked live against the installed binaries, and the differences found are fixed: what you write in PiCode is what the CLI loads.

- **Apps: web apps with their own window identity take the whole tab.** The title bar is gone — an installed web app that declares standalone mode fills its tab edge to edge; Ctrl+F finds in the page and the usual browser keys just work.

- **Apps: web apps with their own window identity open like apps.** An installed web app that declares standalone mode now fills its tab — no address bar, just a slim title with the app's name and a menu for browser actions.

- **Apps: installed web apps now install as themselves.** When a site carries a web app manifest, PiCode reads its name, icon, colors and start page — a bare address launches where the app says it should start, and the install dialog tells you it found one.

- **Agent CLIs: launch profiles open by default** once a CLI has at least one profile, instead of hiding behind a collapsed section.

- **Connectors foundation for every agent CLI (ADR-0150).** Connector capability is now a declared per-CLI driver registry, and `/api/mcp` rejects requests naming a CLI without a driver instead of silently writing Pi's files. Pi behavior is unchanged.

- **Browser tool: a screenshot reaches the agent as an image.** `browser
  screenshot` now hands the capture straight to the model instead of writing
  a PNG to a temp path and returning the path — one call to see the page, no
  `read` after it.

- **Muse Code and Antigravity launch through the PATH wrapper like every
  other CLI.** Same presence lease, same native runtime registration, no
  more launch carve-outs. Reporting is unchanged: Antigravity through
  its title reporter (now with a runtime behind it), Muse Code honestly
  Open (its hooks carry no terminal identity). Maintenance subcommands
  (`muse exec`, `agy update`, …) skip the lease.

- Muse Code lifecycle docs: R3233 grew real hooks, but they cannot feed
  per-terminal activity (see the plan note), so Muse stays honestly Open.

- A session note's `## Debts` bullets now say so at closing time: they ride the
  board for 7 days, so the durable ones name the topic file that owns them.

- **The handoff board never blocks a session again.** It is a bounded view
  (two next steps per topic, session notes for 7 days, open debts only) and
  warns instead of failing when it is over its target, so `make close` no
  longer stops on how much the team wrote down — and nobody prunes another
  session's bullets to make it fit.
- **Debts carry their state**: `- [x]` marks one paid in the topic file that
  owns it; the board shows the open count and keeps the record.

- The options menu opens over the page instead of pushing it down.

- The Activity seed now runs as v2 so hookless-except-by-reporter CLIs
  are picked up on existing installs; owner-switched-off stays off.

### Removed

- **The tray's disk line and Give back item.** The tray no longer shows the
  `WSL … GB · ≈… held by Windows · C: … free` line or **Give back ≈N GB…**.
  The disk facts and the give-back flow live in the Management window's **Disk**
  tab — the surface this change also makes work. Removed with them: the tray's
  disk poller, the Rust-side compact flow (the Go `disk-compact` command keeps
  the readiness interlock), and `desktop-shell/src/diskline.rs`.

- Drop the Chat/Terminal segmented control from the tab strip (desktop and browser) and the mobile agent header. Pi still switches from the sidebar / Work list agent menu (Open chat / Open terminal).

- **Connector file import and "use from another app" are gone.** The Pi adapter already imports, and each CLI's Connectors pane writes that CLI's own config directly — adding the same service in its marketplace replaces copying it from another app. **Custom server…** stays.

### Fixed

- **The "Reconnecting" screen no longer outlives a deploy.** The reload watch called its page-availability check without a default, so every automatic reload threw instead of firing, the health poll stopped rescheduling itself, and the app stayed on Reconnecting until a manual reload — in the browser, the desktop shell and mobile. The check now defaults to this page's own path, and a failing check can no longer end the watch.

- Explain Delivery integration and validation labels in the desktop and mobile views, and remove wording that implied an approval queue or publication status.
- Add a visual glossary to the Delivery guide and clarify the limits of observed evidence.

- **The ADR-0048 mutation gate no longer has the two blind spots it shipped
  with.** It read SQL as text, so a statement assembled from a package-level
  constant, or a table name its pattern could not spell, went unchecked — both
  were filed as a known limitation rather than fixed. A write is now either
  SQL the test can read *or* a call that runs one, which between them have no
  gap: every write in the store reaches SQLite through one `Exec` or the
  other. `VacuumInto` joins the listed exceptions.

- **The default shown was not always pi's default here.** Nine actions bind
  differently on Windows and WSL — `Ctrl+-` is `Alt+Z` on WSL, and `Suspend`
  has no binding at all on native Windows. The pane reads the host's platform
  and prints the binding that machine actually has, with the other platforms'
  rows on a muted line beneath.
- **The CLI pane tabs no longer run past the card.** At 1024px the nine-tab
  strip (Launch…Connectors) was painted over the page gutter with its last tab
  cut mid-word; it scrolls inside the card now, which is what the active-tab
  reveal was always written for.
- **A destructive button reads destructive on desktop.** `.btn-danger` only had
  a hover rule there, so every confirm drew Remove and Cancel as the same grey
  button; the phone app had the fill all along. It now has the rest state on
  both (`Reset all` is the first one you will notice).

- **Dragging a sidebar row no longer opens a horizontal scrollbar.** The
  reorder stays on the vertical axis. A sideways nudge used to slide the
  row out of the column, and the list grew a horizontal bar.

- **Renaming an agent left its editor tab on the old name.** A non-Pi agent
  runs as its bound terminal (ADR-0160), and the tab strip labels that
  `t:<id>` tab with the terminal's own name — a value copied from the agent
  once, at bind time. The rename route only patched the agent row, so the
  sidebar showed the new name while the tab kept the one from creation.
  `Store.UpdateAgent` now renames the bound terminal inside the same
  mutation and announces `terminal.updated`, so the strip, the dashboard
  fleet and every other `term.name` reader follow the rename; a same-name
  patch emits nothing.

- **Two gates added yesterday were passing work they claimed to check.** The
  ADR status check could not read `- **Status:** accepted` (the colon inside
  the bold), so ten of 171 decision records were silently unchecked; an
  unreadable status line is now a failure rather than a skip, and all 171 are
  covered. The ADR-0048 mutation gate scanned only each method's own body for
  SQL, so seventeen exported mutators that delegate their write — `AddAgent`,
  `CreateTerminal`, `EnablePeer`, `ReplaceFrom` and fourteen others — were
  never required to announce anything; three of them announce nothing and are
  now listed with a reason. It also read comments, so "we deliberately do not
  AppendEvent here" would have satisfied it.

- Git now uses one workspace selector above History and Delivery, and keeps the selected view when switching workspaces.

- **Providers**: **Sign in** reuses the CLI's sign-in terminal that is already waiting instead of stacking a new one per click, and sweeps the husk a dead session leaves behind.

- **The Management window answered "not allowed by ACL" on every tab.** Its
  seven commands — `disk_report`, `disk_compact`, `disk_compact_dry_run`,
  `clean_list`, `clean_apply`, `wslconfig_read`, `wslconfig_write` — sat on the
  ACL guard's exception list under the wrong assumption that no webview invokes
  them. The Management window is a served webview, so Tauri refused each call.
  They now live in a dedicated `management` capability
  (`desktop-shell/capabilities/management.json`), the exception list shrinks to
  the one genuinely tray-internal command (`computerlab_open`), and the guard
  test enforces the management commands too.
- **`clipboard_files` was missing from `build.rs`'s command manifest.** It was
  registered in `generate_handler!` and granted in `capabilities/default.json`
  but absent from `AppManifest::new().commands(&[…])` — the one-line gap the
  same guard test flags, and one a fresh permission regen would have surfaced
  as a dangling capability reference.

- **Providers**: **Check now** no longer treats the account already in the file as a finished sign-in. It stays on the strip and says so, until the CLI's file actually holds a different login.

- **Check for updates stays when a CLI has one plugin, or none.** The control
  lived inside the filter bar, which only renders for two or more installed
  plugins, so Claude Code (one plugin) and an empty list had no way to run the
  check the pane itself says is how Update appears.
- **A group count is a number, not a toolbar pill.** The header reused the
  filter's bordered count chip, so "59" read as a stray button next to
  "Ships with the CLI".

- **The API reference was missing every authentication route.** `/api/auth/session`,
  `/api/auth/sessions`, `/api/auth/logout`, `/api/auth/mode`,
  `/api/auth/pairings`, `/api/auth/token/rotate` and the device-revoke route
  are served by the daemon and were absent from the published OpenAPI
  document — the surface an API consumer needs first. The generator records
  routes against an empty dependency set, and auth registration returned
  early when the gate was absent, so nine patterns were never recorded. CI
  compared the committed file against that same generator, so it stayed
  green. The spec now lists 338 paths, and the generator's
  `x-undocumented` names the two non-JSON routes it still leaves out
  (`/pair`, `/preview/**`) instead of implying they do not exist.
- **Restoring a backup could destroy what it was restoring.** Pin
  attachments and pi session files were deleted first and copied second,
  with no rollback, after the database had already been swapped in — so a
  copy that failed partway (full disk, one unreadable file) left the live
  files gone and half rebuilt. Each directory is now built beside the live
  one and renamed into place, and the credential vault a restore replaces is
  kept as `credentials.json.replaced`.
- **Eight accepted decisions were still listed as proposals.** ADRs 0110,
  0120, 0135, 0142, 0150, 0153, 0155 and 0157 said `accepted` in their own
  file and `proposed` in the index, which reads as half the recent work being
  speculative.
- **`CHANGELOG.md` told agents to do what the commit hook refuses.** Its own
  header asked for an entry in `[Unreleased]`; ADR-0105 assembles this file
  from `docs/changelog.d/` fragments and the hook blocks the direct edit.
  The header now describes the fragment flow, and the hook allows a
  correction to the preamble above the first version heading.
- Three broken links in `docs/architecture/`, a subsystem file
  (`devservers.md`) that the architecture index never linked, a source
  comment pointing at a handoff topic that does not exist, and three forms
  that did not opt out of native browser validation.

- **Claude Code's installed plugins no longer disappear.** The pane reads the
  machine scope by leaving `scope` out of the query, and the roster read passed
  that empty scope through: Claude Code's rows name their scope (`user`), so
  every one of them was filtered away and the pane showed an empty list for a CLI
  that has plugins. The read now resolves the scope once — empty is the machine
  scope — for the list, the catalog and the update check, which is also the key
  they share in the cache. Regression test: `TestTheDefaultScopeReadsTheMachineScope`.

- **Providers**: the sign-in strip now carries **Open terminal** — the terminal the sign-in runs in is one click away, instead of the strip telling you to type `/login` somewhere it never named.

- **An empty Inbox names its next action.** With nothing waiting but items
  already answered, the blank slate was a dead end (“Nothing needs you
  right now.”) while the Done strip sat a tab away; it now carries the
  action — “See the done item” / “See N done items” — which lands on that
  strip. A mailbox with nothing at all keeps the one line, because there
  is genuinely nowhere to point.

- **Omp shows what its marketplaces offer.** Its plugin catalog was reported as
  unavailable; PiCode now reads `omp plugin discover` and lists the plugins a
  source offers, each marked installed when it already is. Omp's list does not
  say which source provides a plugin, so those rows are information and the
  list states the install form (`name@marketplace`) instead of offering a
  button that would run the wrong command.
- **OpenCode's plugin list says when a config is broken.** A `"plugin": "name"`
  string is refused by OpenCode itself ("Expected array | undefined"); PiCode
  used to list it as an installed plugin. The pane now names the file and the
  form OpenCode expects.

- **Attach now verifies the terminal sent the message.** The paste plus
  Enter raced the TUI render, so the text sat in the composer while the
  bar cleared. The paste path polls the live pane until the composer
  reads empty, retries Enter once, and answers 502 `staged` (bar stays
  open with the reason) instead of a blind success. TUIs without a reader
  keep the previous behavior.
- **The attach bar closes after sending** and hands the pane its keyboard
  back. Failures keep it open with the text staged, as before.

- **The Inbox no longer promises a pickup nobody made.** Answering a
  question whose source has no reply channel records the answer on the
  item (so the human's Reply closes it), and the note said "the CLI that
  asked will pick it up here" — true for `picode inbox ask --wait`, false
  for a plain `ask` and for a pi launched outside the launcher, which
  read the durable queue instead. The note and the toast now name both
  paths: a waiting asker reads it here, a non-polling asker must be told
  another way.

- **Providers**: the pane's action bar lays its controls out as one group at the right edge — the account count on the left, **Sign in** and the single **Add API key** beside each other instead of a button floating mid-row, and a multi-provider pane no longer shows its own Add directly above the first provider's.

- **Desktop/Web:** removing the agent whose tab you are on no longer lands on a
  dead surface. Selection now moves to the neighbouring tab (right, else left),
  and to the dashboard when no tabs remain. Closing a tab picks the same
  neighbour instead of jumping to the last tab. A CLI agent's bound terminal
  tab and a removed workspace's tabs follow the same rule.

- **Muse Code's plugin catalog is actionable.** A catalog row now carries the
  spec the CLI installs from (`name@marketplace`) and the path the marketplace
  resolved, so **Install** in the Marketplace tab runs the right command.
- **Marketplace rows say when a plugin is already installed.** Muse's own
  catalog keeps answering `available` after an install, so PiCode joins the
  catalog with the CLI's own list and shows **Installed** instead of an Install
  button for something that is there.

- Context menus that open over the tab strip no longer paint behind it.

- **Muse Code's own plugins are listed.** `muse plugins list` nests each
  plugin under `record` and `plugin`, which PiCode read as a list with no
  names — so a machine that has Muse plugins saw an error where the list
  belongs. Install, remove and enable/disable already worked and are unchanged.
- **The Muse plugin surface is per machine, and the pane says so.** Muse turns
  its plugin commands off through its own cached feature configuration; when
  that is the case the CLI answers *"plugins are not available in this build"*
  and PiCode shows the CLI's own sentence instead of an empty list.

- Answering a question filed by a CLI with no PiCode identity (a guest
  running outside PiCode) now records the answer on the item, where the
  asking CLI picks it up. The Inbox used to refuse with "no reply channel"
  and the item stayed open forever, leaving the asker waiting on a poll
  that could never end.

- **Ctrl+V / Ctrl+Shift+V now attaches images in terminal panes.**
  The keydown used to kill the browser's own paste (stopping xterm's ^V)
  and then read text only, so an image clipboard pasted nothing at all.
  It now re-reads the clipboard (files included) and fires an equivalent
  paste, routing through the same attach bar the native menu already
  opened. Text-only pastes behave exactly as before.
- **The PiCode menu's Paste row stages files.** It read text only and
  dropped images silently; with files on the clipboard it now opens the
  attach bar, like the keyboard does.

- Fix the work-browser split ("Open browser" beside an agent/terminal) in the
  `/browser/` web app: the tab strip spans the top and the agent and browser
  panes now share the row below, so the terminal no longer crushes to a sliver
  and the browser pane no longer pushes off-screen. Desktop behavior is
  unchanged.

- Settings ▸ Browser/Computer audits: a refused call reads neutral (the
  policy working as intended) and only failures read danger; the outcome
  filter lists every outcome present in the data.

- **Terminal paste now answers the same door as the Attach menu.**
  An interactive agent pane whose bound record carries no launch fields
  no longer refuses a files-paste the menu would accept.
- **A double-paste stages both pastes.** Rapid Ctrl+V,V used to keep only
  the second files; concurrent stages now append functionally under the
  4-file cap.
- **Text pasted alongside files seeds the message.** It used to be dropped
  (or land stray in the terminal when clipboard-read was granted); the
  keydown path's late text is now claimed away for that gesture.

- Settings ▸ Browser and Settings ▸ Computer now fill the full page card
  instead of a narrow 780px column, and all row controls share the 36px
  control height.
- Agent permission lists carry workspace provenance on every row (workspace
  chip resolved via `/api/workspaces`, plus managed/terminal kind), with a
  search + filter toolbar and paged rendering instead of one unbounded list.
- Recent steps (Computer) and Raw calls (Browser) are filterable by search
  and outcome, with expandable rows showing the full reason, actor and
  timestamp instead of a truncated single line.

- Codex's Memory pane no longer offers to copy `codex` as "the vendor's own
  command" — that command does nothing to a memory, and the pane now shows the
  note alone.

- **Mobile Inspector:** Git actions now address the checkout under review
  when following a sibling worktree (branch and upstream come from the
  followed checkout, not the anchor), and the agent screen's review glance
  clears instead of freezing when the agent's folder moves.

- **Agent CLIs: Omp terminals now report Needs you.** The omp reporter only
  knew pi's event set, and omp never fires those — so an approval or a
  question waited in the terminal while the row said Working. The extension
  now listens to omp's own events (`tool_approval_requested`/`resolved`
  and the ask card's `tool_execution_start`/`tool_result`, verified against
  the 18.2.6 bundle), files the same Inbox item the other CLIs file, and
  only settles on `agent_end` when the run is really over (`willContinue`
  marks a mid-run turn).

- Verify on a saved key spends exactly one listing call to the provider, named
  on the button, and stores the answer with its age on the account's row.
  Nothing else in the app talks to a provider about credentials.

- **Hermes' Show reasoning row said Off when Hermes turns it on.** Every
  boolean now declares what its CLI does when the key is absent, checked
  against a recorded table, so a row left alone shows what will actually
  happen. Hermes' turn limit says 20 rather than a vague "default".

- **A terminal no longer dies the moment it opens on a service without a
  `SHELL` setting.** The daemon fell back to `/bin/sh` — dash on
  Debian-family systems — and handed it a bash-only argument, so the pane
  exited before the first prompt and took its tmux server with it while the
  app still showed the terminal as running. The fallback is bash now, and
  the argument goes only to a shell that accepts it.

- Opening a terminal whose shell exits immediately reports the failure and
  keeps nothing behind, instead of leaving a terminal that reads as open
  with no session to attach to.

- A boolean row in Pi's Settings now uses the same switch as the rest of the
  pane instead of a second control for the same job.

- Inspector Git menu hints now carry the exact prepared command as a hover
  title, including the real branch on `git push -u origin`.
- The session Changes cap note reads "+N more copies of the project not
  scanned" instead of jargon, with the linked-checkout explanation on
  hover.

- Use the Agent CLIs terminal screen for mobile TUI agents, including touch scrolling, menus, attachments, keyboard controls and recovery states; show Pi's Chat/Terminal switch as toolbar icons.
- Share terminal input and connection wiring across browser, desktop and mobile; preserve live legacy Pi addressing without retaining a second terminal renderer.
- Keep bound terminal identity and actions consistent across agent views, terminal links and Canvas; open non-Pi agents in their TUI and keep Pi's view switch in the existing tab toolbar.

- **A guest CLI's setting could be written to the wrong key.** In YAML, a path
  like `memory.memory_enabled` matched a block of the same name nested anywhere
  else, so the save landed on it and the key the pane showed never changed. The
  locator now anchors the first segment at the document's top level and keeps
  every next one inside the block its parent opened.
- **TOML writes could land inside a multi-line string.** A `"""…"""` body
  containing `key = value` or `[table]` lines was scanned as configuration,
  which rewrote the user's prose and could silently retarget a save. The
  scanner now tracks string state, and an array of tables is refused rather
  than edited.
- **JSON writes could shadow a value.** A key present twice was written on the
  copy every parser ignores, and a path whose parent was a string, an array or
  null appended a duplicate member at the root. Both are refused with a
  sentence naming the file.
- **Handing a key back could delete more than the key.** Emptying a YAML block
  swallowed the comments and blank lines that followed it, which emptied a file
  whose block was followed by commented-out configuration.
- A file with no final newline, and a JSONC file with a comment before its
  closing brace, can now take a new key instead of refusing every save.
- **Claude Code's Approvals row could not show the value in the file.** The
  options were missing `auto` and `dontAsk`, so a machine set to `auto` drew a
  blank control and any choice moved it off. OpenCode's `autoupdate` row is
  withdrawn: it is `true`, `false` or `notify`, and a switch would destroy the
  third. Grok's options now match what that CLI accepts.

- Pi agents now open the same contextual Launch settings editor as other Agent CLIs. Model, reasoning, tools and checklist remain under Settings; bound agents keep their CLI fixed and Pi launch controls omit reserved model and reasoning flags.

- Keep a one-time browser permission answer from becoming a remembered site permission.

- **Session Changes no longer corrupts fleet pills.** Linked-worktree
  watcher events are path-only, so the workspace and agent pills keep
  describing the anchor folder while followed groups still update live.
- A file tab whose worktree checkout is gone now reads "That file is
  gone." instead of the raw server message.

- The tmux app's server view no longer shows an empty version and keyboard
  mode while a drained session is answered by the older server.

- Pi agents in terminal mode now reuse the shared CLI launcher and activity integration, including working, needs-you and idle states across desktop and mobile. Existing open Pi terminals remain usable until explicitly restarted.
- Pi terminal and chat transitions now share lifecycle guards, preserve agent session ownership and refuse replacement while a previous process is still closing. Agent launch settings remain available alongside Pi-specific configuration.

- Refresh the Windows desktop's native chrome region when its viewport or display scale changes, so maximizing or restoring a window without an active work-browser page does not leave the interface clipped to its previous size.

- Keep desktop work pages visible and live beneath address suggestions, menus and dialogs in the updated Windows shell. Native regions preserve the existing HTML controls without capturing the page; older shells retain their previous behavior until upgraded.

- Agent CLIs **Update** on an npm-global Pi runs `npm install -g` instead of
  `pi update`, which refuses when the package dir is not writable (a sudo
  npm install into an nvm prefix) and leaves the button failing.

- Bind newly created interactive Pi sessions as soon as their JSONL file appears,
  so the agent menu exposes `Continue in…` without a manual Sessions refresh.
- Allow a bounded Muse index-update window when pinning a terminal's latest
  session, reducing the shutdown race before the index is refreshed.

- The Windows installer no longer hides a failure: the launcher waits for the
  elevated run and reports its exit code, the last error line is written to
  `%ProgramData%\PiCode Desktop\install.log`, and the elevated window pauses
  for Enter instead of closing with the evidence.
- `install --user <name>` now aims the picode binary and pi at that account
  (and `--user root` keeps the system-wide install); stage scripts no longer
  rely on `$` surviving the `wsl.exe` argument boundary.

- Removing an agent no longer leaves the row behind when the delete
  response is lost: "not found" is treated as removed and the fleet is
  refetched, so a second remove cannot fail with "agent not found".

- Use consistent Start, Restart and Stop agent actions across workspace agent menus, with shared ordering, capability-specific options and separated removal.
- Offer scoped launch settings and confirmed restart for Pi agents; restarting preserves the managed or interactive mode and reports failures without claiming success.

- A Send with three or more annotations no longer fails: it used to hit the
  prompt door's four-file limit and deliver nothing while keeping the notes in
  the store. A Send that cannot be staged whole now stages nothing at all
  instead of half a package.

- The stale `btab_layer` permission artifact is gone; the generated schemas no
  longer advertise a command that does not exist.

- The annotation card's "Background" row no longer truncates its label: the
  label column fits every row's name at the card's own width.

- The annotation screenshot shows the page, not our own annotation UI: the
  pin, the chip and the card are hidden for the instant of the capture, so
  the agent receives the element instead of a picture of the card covering
  it.

- A missing annotation screenshot now says why instead of vanishing: the
  Send names the step that gave up (and the shell's own words when the page
  capture is what failed), and the capture retries on a short ladder so a
  moment of hidden page does not cost the picture.

- Annotation screenshots actually arrive: the crop asked the shell for the
  page preview with the React tab id, which the shell could not resolve, so
  every Send staged the note alone and the picture failed in silence. The
  shell now resolves either id shape and the call site passes the native one.

- **Mobile tab bar docks to the bottom of the screen.** On an iPhone
  home-screen install the bar grows into the system letterbox instead of
  leaving the tabs floating above a blank strip. Pick **Low** in Layout
  if a label is clipped.

- **Mobile terminal uses the bottom of the screen.** On an iPhone home-screen
  install, a pushed screen (terminal, agent) grows into the system letterbox
  instead of leaving a blank band under the TUI.

- The phone Work list ⋯ on a terminal is no longer only Remove: Rename,
  Continue in…, Start or Restart/Stop, then Remove — the same menu as
  desktop, without Launch and Terminal settings (those stay in Agent CLIs
  and the open pane).

- **Mobile header is no longer frosted.** The iOS home-screen app uses an
  opaque status bar, so iOS 26 Liquid Glass does not wash out the title
  and icons at the top of the screen.

- **Restart terminal** on an Agent CLI now reopens the conversation that
  was running (the pinned session's verified resume arguments), instead of
  starting a blank chat. Stop then Start, and Resume last session on a
  stopped terminal after a crash, stay as they were.

- Connector gallery refreshes no longer race over the same cache file: two
  at once (the background timer and a manual refresh) could make one fail
  with "no such file or directory" and the gallery answer an error. The
  cache write is serialized and uses a private temp file.

- Annotations reach the strip even when the page cannot post to the app:
  the annotate strip now asks the host for the page's state on a slow tick
  (a path that needs no page-side bridge), so the count and Send light up
  and the batch carries the real payloads.
- The work-browser page is created with web messages enabled, instead of
  enabling them only when annotate mode is armed — which left the already
  loaded document without the channel.

- Annotate mode survives a full page load: the shell re-injects the in-page
  script when a navigation completes, so the overlay no longer vanishes
  while the strip still says it is armed.

- Annotation Send lights up again: the page's message channel is
  re-subscribed on every arm (a webview recreated under the same tab id
  used to inherit a stale subscription and go silent), the receiver is
  dropped when its tab closes, and a page that does not answer is retried
  once and then named out loud instead of leaving a dead Send.
- Re-arming annotate mode keeps the pins on the page and re-reports them.

- Typing in the annotation card no longer triggers the page's own keyboard
  shortcuts: keystrokes are shielded at the card's shadow root, so a site
  hotkey (GitHub's "s" opened its search) can no longer steal focus and
  swallow the character mid-word.
- Enter saves the note again — it had never worked, for the same reason
  (the event target outside a shadow root is the host, not the input).

- Annotating no longer loses the draft: while a card is open, page clicks,
  pins, chips and menus do not steal it onto another pin — Save, Cancel,
  Esc or trash first, then pin the next one.

- Annotation Send delivers to the bound session: a browser pane open on an
  agent or terminal tab sends to that session's chat (agent prompt door for
  agent panes, terminal prompt door for terminal panes). A bound miss never
  reroutes to a stranger's terminal — it names the reason and keeps the
  notes. Standalone tabs keep the first-running fallback.

- **`computer` typing under load.** Characters the keyboard layout has are
  now typed as keystrokes (with Shift when needed), which carry their own
  character and survive a busy app; only characters the layout lacks, AltGr
  characters and dead keys still travel as Unicode packets. Pacing alone
  (the earlier fix) still garbled text when the app fell behind.

- Annotation Send delivers again: it posts to the prompt door
  (`{message, paths}` pasted into the agent TUI), not the file-drop door —
  which answered 400 while the toast blamed the terminal. A refused paste
  now names the server's reason.

- Annotations reach the strip again: the page posts message objects (the
  host serializes once) and the chrome unwraps at any encoding depth, so a
  saved note always shows up as Send N instead of a silent Send 0.

- **Apps: Clear data now waits its turn.** Clearing right after using an app could fail silently ("Could not save the web app") because the app's files were still being released. The clear now waits up to ten seconds and, if anything still blocks it, says exactly what.

- **`computer` typing came out as one repeated character.** The desktop
  app fired every Unicode key event back to back, and Windows 11 Notepad
  read each one as the last character of the text. The app now paces
  characters 5 ms apart and types at most 2 000 characters per call, so
  `type` writes what the agent sent.

- Annotate hover no longer starts dead: the highlight shows from the first
  mouse move and stops while a card is open (open-state travels in flags,
  never in inline styles). First-press Esc exits the mode again.

- **Claude Code connectors now save correctly from the pane.** The add command PiCode runs now matches what `claude mcp` accepts, and rows read from `claude mcp list` keep their address and kind.
- **Codex and Antigravity save to the one config file each CLI actually reads.** The "This folder" target refused instead of writing a file the CLI never loads.
- **Grok connectors can be turned on and off again.** The switch mirrors Grok's own enable/disable, and removing a connector no longer leaves stale entries behind.
- **OpenCode connectors land in the file OpenCode itself uses** (`opencode.jsonc` when present), so edits are never shadowed.
- **Muse Code connectors no longer produce a settings file Muse rejects** (the required `schema_version` stays present), and Grok rows report their enabled/disabled state.

- Annotate mode no longer dies on the ACL (the command was missing from the
  shell manifest) or on Send (the page URL ref did not exist).
- The fallback preview no longer blanks the app when a second note is
  pinned (the event was read after React released it).

- A malformed connector config file (JSON, TOML or YAML) no longer fails
  the Connectors pane with an error — the pane names the file and carries on.

- **desktop: deploys and shell swaps no longer end every terminal.** The
  WSL distro's keepalive moved out of the shell process into a scheduled
  task (`PiCodeDistro`) that outlives it (ADR-0155): a `make
  desktop-restart`, a shell crash or any taskkill can no longer leave the
  distro unowned for WSL's idle reclaim — the failure that killed every
  tmux session, agent and terminal at once (2026-09-18, four times). The
  task's action wraps wsl.exe in a headless conhost, so the keepalive
  holds the distro with no console window on the desktop. The shell
  ensures the task on its health poll (child-spawn kept as fallback), and
  `desktop-swap.sh` ensures it before killing the old resident. Also
  fixed: `make adr` failed whenever a registered worktree's directory was
  pruned.

- **Desktop: the shell no longer bricks on a daemon restart.** Back-to-back deploys could leave the desktop app parked on a 404 body forever. The shell now waits for the app to really answer before first paint, and an automatic reload waits for the page to serve again (it never reloads into a dead body).

- **Agent CLIs: no more one-request "Not found" flicker while a CLI updates itself.** A vendor updater rewrites its launcher in place; the catalog now retries once before reporting a CLI missing, so lifecycle menus stop flickering off during updates.

- **Windows shell caption buttons and native commands work again.** Covering
  `/desktop/` with the app CSP had blocked Tauri's IPC (`http://ipc.localhost`),
  so every `invoke()` died in the console. The desktop shell's policy now
  names that host; the browser and mobile shells do not.

- `claude mcp list` output with no servers no longer fabricates a phantom "No" connector row, and the Claude Code driver id now matches the CLI catalog (`claude-code`), so `#/clis/claude-code/connectors` highlights the right roster entry.

- **A stopped server no longer keeps its row.** The panel's cached probe answer
  was trusted for two minutes regardless of whether anything still listened, so
  a server that had just stopped stayed on screen (found in the first QA pass of
  the rework, minutes after Stop said it was gone). A row now needs the port to
  be listening at this read, unless a PiCode pane still owns the socket.
- **"unnamed page" beside "not a page".** A port that is not a page and has no
  title shows its port alone instead of borrowing the page vocabulary.

- **Dashboard loads faster.** The six CLI meters now aggregate in parallel (cold 7d window drops from ~8s to ~5s on this machine), the server pre-warms the stats cache after boot, and the dashboard paints the last good numbers instantly while refreshing underneath instead of a full skeleton.

- Desktop: editor tabs can be reordered again — the shell now disables Tauri's native drag-drop handler, which on Windows swallowed the HTML5 drag events the tabs (and composer file drop) rely on.

- **Agent CLIs: an Omp terminal no longer stays Working after it replies.**
  omp never fires the events pi uses to settle a run, so the activity
  reporter waited forever at Working. omp now uses its own event set —
  Ready comes back when the reply finishes — and, measured against its
  approval dialog, it never claims Needs you: approvals happen in the omp
  terminal itself.

- **Clicking “PiCode” opens the dashboard again in the desktop app.** The
  wordmark in the window's top row was a window-drag region as well as a
  button, so pressing it dragged the window and the click never arrived;
  the sidebar wordmark also did nothing while a non-workspace page (Agent
  CLIs, Browser, Preferences, Devices) was open. Both now open the
  dashboard, and the top row stays draggable around the button.

- **The browser menu no longer blinks the page.** Opening the ⋮ menu while a
  site is on screen used to park the live page behind a strip of gray for the
  length of a screen capture; the menu now waits for its backdrop (the frozen
  frame it sits on) and opens with it — the page never disappears.

- **Opening a link no longer flashes a stray page.** A browser tab created
  before its pane reported its size painted at a placeholder rectangle over
  the pane's own empty state until the real bounds arrived. Such a tab is now
  born hidden and appears in place — the moment its rect is known.

- **Ctrl+click on a link in an agent's terminal opens it in PiCode**, not in
  the system browser: a printed URL now follows the same preferences as every
  other door (Browser settings — local dev sites and web URLs, both defaulting
  to PiCode's own browser). Set either to "Default browser" to send those links
  out, and paths under the terminal's folder still open in the file pane.

- **Resetting a site's permission now really forgets it.** The shell kept
  every decision in the engine's own per-origin memory (`SetPermissionState`)
  as well as in its map, so "Reset" in Site settings left the site working
  until the app restarted. A policy write that names a site now tells the
  engine too, and "Always allow" from the Ask bar writes the site's standing
  on both sides.

- **A browser grant with domains reads properly** (Settings ▸ Browser ▸ Agent
  permissions): the row stacks — name and what the domains mean, then the
  tier and the domain field on their own line — instead of squeezing the
  field to a stub with the select wrapped above it, and the field matches the
  height of the controls beside it.

- **The browser tool now shows what the act verbs answer.** `evaluate` returns
  the value (or `the page threw: …`), `navigate` says where it went, `cdp`
  returns the method's JSON. All three used to be rendered as an accessibility
  tree, so a call that worked arrived as "the page has no accessible content".

- **Message delivery keeps up with vendor UI changes.** Grok 1.0.34
  restyled its composer footer and Claude Code predicts dim follow-up
  suggestions — both silently stalled automatic prompts. Both shapes are
  now recognized, with the same strict frame, cursor and footer checks.

- Plan file lists no longer repeat the wrapper entry.

- **Sidebar scrollbar alignment and track.** The sidebar's `.side-scroll`
  scroller previously lived inside `.side-section` with an 8px horizontal
  padding, causing the scrollbar to float 8px away from the right border and
  look disconnected when scrolling down past the pinned header.
  The container now spans full width with `padding: 0 8px 10px` on
  `.side-scroll`, placing the scrollbar flush against the sidebar's right
  edge while maintaining consistent 8px breathing room for item rows and headers.

- **Activity reports without a native runtime no longer 409.** Session
  reports (which carry a session id) used to require a live native
  runtime, so observers that never start one — the Antigravity title
  reporter today — had every report refused and their terminals read
  Open forever. With no runtime entry there is no identity fence to
  protect, so those reports now land as plain terminal states; terminals
  with a live runtime keep the strict conflict.
- **The server test suite no longer writes into your real home.** Booting
  a server seeds Activity on and syncs integration files, and the two
  CLIs installed through your own settings (Antigravity, Muse) resolved
  the developer's real home in tests that never isolated it. The whole
  suite now runs under a throwaway HOME (plus XDG), guarded by a test
  that fails if the sandbox ever goes missing.

- **Menus and other layers no longer come up hidden under the work browser.**
  The editor's tab menus, the command palette, dialogs, dropdowns and toasts
  now decide by geometry: any floating layer that reaches the page parks the
  native view and freezes the page behind it. Previously only dialogs did
  (and only the ones wearing the app's own dialog class), so the palette and
  every chrome menu could appear behind the page.

- **The domains field says what it means.** Each entry is described as you
  type it — including the shapes that open nothing (`*example.com`, `*.`, a
  lone `.`), which used to be accepted and saved in silence.

- **The JavaScript switch reaches the shell on load**, not only when it is
  clicked: after an app restart the shell's copy of the setting is now the
  saved one, not its default.

- **The desktop app could not be opened from the tray.** With the resident
  started by the logon task, *Open PiCode* (and a second launch) did
  nothing: the window was alive but the app's own lookup missed it, so
  every open path quietly no-op'd. The shell now keeps its window handle
  and asks Windows directly to restore and focus it.

- **Lifecycle buttons on mobile (and Muse Code on desktop).** The Install
  and Update buttons were gated on the integration capability — inverted
  on mobile (`!cap.integration`) — so update-capable rows without it
  never showed Update. Both buttons now follow only the lifecycle flags,
  like the `···` menu already did.

- **Work-browser options menu in the agent split.** The menu's settings
  links (Browser settings, History, Downloads, Clear browsing data,
  Passwords) did nothing in the split pane — that surface never received
  the navigation callback. They now land on Settings ▸ Browser, like the
  tab version does.
- **Menu still.** An empty page capture used to hide the live page behind
  nothing (uniform gray). The tab now only hides the page behind a capture
  that decoded to real pixels; a failed capture keeps the menu usable over
  the host background.
- **Settings views under a live page.** Opening any settings view with a
  work tab selected left the page painted over it. The tab is now parked
  while the route is elsewhere and restored on return.

- Application dialogs (New workspace, confirmations, the Settings dialogs)
  are no longer covered by the work browser's page.

- **Sign-in popups open as real windows.** “Continue with Google” (and the
  other OAuth providers that open a sized popup) now completes instead of
  dead-ending: `window.open` with a size keeps its link to the page that
  opened it, which is how the credential comes back. Plain `target=_blank`
  and unsized `window.open` still open as editor tabs.

- Choosing “Platform default” in Site settings now clears the per-kind
  policy; the choice failed before.
- The Manage dialogs (Site settings, passwords, and the rest) scroll inside a
  short window instead of running past its bottom edge, where Close was
  unreachable.
- A work-browser tab no longer re-arms its page-metadata poll on every
  render, which pinned a CPU core while the tab was open.

### Security

- **Use** is refused while a terminal of that CLI is running: replacing a
  credential under a running agent can corrupt its session. The refusal names
  how many terminals to close.
- A CLI whose login PiCode cannot write faithfully says so and offers no
  control: Omp (its own database), a Grok file with no session yet, and the
  API-key paths of Hermes and Muse.

- Credential values never leave the server: the roster shows a masked hint
  (`sk-ant-…f2a`) at most, and a vault that cannot be read (missing key,
  tampering) is reported in words instead of being silently replaced.

- **A memory could be written outside its folder.** Containment resolved the
  target file, which does not exist when one is being created, so a write
  through a symlinked subdirectory landed outside. It now resolves the deepest
  existing ancestor, and listing skips anything that is not an ordinary file —
  a symlink named `notes.md` had its first line published, and a FIFO hung the
  pane.
- A relocated memory folder is bounded to the user's home directory, so the
  pane cannot be pointed at `/etc` or `/proc` from the Settings pane.
- Masking covers more credential shapes (cloud secrets, `password=` and
  `api_key:` assignments, passwords in URLs, Slack webhooks, PGP and truncated
  key blocks) and no longer destroys ordinary words: `risk-assessment-…` was
  being redacted.
- Reading a memory refuses anything that is not an ordinary file, so a
  pseudo-file reporting size 0 no longer walks past the size limit.

- Memory files are masked on read for credential-shaped values (provider keys,
  tokens, private-key blocks). The file keeps what the CLI wrote; the browser
  does not echo a secret back, and memory text never reaches the change feed.

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

[Unreleased]: https://github.com/cfpperche/picode/compare/v0.4.0...HEAD
[0.4.0]: https://github.com/cfpperche/picode/compare/v0.3.1...v0.4.0
[0.3.1]: https://github.com/cfpperche/picode/compare/v0.3.0...v0.3.1
[0.3.0]: https://github.com/cfpperche/picode/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/cfpperche/picode/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/cfpperche/picode/releases/tag/v0.1.0
