# CLI terminal launch settings (ADR-0069)

> Part of [PiCode's architecture](../architecture.md) (ADR-0105: one file per subsystem). Edit here; the index only links.

`internal/clilaunch` holds a small catalog and configuration resolution; it
does not implement an agent runtime. A workspace instance of a launchable CLI is an agent (`agents.cli`,
ADR-0160). The launch stack here is that agent's interactive mode — see
[managed-principals.md](managed-principals.md) (0159 binding is
transitional). Launch is not a second identity. SQLite tables `cli_configs` and
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

Workspace agents backed by Omp get one additional native boundary: the launcher
appends `--session-dir <data-dir>/omp-sessions/<agent-id>` and creates that
directory with private permissions. The directory is durable across terminal
generations and contains Omp transcripts only; Omp's normal authentication and
configuration home is unchanged. An agent whose launch arguments already name
a conversation or folder (`--resume`, `--session-dir` — a resumed session or a
handoff, whose file is in Omp's default directory) keeps them instead, and so
do standalone Omp terminals, which have no agent owner. This is the only non-Pi
per-agent `/resume` isolation currently enabled.

**One door (ADR-0184).** Every user-facing launch creates an agent:
`POST /api/agents` and `POST /api/workspaces/{id}/agents` take `overrides`
(the launch overrides of a profile, a resumed session's arguments), checked
before any row exists — the workspace route also takes a first `prompt`
(Draft with an agent, `docs/architecture/cli-instructions.md`) — and bind a launch terminal (`newLaunchAgent`). A Pi
agent created with overrides is the interactive shape: its terminal runs `pi`,
and `--session` moves onto the agent's `SessionPath` (reserved on a Pi agent's
launch). The cross-CLI handoff uses `createCLIAgent`, which also starts the
terminal. `web/shared/client/launchAgent.js` is the client door for both apps.
`POST /api/clis/{cli}/terminals` is gone (slice 3). The one terminal that
still carries a launch without an agent is a credential sign-in:
`terminals.kind = 'signin'` (migration 068), created only by
`createSigninTerminal`, kept out of `/api/terminals`, peer owners and the app's
lists (`kind` travels in the terminal view and the feed reducer drops it),
and closed by the server (`signin.go`): when the credential is imported; when
its CLI exits (a sign-in's launch script ends with `exit`, not the shell every
other launch returns to, so the session goes); after `signinOpenLimit` (15 min)
without tmux activity (`#{session_activity}`); and at boot —
`StartSigninReaper` ticks every minute and judges each one under its terminal
lock, with `signinGrace` (30 s) for a row whose session is still being created.
The stamp of the account the store held when the sign-in started is kept per
terminal (`signinStamps`) and returned by the reused POST and by
`GET /api/credentials/signin`, so a card that comes back still tells a new
login from the old one. The card's dialog keeps Esc and outside clicks for the
login (phones get the terminal key bar), Cancel deletes the terminal, and the
strip reads `terminal.deleted` to say it closed.

**The launch editor is an agent's (hardening).** `PUT /api/terminals/{id}/launch`
answers 409 for a terminal no agent owns ("Make it an agent first") and for a
sign-in — the one path that could still give a shell a launch after slice 3.
A launch agent refuses a named folder that is not there instead of creating
it (`launchFolderExists`); a plain Pi New agent (no launch changes) is the
same Pi agent the palette makes.

**Make agent (slice 4).** `POST /api/terminals/{id}/adopt` binds a shell to a
new agent when PiCode has seen a launchable catalog CLI running in it
(`TermRuntimes`, ADR-0062): same workspace (free → free agent), `work_path` =
the pane's live folder unless it is the workspace's, launch set to that CLI
with PiCode's tools filled in (applies from the next start). Refused (409):
nothing detected, already an agent, a sign-in, a launch of another CLI. The
app offers it only where `adoptOffer` says so — a shell with `tui.cli`, no
`launchCli`, no kind and no bound agent.

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

Pi and other agents use the same contextual terminal launch editor. The
desktop agent menu offers Launch settings only after a terminal is bound,
without a duplicate Terminal settings entry for Pi. Both app editors resolve
ownership from workspace and free-agent bindings, including on direct links,
and show the agent's CLI as read-only. Bound Pi terminals omit the model and
thinking quick flags, which `validatePiAgentArgs` reserves for agent Settings;
the editor links there without losing the unsaved-change guard. Defaults,
profiles and standalone Pi terminals retain those controls. Saving launch
overrides applies to the next interactive launch, never restarts a process or
changes a managed run. This adapts the progressive disclosure pattern in
`docs/benchmarks/2026-09-17-cli-launch-quick-presets.md`.

The browser's Pi interactive view uses the same bound terminal record as the
Agent CLIs surface whenever `agents.terminal_id` is present. It attaches via
the terminal session and terminal APIs, so lifecycle, launch settings, and pane
identity remain one port across desktop, mobile, and browser.
Pi Chat versus Terminal is chosen from the sidebar agent menu (Open chat /
Open terminal) and does not create a second terminal or chat runtime.
`GET /api/clis` resolves installation without starting a conversation;
A vendor self-update rewrites its launcher or symlink in place, so one
`Stat` can land inside that swap window and read the CLI as absent — the
resolver retries once (~150 ms) before declaring a CLI not installed
(2026-09-18: four rows flipped their lifecycle off for a single request
while grok, hermes, muse and omp updated themselves).
`POST /api/clis/<cli>/check` explicitly runs bounded `--version` and checks
reporter prerequisites. It does not certify authentication or every hook.
`GET /api/clis/<cli>/sessions?cwd=` lists a CLI's on-disk sessions read-only
(ADR-0079 phase 2; `internal/clisession`): pi reads its JSONL root, Claude
Code its `~/.claude/projects` transcripts, Codex its `~/.codex/sessions`
rollouts, Grok its `~/.grok/sessions` prompt history, Hermes Agent its
`~/.hermes/state.db` (or `$HERMES_HOME/state.db`) SQLite rows, OpenCode
its `~/.local/share/opencode/opencode.db` (or `$XDG_DATA_HOME/opencode/opencode.db`)
SQLite rows, Muse Code its index, Antigravity its summaries DB, and Omp its
`~/.omp/agent/sessions` JSONL (or the launcher's private Omp agent
`--session-dir`, with pi's bucket-per-cwd layout and Omp's own encoding;
`$PI_CODING_AGENT_DIR` is honored for ordinary Omp commands; XDG relocation via
`omp config init-xdg` is not followed yet), each parsed defensively (malformed
files are skipped, missing roots are an empty list). Hermes listing is the
active home only (no `profiles/` scan), `source` cli/tui with a folder and
at least one message; preview is the session title. OpenCode listing skips
child sessions (`parent_id`), archived rows and empty transcripts;
timestamps are milliseconds; preview is the session title. The OpenCode
CLI's own `session list` is project-scoped and is not used. Omp listing
reads the schema-v3 header for id/cwd, the newest `title` record, the
provider-qualified `model_change`, and the first user text as preview; a
file with no header or no message is not a session.
Rows carry the server-verified resume arguments for that CLI (verified
2026-09 against each CLI's `--help`: `claude --resume <id>`, `codex resume
<id>` positional, `grok --resume <id>`, `hermes --resume <id>` on Hermes
Agent v0.18.2, `opencode --session <id>` on OpenCode 1.18.29, `omp --resume
<id>` on omp 18.2.4); pi's row carries none — pi resumes
through its own chat flow. Cost is pi-only on this surface: CLI formats
are not shown with per-session spend. Non-Pi sessions open through
the handoff door (`POST /api/clis/<cli>/sessions/handoff`, ADR-0088) or the
one agents door (ADR-0184), the resume arguments riding as launch
overrides — no transcript replay, no writes, no deletes for other
CLIs' sessions. pi's management surface (delete, auto-clean, resume,
in-use guards) stays on its own endpoints.
`POST /api/agents` (ADR-0184's one door) saves and opens a configured
terminal as an agent.
`/api/terminals/<id>/launch` reads/writes overrides; its `start`, `stop`,
`restart` and `remove` POST routes serialize per terminal. Start is idempotent
while live, Stop retains configuration, and active destructive actions require
confirmation. Restart validates the next executable and directory before
ending the current session. When a conversation is pinned, Restart prepares
that session's verified resume arguments for the next generation (ADR-0158;
same one-shot recipe as `start` with `resume: true`); without a pin it
applies current settings to a fresh conversation. A CLI exit returns to an
interactive shell; browser/daemon reconnect only reconciles, never restarts
work automatically.
Codex can report a native session ID before writing a rollout. PiCode keeps
that live identity but pins it for resume only after the matching top-level
rollout exists in the terminal's folder. An older saved pin survives until
then. A stale Codex pin fails launch preparation before the live pane is
stopped. Omp restart passes the pinned transcript's exact path to `--resume`,
because agent-owned sessions live outside Omp's default lookup directory; a
missing file also fails preparation before stopping the pane.

| Restart condition | Action |
|---|---|
| Codex ID announced, rollout not saved, older pin exists | Keep older pin and resume it |
| Codex pin has no matching saved rollout | Return 400; leave live pane running |
| Codex rollout exists in this folder | Resume that saved conversation |
| Omp private session file exists | Resume by exact file path |
| Omp private session file is missing | Return 400; leave live pane running |
| No pin | Use current launch settings for a fresh conversation |

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
npm where the vendor has no command, or where the vendor command refuses on
npm-managed installs (pi, claude-code, omp). Update facts (`Latest`,
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
the executable is absent, npm-backed for pi, omp, claude-code, codex and
opencode (the same argv as reinstall) through the same job lane, and
guided (docs link, no executable action) for grok, hermes, muse and agy,
whose installers are vendor curl scripts or launchers.
Installing an installed CLI is refused — reinstall covers it. The update
check line no longer reports "failed" for unmanaged installs; the real check
error text is shown for genuine failures.

**Catalog capabilities**: Muse Code (`muse`) and Antigravity (`agy`)
launched as `surface: terminal` rows and graduated to full rows in the
launch-parity project (launch defaults, sessions, PATH wrapper, lifecycle);
their update checks still read the vendor's channel JSON —
Muse Code's release channel and Antigravity's per-platform release manifest
(`manifests/<os>_<arch>[_musl].json`, the same file its installer reads; only
`version` is used). Lifecycle jobs run the vendors' own commands: `agy
update` for Antigravity, and Muse's launcher as its own updater
(`MUSE_LAUNCHER_INSTALL=1` on a bare run — it exits 0 before arg parsing,
so the plan carries env instead of argv); neither ships an uninstaller, so
both uninstall guided with the vendor docs link. Muse still reports no
Activity: R3233 grew real hooks, but the CLI scrubs the hook environment
to PATH alone (measured; `managed_hooks_env_vars` is enterprise-policy
tier), so a settings reporter cannot attribute a report to a terminal —
Open stays honest until a vendor surface carries identity. Antigravity reports
through its title command. `muse --version` runs
with `MUSE_NO_AUTO_UPDATE=1` so the launcher does not background-update.
Every installed row reaches Check for updates through the same ⋯ menu, so a
channel-backed CLI needs no bespoke button.

**Omp is a full row.** Omp (oh-my-pi, a Pi fork) launched as terminal-only,
gained sessions, then its adapter in the omp-adapter slice: editable launch
defaults, the PATH wrapper (presence lease; maintenance subcommands and the
protocol modes `--mode rpc|json|acp|rpc-ui` exec or mark non-TUI), and
activity through the omp-shaped terminal-state extension injected with
`-e` — the same extension load mechanism as pi, but omp's own event set
(verified against the 18.2.6 bundle: `agent_start` opens the run,
`agent_end` settles it only when `willContinue` is unset, approvals arrive
as `tool_approval_requested`/`resolved` and the ask card as
`tool_execution_start`/`tool_result`, so omp reports Ready, Working and
needs-you; the pi events `agent_settled`/`ui_prompt_*` never fire). The
checklist is not injected: `packages/omp-checklist` is an installable omp
package (ADR-0055) the user puts at whatever scope they want — Global
(Omp's user layer), This workspace (`.omp/settings.json`, like
`pi-browser`) or an agent's package list (each entry launches as `-e`).
It watches omp's native `todo` tool — no new tool, nothing gated — and
mirrors the committed phases to the checklist routes, so the sidebar
shows the current step the way `pi-checklist` does for pi.
`Check setup`
runs `omp --version`, which needs
Bun ≥ 1.3.14 on PATH; an older Bun dies with a syntax error from its
bundle. Lifecycle mirrors pi with the real npm package
(`@oh-my-pi/pi-coding-agent`), split by install method — each measured:
npm installs update/reinstall/uninstall through npm (a deterministic
target; run from a bun install, the vendor updater resolved by PATH and
updated an npm copy), native installs run `omp update` / `omp update
--force` with the vendor's own check (`omp update --check`, output
measured), a bun-global install is honestly unknown (no deterministic
mutation target), and a missing omp installs through npm like pi. One measured foot-gun guards both preview and
prepare: omp refuses a run outright when a `--trusted-extension` launch
argument meets the injected `-e`, so that combination is a named problem,
never a broken launch.

**One launch surface, read-only without an adapter.** A CLI with no adapter
keeps the *same* launch screens, with the launcher's own data and nothing to
edit: the Launch tab renders the plan summary
(`executable`, `args`, `path`, `env`, `injection`) instead of the defaults
editor, `#/clis/new/<cli>` keeps the CLI row (its options are every
`launchable` row, so the screen can switch to a CLI that does have an
adapter) and the Launch preview, and `#/clis/terminal/<id>` shows that
preview for a terminal whose launch failed. Only what cannot act is absent
— the Customize checkbox, launch profiles and the row menu's Launch
settings. The preview is the server's real plan
(`POST /api/clis/<cli>/preview`), including the blocked state: an uninstalled
CLI reads *Not found* and the plan's problem line instead of an empty grid.

**Presence lease without activity (Muse Code).** Muse launches through
the PATH wrapper like every other CLI, so presence and pins work — but
its hook reports carry no terminal identity (env scrubbed to PATH), so
there is nothing to report through and activity stays Open.

**Activity through the settings reporter (Antigravity).** The CLI reports its own
lifecycle through the `title` command in its settings file
(`agent_state: idle | thinking | working | tool_use | initializing`), so
PiCode installs a reporter script there — merged into the user's file,
foreign blocks refused, ours removed on toggle-off — and launches the CLI
through the PATH wrapper like every other surface (presence lease and
precise pins; the lease skips maintenance subcommands). The reporter posts
through `picode-hook` (the shared mapper learned the `agent_state`
dialect) and prints the short title the CLI renders. No `hooks.json`
decision hook ships: a PreToolUse command gates tools, and a bad one
would break the owner's sessions. There is no approval signal in the
payload, so Antigravity reports Ready/Working and never needs-you.
The Activity reporting toggle is offered only where the plan carries a
mechanism (arg branches, files, or environment — `hasIntegrationMechanism`
on the catalog row, refused with 400 on PUT and at prepare time); Muse
shows a one-line note instead and its terminals stay Open. The seed that
switches Activity on for empty configs takes the same predicate, and the
startup copy it forces off is never persisted — a future mechanism must
not inherit a stale owner-off.

**The pane tabs are the same everywhere.** `cliPanes` returns Launch,
Terminals, Sessions (when the CLI has a session source) and the setup tabs
— Providers, Settings, Keyboard, Memory, Packages, Connectors, plus Models
where PiCode can ask the CLI what it reaches (omp only today, ADR-0181) —
for every launchable CLI. The setup group is a placeholder wherever the
native editor does not exist yet (`supportsCli*` in `web/shared/domain/`),
rendering *"<Tab> for <CLI> are in development — coming soon"* instead of
the editor. Packages covers the guests since ADR-0167 (their own verb set,
and one line where the CLI has none), Settings and Connectors cover them
since ADR-0163/0150, Providers reads the shared vault (ADR-0165), Memory
serves the six CLIs with a native memory shape (ADR-0163), and Keyboard
stays Pi-only. A CLI's placeholder never links to
Pi. `CliPaneTabs` scrolls the selected tab into view, so a deep link to a late
tab is not a strip reading Launch…Sessions while the panel says Packages.

**Identity is the runtime CLI, else what it was launched with.** A sidebar
row, a tab, a canvas face, a terminal list row and the dashboard's fleet
bucket all resolve through `terminalDisplayCli`
(`web/shared/domain/terminalCli.js`): `tui.cli` while the adapter reports a
runtime, otherwise `cli`, otherwise `launchCli`. A CLI with no adapter never
reports a runtime, so without that last fallback its terminal wears the plain
shell mark, is called *Shell session*, and lands in the dashboard's shell
bucket — `terminalCli` stays the *activity* question ("is a supported CLI
reporting now?") and still gates handoff and the *Activity not reported*
badge. The same identity rides the status label: a CLI terminal with nothing
to report reads **Open**, a plain shell *Terminal open*, and *Ready*/*Working*
stay earned by an activity report.

A terminal created while a page is open reaches that page through the feed,
where the store's `terminal.created` carries the bare row (id, name, cwd,
workspace, createdAt) — no CLI, no runtime, no launch state. Creation
therefore also publishes the `liveTermView` every other terminal response
uses (`publishTerminalState`), and the fleet reducer lets that complete view
seed a row (`terminal.changed` for an unknown id) because the two frames race;
a late bare `terminal.created` never overwrites a filled record.

`terminal_launches.attempt` retains the latest redacted launch failure/time.
Snapshots include injected branches/files and executable identity. Pending
state detects configuration and binary changes; the editor compares next and
last-applied settings and lists terminals affected by a defaults edit. Restart
prepares all launch artifacts before stopping the existing process, then uses
that exact generation — including the pinned session's resume recipe when one
exists (ADR-0158). Spawn failure after stopping is reported without claiming
rollback. Workspace deletion holds terminal locks and collects exact private
launch directories; native CLI data and unrelated terminal directories survive.

The default view is a read-only summary. Restore clears launch customization
without changing reporting; Save remains explicit. Editors guard dirty
navigation through a hash guard installed before route observers, use shared
Zod schemas, and preview after edits with a debounce (not periodic API polling).
Feed invalidation keeps profiles and checks current.
Workspace menus and the palette open the shared terminal editor with context.

**Quick launch settings** (2026-09-17, benchmark note
`docs/benchmarks/2026-09-17-cli-launch-quick-presets.md`): for pi,
Claude Code, Codex, Grok, Hermes Agent, OpenCode and Omp the launch editors
render verified per-CLI controls — model, additional folders
(`--add-dir`, repeatable), approvals/permission mode,
thinking or reasoning effort, sandbox, and the vendors' skip-everything
flags (codex `--yolo`, hermes `--yolo`, grok `--always-approve`) — above an
**Advanced** reveal of the raw fields. Each control patches the draft
argument array in place (`web/shared/domain/cliLaunchPresets.js`): the
generated flag is replaced at its position, every other argument keeps its
order, and an emptied control removes the flag. Codex's yolo/sandbox/
approval controls form an exclusivity group (the CLI refuses the
combination); picking one clears the others. No new persistence — Save,
preview, overrides and profiles keep their contracts. A dangerous pick
shows a one-line warning inline. Muse Code and Antigravity have no adapter
and keep their read-only launch screens.


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
**or** an interactive Pi agent TUI — and, since Fatia F (ADR-0160), the one
place automations deliver to a CLI agent. The attach composer opens on demand —
the pane's context menu on the desktop (`Attach files…` / `Ask … about this`),
the header paperclip on the phone — and stages up
to four attachments (4 MB each) under `<cwd>/.picode/drop/` before
`POST /api/terminals/{id}/prompt` (CLI terminal) or
`POST /api/agents/{id}/drop` + `/prompt` (interactive agent) pastes one
bracketed message. Never send an agent id to `/api/terminals/{id}/drop`. Its message
field is a textarea one control height tall that grows to four lines and then
scrolls (`web/shared/domain/attachText.js`): on the desktop Enter sends and
Shift+Enter keeps the newline; on a phone Enter is the newline key (a soft
keyboard has no Shift) and Send or Ctrl/⌘+Enter submits. Deliveries to CLIs
with a measured input reader are **gated and verified** (Fatia F): the
composer must read empty before, and reads empty again after Enter — the
response carries a `delivery` receipt (`verified` / `unconfirmed` /
`unverified`) and refusals name themselves (`working`, `needs-you`,
`occupied`, `busy`, `closed`). CLIs without a reader keep the blind paste
and answer `unverified`. A working CLI is no longer only refused
(ADR-0206): the request may carry `delivery: "steer" | "follow_up"`, and
`term_delivery.go`'s adapter table — the live measurement in
[the delivery-modes study](../benchmarks/2026-09-23-attach-delivery-modes.md) —
names the one sequence per CLI and mode (a key after the paste, or a slash
command for Hermes `/steer` / `/queue`; Omp's follow-up is Ctrl+Q, which
keeps the payload's lines). A mode the CLI
lacks answers `unsupported-mode`; `needs-you` and a recognized draft are
refused in every mode; an idle CLI gets the prompt path whatever was asked;
no second Enter is ever pressed mid-turn (Grok reads it as "cancel and send
now"). The receipt is `queued` only when the payload's head shows on a new
row outside the input row, else `unconfirmed`; Hermes `/queue` renders
only at turn end, so its receipt is `accepted` when the input row held the
command after the paste and let it go after Enter. `GET` on the same route
returns `{cli, modes, state, termId}`; the composer shows the selector only
while the state is `working`. Interactive Pi agents pass the mode to the
receiver as `deliverAs`. Automations and the extension stay prompt-only. A fourth mode, `interrupt` ("Stop and send",
ADR-0206 amendment), presses the CLI's measured stop key, waits for its
stop line (or its state leaving `working`, or — Omp — its `esc …`
working row gone for two reads) and then runs the verified
prompt path (`doorPasteVerified`); an unseen stop answers `not-stopped` and
a refilled field `restored`, with nothing pasted. Automations aimed at a CLI agent deliver through
this door on the agent's bound terminal and finish with the receipt as
their reason. A **Sketch** button
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
