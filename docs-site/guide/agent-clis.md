---
description: Terminals, session history, and moving a conversation between your installed CLIs.
---

# Agent CLIs

Terminals, session history, and moving a conversation between your installed
Pi, Claude Code, Codex, Grok, Hermes Agent and OpenCode commands. Muse Code
and Antigravity can open a terminal of their own and list and resume their own
sessions. Muse Code and Antigravity have editable launch settings
(executable, arguments, environment, profiles) like the rest. Antigravity
reports activity (Working while it thinks, uses tools or starts up; Ready
when idle) through a reporter PiCode installs as its title command — it
never reports needs-you, approvals still happen in its own terminal. Muse
Code reports nothing yet (its terminals read Open).

Omp (oh-my-pi, a fork of Pi) works like the rest: editable launch settings,
an Activity reporting toggle (it shares pi's extension API, so it reports
Ready and Working), Sessions with resume, and Check for updates. It joins
**Continue in…** both ways: its own sessions move to another CLI, and other
CLIs' conversations arrive as native omp sessions you resume with
`--resume <id>`. It is a
Bun program and needs Bun 1.3.14 or newer on PATH — an older Bun fails
Check setup with a syntax error from its bundle. One conflict to know:
omp refuses to start if your launch arguments carry `--trusted-extension`
while Activity reporting is on, so PiCode names that combination in the
launch preview instead of starting a broken terminal.

Every CLI's pane carries the same tabs — Launch, Terminals, Sessions,
Providers, Settings, Keyboard, Packages and Connectors. The tabs a CLI has no
native editor for say so: *in development — coming soon*. They are
placeholders, not a promise about a specific release.

- **Where:** last icon in the desktop sidebar header, `Ctrl+K`, or **More** on a phone (`#/clis`).
- **Not this:** a CLI terminal is not a managed Pi agent. Structured chat is Pi's managed mode; every other CLI runs its own TUI.

## Native packages

The **Packages** pane on a CLI manages that CLI's own plugins — Pi's packages
for Pi, the vendor's plugins for the other eight. The catalog lists the CLIs
with an implemented package integration; terminal support alone does not add
package management. Open Packages from an agent to retain its workspace and
agent target. Pi and Omp also carry PiCode's own per-agent layer, whose entries
the CLI receives at its next start. See [Packages](/guide/packages) for
installation and configuration.

## Start an agent

1. Choose a CLI and select **Check setup** to verify its executable.
2. Select **New agent**, give it a name and choose a workspace or folder
   (a free agent with no folder gets a private one).
3. Select **Create agent**. It starts at once in its terminal; use the
   CLI's own interface there.

Every CLI you start from PiCode is an agent: it appears in the sidebar and
the fleet, and permissions, Inbox and automations address it by name.
**New agent** here is the same door as New → Agent in a workspace, with the
launch settings and profiles of this page on top.

If you type a CLI into a plain PiCode shell instead, the shell shows
"*Claude Code is running in this shell* · **Make agent**". Nothing changes until
you press it: the shell then becomes an agent in the same workspace (or a free
one, working in the shell's folder), and from its next start it launches that
CLI with its agent settings. Removing that agent later closes the shell
with it. A CLI you run outside PiCode is not affected.

PiCode installs a missing CLI only through npm: **Install** appears for Pi,
Claude Code, Codex, OpenCode and Omp when the CLI is missing and npm is
available, running the same npm command the reinstall action uses. Grok,
Hermes Agent, Muse Code and Antigravity install through their own guides.
Credentials are not entered here; they live in the CLI's **Providers** pane. A missing
executable also offers
**Customize**; the CLI's documentation link explains its installation.
Checking setup runs `--version`, without starting a conversation.

On a running CLI terminal, **Attach** sends a photo or a file: PiCode saves
it in the terminal's folder and types the path into the TUI. On a phone it is
the paperclip in the header; on desktop, right-click the pane and choose
**Attach files…**. The Pi agent composer still opens Photos for managed chat.

## Inside the terminal pane

Right-click anywhere in a terminal for PiCode's own menu: copy, paste and
select all, **Find…**, the text size, the folder in **Files**, and rename,
settings, close the tab or remove the terminal. Point at a path or a link
before you right-click and the menu offers to open it. On a running Agent CLI
the menu adds an **Ask** row that names it — **Ask Claude Code about this** —
carrying the selected text into the message bar, and **Attach files…**.
Hold **Shift** while you right-click for the browser's own menu instead; that
modifier is yours to change under **Settings → Context menu**.

The message bar is not permanent chrome. It opens from that menu and closes
with its close button or `Esc`, so the terminal keeps the whole editor the
rest of the time. One selected line arrives as the message; a longer
selection is attached as `selection.txt`, which the CLI reads as a file.

### Find in the terminal

`Ctrl+Shift+F`, or **Find…** in the menu, opens a field over the pane —
searching never resizes the terminal, so nothing running inside it redraws.
It highlights every match and counts them; `Enter` and `Shift+Enter` walk
forward and back, `Esc` closes. Three toggles narrow the search: `Aa`
matches case, `ab` matches whole words, and `.*` reads what you typed as a
regular expression. A pattern that does not compile yet says *Invalid
pattern* instead of reporting nothing found. The toggles are remembered until
you reload the page.

### Keyboard

| Key | What it does |
|---|---|
| `Ctrl+Shift+F` | Find in the terminal |
| `Enter` / `Shift+Enter` | Next / previous match, while the find field is open |
| `Esc` | Close the find field |
| `Ctrl+Shift+C` / `Ctrl+Shift+V` | Copy / paste |
| `Ctrl+C` | Copy when text is selected; interrupt otherwise |
| `Ctrl+V` | Paste |
| `Shift+drag` | Select text |
| `Ctrl+=` / `Ctrl+-` / `Ctrl+0` | Bigger, smaller and default text size |
| `Shift+Enter` | New line in the CLI's prompt (the key is a terminal setting) |
| `` Ctrl+` `` | New terminal |

Plain `Ctrl+F` stays with whatever runs in the pane — `less`, `vim` and the
shell's own line editor all use it — which is why find takes the shifted
chord. You can change every key here under **Settings → Shortcuts**.

## Launch settings

In a workspace agent's **⋮** menu, choose **Launch settings** to edit its
terminal launch. Pi uses the same editor as Claude Code, Codex and the other
integrated CLIs. The action appears once the agent has a linked terminal.
Saving takes effect on its next interactive launch; it does not restart the
agent or switch it out of chat mode.

For a Pi agent's model, reasoning, tools or checklist, choose **Settings** in
the same menu or follow the **Settings** link in the launch editor.

On a CLI's page, **Launch**, **Terminals** and **Sessions** are panes of
that CLI. **Launch** shows the resolved executable, additional arguments,
PATH additions, environment names and PiCode's injected integration. Automatic
detection stays automatic: displaying a resolved path does not save it as an
override. Select **Customize** to edit, then **Save changes** or **Discard**.
Navigation asks before discarding unsaved edits.

**Restore defaults** clears launch overrides in the editor and keeps the
activity-reporting switch unchanged. Review the preview and save to apply.
New launches use the saved settings; existing processes keep running.

**Common controls** sit above an **Advanced** reveal of the fields below.
They write the CLI's own documented flags into the same argument list —
everything you typed there stays, and clearing a control removes its flag.
Each CLI offers what its installed flags support:

| Control | Flag it writes | Offered for |
|---|---|---|
| Model | `--model` | Pi, Claude Code, Codex, Grok, Hermes Agent, OpenCode, Omp |
| Additional folders | `--add-dir` | Claude Code, Codex, Omp |
| Thinking / Reasoning effort | `--thinking`, `--effort`, `--reasoning`, `-c model_reasoning_effort=…` | Pi, Codex, Grok, Hermes Agent |
| Approvals | `--permission-mode`, `--ask-for-approval` | Claude Code, Codex, Grok |
| Sandbox | `--sandbox` | Codex, Grok |
| YOLO | `--yolo`, `--always-approve` | Codex, Hermes Agent, Grok (auto-approve) |

Codex refuses YOLO together with a sandbox or approval choice, so picking
one clears the others (and vice versa). Skipping prompts or granting full
access shows a one-line warning on the control. The Grok sandbox offers the
built-in profiles (`workspace`, `devbox`, `read-only`, `strict`) and
accepts a custom profile name typed in the field. Muse Code and
Antigravity have no editable launch settings.

| Setting | Meaning |
|---|---|
| Executable | A command name or absolute path. Leave empty to detect the CLI. |
| Additional arguments | One argument per line. Spaces and shell symbols stay literal. Use `""` for an empty argument; JSON-quoted strings are decoded. |
| Extra PATH entries | Absolute directories searched before the service's PATH. |
| Environment | One `NAME=value` per line. Values are stored locally with launch settings. |
| Activity reporting | Adds PiCode's invocation-scoped hooks or extension. On for every catalog CLI until you turn it off. |

Use **Customize this terminal** for exceptions. Unchanged fields inherit the
CLI defaults; changed fields override them. Clearing arguments removes the
defaults, and removing an environment line removes that default key. HOME,
SHELL, PATH, GROK_HOME, HERMES_HOME, OPENCODE_CONFIG and PiCode's correlation variables are launcher-owned;
use the dedicated PATH field for extra executable directories.

Settings edited after a launch show **Launch changes pending**. The terminal
editor compares **Next launch** with the last applied launch. Executable
replacement also marks a launch pending. **View launch details** separates
user arguments from PiCode's injected arguments, files and environment.
The preview uses the launcher's resolver and does not execute a CLI or write
files. `run-{next}` is a placeholder for a new private directory. Codex's hook
capability is checked by its wrapper at launch; both possible branches are shown.

Arguments with common secret flags are masked, and environment values are
hidden in previews and terminal events. The editor can display saved values:
this is configuration storage,
not a credential vault. Prefer the native CLI's authentication mechanism.
Native model, permission and session settings are not part of this preview.

## Find a session

Open a CLI and choose **Sessions**. The list is that CLI's on-disk sessions,
grouped by folder. Search by name, preview or folder. **Resume as agent**
starts that CLI again as an agent in the session's folder; the exact arguments come from
the CLI itself (for example `claude --resume`, `hermes --resume`,
`opencode --session`).

Pi sessions add management actions: **Open with…** moves one of the
folder's agents to that session, **Compact** summarizes its older turns,
**Delete** removes the file, and **Auto-clean orphans** removes abandoned
sessions after the chosen number of days. Sessions are read from disk,
deleting is permanent, and sessions in use by an agent refuse deletion.

## Continue a session in another CLI

A conversation is not stuck in the CLI that started it. Every session row
has a **•••** menu offering **Continue in &lt;CLI&gt;…** for each other CLI that
can receive it. The same action is on the terminal's **•••** menu (sidebar
and Agent CLIs → Terminals) and on the pane's right-click menu, as
**Continue in…**, for the conversation that terminal is running. The list
comes from what each CLI can actually do, so it changes with the source
you picked and with what is installed.

The original terminal stays where it is. Continue opens the other CLI as
a new agent in its terminal — or, for Pi, as a stopped agent in chat. If that CLI
is still writing, you confirm before the newest turns are left behind.

Pi is both a CLI and the platform's managed agent, so its menu entry has
a second level: **Pi agent · chat** opens the conversation as a
stopped Pi agent in chat (it needs no installed CLI), and **Pi agent ·
terminal** starts a Pi agent whose terminal runs the CLI on it. The dialog shows a **Where** section
for Pi, so you can change your mind before continuing.

This copies the conversation into the other CLI. A memory tool that
already injects a “where you left off” note in that folder is separate:
both can run; they are not the same action.

Nothing is written until you confirm. The dialog first shows what would
travel — how many turns and tool calls — and what would stay behind.
Reasoning never travels: it belongs to the model that produced it and its
provider signs it.

There are two ways for the conversation to arrive.

**Native session.** It becomes a real session of the other CLI, which
resumes it as its own. Claude Code, Codex, Grok and Pi get one written
into their session store, always as a new session and never over anything
already there. OpenCode and Hermes Agent keep their sessions in a database
their CLI holds open, so PiCode hands the conversation to their own
`import` command instead of writing that database. Either way PiCode reads
the session back before telling you it worked.

**Brief.** For a CLI with no import path, or when the conversation is
too large to import as a native session (over 64 MB), PiCode writes a
short summary of the recent turns —
the last request, where the previous agent stopped, the recent turns, the
files and commands it touched — and starts the CLI with it. Hermes Agent
has no brief, because it cannot be started with a prompt.

Two more choices appear when they apply. **How much** lets you pick
between what the previous agent still had in view and the whole history,
and shows up only when the conversation was summarized along the way.
**Turn tool calls into plain text** hands them over as ordinary messages,
for a CLI that would rather not read another agent's tools.

What you started stays where it was. Nothing is moved and nothing is
deleted. The new conversation opens with a note saying where it came from,
which model was running there, and that files may have changed since. Both
rows then carry the link — "from Claude Code" on one, "continued in Codex"
on the other — pointing at the terminal or agent that holds it.

Pi arrives differently: instead of a terminal you get a new managed agent,
stopped, in the workspace that owns the folder. Start it like any other.

Two things worth knowing. A session written for another CLI records a
model that CLI can actually serve rather than the one that produced the
turns, because naming a foreign model made one CLI refuse it and switch.
And when the source conversation is still running, PiCode says so and asks
before continuing, since its newest turns may not be on disk yet.

## Fork an agent

**Fork agent…** starts a second agent of the same CLI on a copy of an
agent's conversation, with a task of its own, while the original keeps
going. Use it when an agent finds something worth doing — a failure, a
side question — but already has its next task: the fork takes the finding
with its full context and handles it, and nobody explains the problem
twice.

Open an agent's **•••** in the sidebar and pick **Fork agent…** (offered
once the agent has a conversation, for the CLIs listed below). The dialog
asks for:

- **Name** — the new agent's name.
- **Where** — a **New worktree** (its own checkout on a new branch, from
  the last commit) or the **Same folder** as the original.
- **Task** — what the fork should do first, with photos, files or a
  sketch: paste a screenshot straight in, as in the terminal's attach bar.
  Left empty, the fork opens waiting.

### New worktree or same folder

| Use **New worktree** when… | Use **Same folder** when… |
|---|---|
| the fork will **edit code** while the original keeps editing | the fork only **investigates, explains or reviews** |
| what the fork needs is **committed** | the fork needs the original's **uncommitted changes** (a worktree starts from the last commit and leaves them behind) |
| the project runs in a fresh checkout with no setup | the project needs setup to run there (installed dependencies, a build) |
| the repository has no worktree routine of its own | the agent already makes its own worktree (a repository whose `AGENTS.md` says so) |

The worktree is created by a visible `git worktree add` in a terminal in
that folder. When another agent is working in the repository, PiCode
types the command without running it: the dialog closes, a message asks
you to press **Enter** in that terminal, and the fork starts as soon as
the worktree exists. Its branch is named after the fork.

### Which CLIs fork

Each CLI forks with its own mechanism; PiCode only starts it, so the copy
is exactly what that CLI would make.

| CLI | Fork agent… | How the CLI forks |
|---|---|---|
| [Claude Code](https://code.claude.com/docs/en/setup) | yes | `--resume <id> --fork-session` |
| [Codex](https://learn.chatgpt.com/docs/codex/cli) | yes | `codex fork <id>` |
| [Grok](https://docs.x.ai/build/cli/reference) | yes | `--resume <id> --fork-session` |
| [OpenCode](https://opencode.ai/docs) | yes | `--session <id> --fork` |
| [Omp](https://omp.sh/docs) | yes | `--fork <session>` |
| [Pi](https://pi.dev) | not yet | `--fork` exists; a Pi agent owns its session file in PiCode |
| [Hermes Agent](https://hermes-agent.nousresearch.com/docs/getting-started/installation), [Muse Code](https://dev.meta.ai/docs/muse-code), [Antigravity](https://antigravity.google/docs/cli/) | not yet | only inside their own TUI (`/branch`, `/fork`) |

The fork is not **Continue in…**: that moves a conversation to a
*different* CLI by translating it; a fork stays in the same CLI and copies
it with its own fork. Neither stops the original.

## Reuse launch profiles

**Launch profiles** holds a CLI's reusable settings and opens by default
once a profile exists. Select **New profile**. Name it,
edit its launch settings (quick controls included) and save. **Use** opens
a new-terminal form with that profile selected. Profiles can also be
selected while creating a terminal.

A profile is copied, not linked. Changing or removing it leaves existing
terminals unchanged. Its explicit empty arguments and automatic executable
choice are retained. Clear **Customize this terminal** to return to inheriting
the CLI defaults. Profiles are presets, not managed agents.

The workspace terminal menu offers **Shell terminal** and **Agent CLI terminal**.
`Ctrl+K` also offers **New CLI terminal**, including workspace-scoped shortcuts.

## Check setup and activity

**Check setup** runs a bounded version check and verifies reporter tools.
The result survives daemon restarts; configuration changes or an executable
replacement mark it out of date. It does not start a conversation or repair files.

**Setup and activity** separates executable detection, version verification,
prepared integration files and the latest currently observed activity signal.
**Repair PiCode files** rebuilds only PiCode-owned integration files, without
changing native CLI configuration or restarting terminals. A prepared file is
not proof that every event works with that CLI version. No observed signal
means unverified, even when a CLI process is present.

## Update, reinstall, uninstall or install a CLI

When a newer release exists, the CLI's row shows an **Update** badge and its
detail page offers **Update**. PiCode runs each CLI's own update command —
`claude update`, `codex update`, `grok update`,
`hermes update`, `opencode upgrade` — plus npm for npm-installed Pi,
Claude Code, Codex and Omp. Update checks refresh on
demand and when the saved check is older than six hours.

| What you see | What it means |
|---|---|
| Update badge with a version | A newer release exists; the exact number comes from the CLI's registry or its own check |
| Update check failed | The registry or the CLI's check could not answer; nothing was changed |
| Working… with a progress card | The CLI's updater is running; the card shows its output |
| Done | The update finished and the setup check ran again |
| Interrupted | PiCode shut down while the updater ran. Nothing was retried; run **Check setup** to see the CLI's state |
| No update controls | The install method is one PiCode cannot manage (for example Homebrew or a manual checkout) |

**Reinstall** forces a fresh install of the same CLI. Running terminals of
that CLI keep the old version until you restart them; PiCode asks before it
touches a CLI with live terminals.

When a CLI is **not installed**, its detail page offers **Install** for Pi,
Claude Code, Codex, OpenCode and Omp (needs npm on the machine). Grok and Hermes
Agent install through their own guides — the card links to them and PiCode
never runs their install scripts.

**Uninstall** runs the CLI's own uninstall command (Hermes Agent, OpenCode)
or npm's for npm-installed tools, after you type the CLI's name. Grok and a native
Claude Code install have no uninstall command; PiCode links their official
guide instead. Uninstalling never touches your settings or conversations
beyond what the CLI's own uninstaller does.

## tmux guard

PiCode runs every managed terminal inside tmux, on the same server as your
own sessions. Agent CLIs are literal with commands, and a `tmux kill-server`
typed by mistake takes down every session on that server — PiCode's
terminals and yours. The **tmux guard** is on by default inside PiCode
terminals: it refuses server-wide kills, pattern kills, and kills of
sessions another terminal or you created. An agent may still close a
session it created itself, by exact name.

Each refusal is explained on the terminal, and the reason is logged to
`<dataDir>/tmux-guard.log`. The guard applies only inside terminals PiCode
created; your own shell outside PiCode is untouched. The switch lives on
**Agent CLIs**, above the CLI catalog — not on Terminal defaults, which is
font, color and tmux options. The change applies to terminals opened from
now on. If you are debugging and need raw tmux semantics, the same switch (or
`POST /api/terminals/wiring/tmux-guard/disable`) turns the guard off.

## Where PiCode's terminals live

PiCode runs its own tmux server, one per instance, on a socket inside the
data directory: `~/.picode/tmux.sock`. Your personal tmux stays on tmux's
default socket, so the two can never take each other down — a `kill-server`
on one side leaves the other running. To attach from your own shell:

```bash
tmux -S ~/.picode/tmux.sock attach -t picode-…   # ls, capture-pane, … all take -S
tmux attach -t picode-…                          # your own server: not where they live
```

Terminals created before this change keep running on the old, shared
socket until they end or you restart them — the daemon follows both while
that lasts, and the terminal list shows them side by side.

If you build scripts that create scratch tmux servers, do **not** rely on
`TMUX_TMPDIR` alone: when a command runs inside an existing tmux session,
`$TMUX` wins and the command talks to that session's server regardless of
`TMUX_TMPDIR`. Use an explicit socket (`tmux -S <path>` or `tmux -L <name>`)
for scratch servers, and exact session
names for cleanup.

## Control a terminal

| Action | Result |
|---|---|
| Open | Attach to the existing terminal. No second CLI process. |
| Start | Start a stopped terminal with current settings. |
| Resume last session | Start the terminal and reopen the conversation it was running, using each CLI's verified resume arguments (Claude Code `--resume <id>`, Codex `resume <id>`, Grok `--resume <id>`, Hermes Agent `--resume <id>`, OpenCode `--session <id>`, pi `--session <file>`). Offered on the stopped terminal surface when a conversation is pinned; the surface names the CLI, the conversation and when it last moved. |
| Fork agent… | Start a new agent of the same CLI on a copy of this conversation, with a task of its own, in a new worktree or the same folder ([Fork an agent](#fork-an-agent)). The original keeps going. On the agent's **•••** in the sidebar, when a conversation is pinned and the CLI has its own fork. |
| Continue in… | Open this terminal's conversation in another CLI. The original terminal stays; a new terminal opens, or a stopped Pi agent when you pick "Pi agent · in the app". Offered when a conversation is pinned. |
| Stop terminal | End its processes but keep the saved terminal and settings. |
| Restart terminal | Prepare the next launch, end its processes and launch again. A pinned conversation is reopened with that CLI's verified resume arguments (the same recipe as Resume last session). Without a pin, Restart starts a fresh conversation. |
| Remove terminal | End its processes and remove its PiCode record and launch files. Native CLI data stays yours. |

The action menu names the terminal. Interrupting a live terminal requires
confirmation. Closing a browser tab only detaches, and reconnecting never
restarts stopped work. If the CLI exits, its terminal remains an ordinary shell.
Preparation failure leaves the old process intact. A later process-start
failure is still possible; the terminal retains its settings and last failed
attempt so you can repair and retry. Removing a workspace also removes the
private launch files for its terminals, without removing native CLI data.

### Resume after a restart or crash (ADR-0084)

While a CLI runs, PiCode pins the conversation it is writing (the newest
session of that CLI in the terminal's folder, refreshed as the CLI
reports activity). When a deploy, crash or daemon restart ends the
terminal, its surface offers **Resume last session** — the CLI comes back
in the same conversation. The pin records what was running, so the button
shows the recovered work even after the process is gone, and the surface
names it: the CLI, the conversation and how long ago it last moved (hover
for the opening words and the exact time). Nothing resumes automatically after a crash or daemon restart: a plain Start
still opens a fresh conversation. **Restart terminal** on a live Agent CLI
is the attended path — it reopens the pinned conversation (ADR-0158). Terminals
stopped before this feature shipped have no pin; their conversations stay
reachable in that CLI's Sessions pane via "Resume as agent", and their
surface says *no session to resume* instead of offering a button it cannot
honor. If the last launch failed, the surface says that too — fix the
executable, folder or integration files, then Resume again.

When the terminal died in a daemon restart (not a CLI exit), the surface
says so: "PiCode restarted while this terminal was running." (ADR-0085:
the daemon records its live sessions at shutdown and diffs them at boot.)

### Flight recorder (ADR-0085)

PiCode keeps forensics under the data dir's `var/` folder:

- `shutdown-snapshot.json` — the tmux sessions alive when the daemon
  exited gracefully (root process id, root command, folder).
- `restart-report-*.json` — the boot verdict for each restart: which of
  those sessions survived, which did not. Last 10 kept.
- `deploy-log.jsonl` — one line per deploy: time, binary version, the
  terminal it ran from, and the folder.

If sessions ever vanish around a restart, these three files answer when,
what and who without any forensics archaeology.

Each CLI's page carries a **Terminals** section listing the terminals that
CLI launches — including ordinary terminals where that CLI is observed.
Launch identity, live CLI presence and activity are separate: **Installed**
is not **Working**, and an enabled hook is not a received event. See
[activity reporting](terminal-status).

## Scope

Launch defaults apply to terminals opened by this manager. Commands typed
manually in ordinary PiCode terminals retain the existing wrapper integration,
without adopting these default arguments or environment overrides. PiCode does
not rewrite your global CLI settings or grant blanket hook trust.
