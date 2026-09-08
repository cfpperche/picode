# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

**Agent contract:** every commit with a user-visible change MUST add an entry
to the `[Unreleased]` section. The repository's official language is English
(see `AGENTS.md`); changelog entries included.

## [Unreleased]

### Added

- **Find inside a terminal** (`Ctrl+Shift+F`, or **Find…** in the pane's
  right-click menu): a field floats over the terminal — searching never
  resizes the pane — highlights every match, counts them (`3/14`), and walks
  them with Enter and Shift+Enter; Escape closes it and gives the keyboard
  back to the terminal. The plain `Ctrl+F` still belongs to whatever runs in
  the pane (`less`, `vim`, readline), and the chord is rebindable in
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

### Changed

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

### Fixed

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

### Added

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

### Changed

- **Mobile v2:** compact conversation and terminal controls, searchable Work
  views and grouped tools give more space to active work. Each agent keeps
  its unsent draft and attachments while navigating; message options open
  in a sheet and Enter adds a new line. Failed sends offer Retry.

- **Pi terminals no longer show a checklist strip above the pane.** The
  TUI already draws the plan; the sidebar card still carries the one-line
  step. The pane is just the terminal.

### Fixed

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

### Added

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

### Fixed

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

[Unreleased]: https://github.com/cfpperche/picode/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/cfpperche/picode/releases/tag/v0.1.0
