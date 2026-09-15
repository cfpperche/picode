# CLI terminal launch settings (ADR-0069)

> Part of [PiCode's architecture](../architecture.md) (ADR-0105: one file per subsystem). Edit here; the index only links.

`internal/clilaunch` holds a small catalog and configuration resolution; it
does not implement an agent runtime. SQLite tables `cli_configs` and
`terminal_launches` store defaults and terminal overrides. First boot imports
catalog rows: Activity reporting defaults **on** unless `enabled.json`
explicitly lists the CLI as false. A missing key is not false — that freeze
left OpenCode unwired on its first deploy. `SeedCatalogIntegrationDefaults`
flips empty default-off rows once. Stored settings always win after that.
The legacy wiring API updates the same store. Mutators commit `cli.updated`
or `terminal.launch` invalidation events in their own transaction.

The launcher resolves process environment → CLI defaults → terminal overrides.
Environment keys merge (a null override removes a default), argument/PATH arrays
replace defaults, and an explicit empty array clears them. PATH entries prepend
the service's inherited PATH. PiCode correlation variables, HOME, SHELL,
GROK_HOME, HERMES_HOME and OPENCODE_CONFIG cannot be overridden through the environment field. An executable
may be a command name or absolute path; resolution skips PiCode's own wrappers.
Argument and environment values are individually shell-quoted, never evaluated.

Each launch writes a private generation under `cli-launch/<terminal>/run-*`.
Integration uses the existing CLI adapter with the resolved executable pinned
in that generation. Shared executable reporters are replaced atomically.
Saving configuration never modifies a live generation. The last-applied
snapshot records CLI identity, executable, fingerprint, redacted common secret
arguments and environment key names, not environment values. Terminal events
carry that diagnostic view or an ID, never the full environment configuration.
Pending changes compare effective settings and CLI identity, not activity.
Old generations are reclaimed after a successful new launch; Remove deletes
only that terminal's private launch files, not native CLI data.

`GET /api/clis` resolves installation without starting a conversation;
`POST /api/clis/<cli>/check` explicitly runs bounded `--version` and checks
reporter prerequisites. It does not certify authentication or every hook.
`GET /api/clis/<cli>/sessions?cwd=` lists a CLI's on-disk sessions read-only
(ADR-0079 phase 2; `internal/clisession`): pi reads its JSONL root, Claude
Code its `~/.claude/projects` transcripts, Codex its `~/.codex/sessions`
rollouts, Grok its `~/.grok/sessions` prompt history, Hermes Agent its
`~/.hermes/state.db` (or `$HERMES_HOME/state.db`) SQLite rows, and OpenCode
its `~/.local/share/opencode/opencode.db` (or `$XDG_DATA_HOME/opencode/opencode.db`)
SQLite rows, each parsed defensively (malformed files are skipped, missing
roots are an empty list). Hermes listing is the active home only (no
`profiles/` scan), `source` cli/tui with a folder and at least one message;
preview is the session title. OpenCode listing skips child sessions
(`parent_id`), archived rows and empty transcripts; timestamps are
milliseconds; preview is the session title. The OpenCode CLI's own
`session list` is project-scoped and is not used.
Rows carry the server-verified resume arguments for that CLI (verified
2026-09 against each CLI's `--help`: `claude --resume <id>`, `codex resume
<id>` positional, `grok --resume <id>`, `hermes --resume <id>` on Hermes
Agent v0.18.2, `opencode --session <id>` on OpenCode 1.18.29); pi's row carries none — pi resumes
through its own chat flow. Cost is pi-only on this surface: guest formats
are not shown with per-session spend. Non-Pi sessions open through
`POST /api/clis/<cli>/terminals` with the resume arguments as launch
argument overrides — no transcript replay, no writes, no deletes for other
CLIs' sessions. pi's management surface (delete, auto-clean, resume,
in-use guards) stays on its own endpoints.
`POST /api/clis/<cli>/terminals` saves and opens a configured terminal.
`/api/terminals/<id>/launch` reads/writes overrides; its `start`, `stop`,
`restart` and `remove` POST routes serialize per terminal. Start is idempotent
while live, Stop retains configuration, and active destructive actions require
confirmation. Restart validates the next executable and directory before
ending the current session. A CLI exit returns to an interactive shell;
browser/daemon reconnect only reconciles, never restarts work automatically.

Manual CLI commands in ordinary terminals retain session-local wrapper
instrumentation. Launch defaults apply to the central manager, not to commands
typed in a shell. Configured CLI, observed presence, observed activity and
installation/setup checks are separate facts in the UI.

ADR-0070 adds read-only launch inspection and copied profiles. The backend
`launchPlan` resolver supplies both previews and launch preparation; shared
`cliIntegrationPlan` argument vectors drive the actual adapter writers and
their inspectable branches. Preview never runs an executable or writes files.
It uses symbolic generation paths and explicitly labels Codex's runtime
capability choice and Pi's maintenance bypass. The actual process environment
inherits the daemon environment, with settings and correlation keys layered
on top. Environment values are absent from preview/terminal events.

`cli_profiles` stores named full configurations; applying one copies explicit
overrides, not a live foreign-key relationship. Updates/removal cannot modify
terminals already created from it. `cli.profile` events carry IDs only.
`cli_checks` persists bounded checks (`cli.checked` events); executable path,
size/mtime and configuration fingerprint determine staleness. Check runs
`--version` and tests prerequisites, while the separate repair route regenerates
only private integration files. Observed activity remains ephemeral and is never
inferred from a prepared file or a version check.

ADR-0087 adds a lifecycle layer in `internal/clilifecycle` and
`internal/clijob`. Detection classifies the resolved executable's realpath
(npm, native, vendor, git, unknown) and a per-CLI plan pins the exact argv:
checks read the npm registry (reusing `pipkg`) or the vendor's own `--check`
command, and mutations run the vendors' update/reinstall/uninstall commands —
npm only where the vendor has no command. Update facts (`Latest`,
`UpdateAvailable`, `UpdateCheckedAt`, `InstallMethod`) live in the
`cli_checks` diagnostic; `POST /api/clis/<cli>/update-check` refreshes them
on demand and the surface triggers it once when the stored check is older
than six hours — there is no browser polling timer. Mutations are durable
jobs in `cli_jobs` (`cli.job` events): HTTP 202, request-key idempotency, at
most one active lifecycle job, revision-guarded transitions, bounded output
tail. Restart recovery marks queued/running jobs `interrupted` and never
replays an install; a graceful shutdown cancels the command and marks the job
`interrupted` too. A job refuses while live terminals of that CLI run unless
the caller confirms; uninstall additionally requires typing the CLI name.
`unknown` install methods get no mutation controls — only the docs link.
After a succeeded job the setup check re-runs and update facts reset so no
stale badge survives.

ADR-0093 closes the cycle for missing CLIs: `install` is offered only when
the executable is absent, npm-backed for pi/codex/claude-code (the same argv
as reinstall) through the same job lane, and guided (docs link, no
executable action) for grok/hermes whose installers are vendor curl scripts.
Installing an installed CLI is refused — reinstall covers it. The update
check line no longer reports "failed" for unmanaged installs; the real check
error text is shown for genuine failures.

**Catalog capabilities** (`surface`): Muse Code (`muse`) and Antigravity
(`agy`) carry `surface: terminal` — PATH detection, `POST …/check`
(`--version`), `POST …/terminals` (New terminal runs the CLI with its own
defaults) and `POST …/update-check` against the vendor's channel JSON —
Muse Code's release channel and Antigravity's per-platform release manifest
(`manifests/<os>_<arch>[_musl].json`, the same file its installer reads; only
`version` is used). They seed no Activity, install no intercept wrapper, expose no launch
settings (`PUT /api/clis/{cli}` is refused), no session source and no
lifecycle jobs (`POST …/lifecycle` reaches the plan and is refused as
unmanaged). The surface derives that from two capabilities the server sends:
`integrationCapable` (activity, launch settings, setup panes) and `launchable`
(New terminal); `cliPanes` maps them to the pane list. `surface: detect`
remains for a row that must not launch at all, and `TestCatalogCapabilities`
pins which catalog rows launch and which integrate. `muse --version` runs
with `MUSE_NO_AUTO_UPDATE=1` so the launcher does not background-update.
Every installed row reaches Check for updates through the same ⋯ menu, so a
channel-backed CLI needs no bespoke button.

**One launch surface, read-only without an adapter.** A CLI with no adapter
keeps the *same* launch screens, with the launcher's own data and nothing to
edit: the Launch tab renders the plan summary (`executable`, `args`, `path`,
`env`, `injection`) instead of the defaults editor, `#/clis/new/<cli>` keeps
the CLI row (its options are every `launchable` row, so the screen can switch
to a CLI that does have an adapter) and the Launch preview, and
`#/clis/terminal/<id>` shows that preview for a terminal whose launch failed.
Only what cannot act is absent — the Customize checkbox, launch profiles and
the row menu's Launch settings. The preview is the server's real plan
(`POST /api/clis/<cli>/preview`), including the blocked state: an uninstalled
CLI reads *Not found* and the plan's problem line instead of an empty grid.

`terminal_launches.attempt` retains the latest redacted launch failure/time.
Snapshots include injected branches/files and executable identity. Pending
state detects configuration and binary changes; the editor compares next and
last-applied settings and lists terminals affected by a defaults edit. Restart
prepares all launch artifacts before stopping the existing process, then uses
that exact generation. Spawn failure after stopping is reported without claiming
rollback. Workspace deletion holds terminal locks and collects exact private
launch directories; native CLI data and unrelated terminal directories survive.

The default view is a read-only summary. Restore clears launch customization
without changing reporting; Save remains explicit. Editors guard dirty
navigation through a hash guard installed before route observers, use shared
Zod schemas, and preview after edits with a debounce (not periodic API polling).
Feed invalidation keeps profiles and checks current.
Workspace menus and the palette open the shared terminal editor with context.


ADR-0107 amends session pinning for communication: Grok/Hermes pins use native
reports, and enrolled owners never fall back to cwd/latest-session discovery.
A resumed native conversation retains its mailbox; `/new` invalidates its prior
binding. Grok/Hermes tool calls resolve private setup by their current native
session ID, rather than an inherited bearer. Their launcher installs a small
native hook/plugin with an ownership receipt; it preserves native homes and
executable entrypoints. See [direct communication](direct-session-communication.md).

Communication enrollment preserves the captured tmux pane dimensions during
an exact native resume, so a detached replacement keeps the same composer layout
before a browser attaches. Codex joins Grok/Hermes in per-tool message discovery
(ADR-0111): launch preview and execution include the same credential-free client
shim and discovery variables. Enabling an identified Codex conversation prepares
private setup without replacing its process.

The prompt door (ADR-0089) is the one place a user types into a CLI terminal
**or** an interactive Pi agent TUI. The attach composer opens on demand —
the pane's context menu on the desktop (`Attach files…` / `Ask … about this`),
the header paperclip on the phone — and stages up
to four attachments (4 MB each) under `<cwd>/.picode/drop/` before
`POST /api/terminals/{id}/prompt` (CLI terminal) or
`POST /api/agents/{id}/drop` + `/prompt` (interactive agent) pastes one
bracketed message. Never send an agent id to `/api/terminals/{id}/drop`. Its message
field is a textarea one control height tall that grows to four lines and then
scrolls (`web/shared/domain/attachText.js`): on the desktop Enter sends and
Shift+Enter keeps the newline; on a phone Enter is the newline key (a soft
keyboard has no Shift) and Send or Ctrl/⌘+Enter submits. A **Sketch** button
opens `SketchEditor.jsx` (one copy per shell) — an Excalidraw pad that borrows
the dependency, not the pin studio: no background picture, no pin tables, no
stored scene. The drawing leaves as `sketch.png` through the same drop route,
and its chip reopens for editing (the scene lives in the composer's memory)
until Send. The mobile sheet unmounts while the pad is open, because vaul
treats a pointerdown outside the dialog as a dismiss.

The same door carries **snippet runs** (ADR-0130): the terminal's context
menu gains **Send to terminal…** beside Attach, which opens the snippet
fill sheet and delivers through the shared `pasteToTerminal` path. The
pane's foreground is re-checked at handler time — a launch row stays
true after the TUI exits, and a prompt snippet meeting a bare shell is
refused with 409 `kind` instead of pasted. A **bare shell** pane offers
**Run command…** instead: the snippet-run door (ClearLine + paste +
Enter, one confirm that shows the exact command; a live CLI lease
refuses).

On the phone the pad is mounted inside the shell root (`#m-app`), which owns
the visual viewport and the safe areas (ADR-0044): a body portal is fixed to
the layout viewport, so it sat under the status bar and left the home-indicator
strip unpainted. Its canvas is white paper in both themes — Excalidraw's dark
theme inverts the bitmap (`invert(.93) hue-rotate(180deg)`), so a dark
`viewBackgroundColor` came back as a light canvas under dark chrome, and the
exported PNG carried that inverted paper. The mobile composer's pin sketch
(`PinSketch.jsx`) moved into the shell root the same way.

On a phone the pad follows upstream's mobile-toolbar arrangement instead of the
0.18.1 default: the tool island is pinned to the bottom with the shape actions
above it, and the library/lock/hand column is not shown (the pen is sticky and
Excalidraw's two-pointer gesture pans and zooms). `web/mobile/src/styles/app.css`
carries the rules; they can be dropped once the package ships that layout
(0.18.1 is the latest on npm). The pad's canvas menu keeps background and reset
only — load, save, export and save-as-image belong to the host, whose Insert is
the way out.
