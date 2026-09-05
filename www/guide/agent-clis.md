# Agent CLIs

Open **Agent CLIs** from the desktop user menu, `Ctrl+K`, or **More** on a
phone. It manages terminals for your installed Pi, Claude Code, Codex and Grok
commands. Managed agents, structured chat, packages and automations still use
Pi; a CLI terminal is not a new type of managed agent.

## Open a terminal

1. Choose a CLI and select **Check setup** to verify its executable.
2. Select **New terminal**, give it a name and choose a workspace or folder.
3. Select **Open terminal**. Use the CLI's own interface and login flow.

PiCode does not install CLIs or manage their credentials here. A missing
executable offers **Customize**; the CLI's documentation link explains its
installation. Checking setup runs `--version`, without starting a conversation.

## Launch settings

**Launch settings** shows the resolved executable, additional arguments,
PATH additions, environment names and PiCode's injected integration. Automatic
detection stays automatic: displaying a resolved path does not save it as an
override. Select **Customize** to edit, then **Save changes** or **Discard**.
Navigation asks before discarding unsaved edits.

**Restore defaults** clears launch overrides in the editor and keeps the
activity-reporting switch unchanged. Review the preview and save to apply.
New launches use the saved settings; existing processes keep running.

| Setting | Meaning |
|---|---|
| Executable | A command name or absolute path. Leave empty to detect the CLI. |
| Additional arguments | One argument per line. Spaces and shell symbols stay literal. Use `""` for an empty argument; JSON-quoted strings are decoded. |
| Extra PATH entries | Absolute directories searched before the service's PATH. |
| Environment | One `NAME=value` per line. Values are stored locally with launch settings. |
| Activity reporting | Adds PiCode's invocation-scoped hooks or extension. |

Use **Customize this terminal** for exceptions. Unchanged fields inherit the
CLI defaults; changed fields override them. Clearing arguments removes the
defaults, and removing an environment line removes that default key. HOME,
SHELL, PATH, GROK_HOME and PiCode's correlation variables are launcher-owned;
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

## Reuse launch profiles

Expand **Launch profiles** for a CLI, then select **New profile**. Name it,
edit its launch settings and save. **Use** opens a new-terminal form with that
profile selected. Profiles can also be selected while creating a terminal.

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

## Control a terminal

| Action | Result |
|---|---|
| Open | Attach to the existing terminal. No second CLI process. |
| Start | Start a stopped terminal with current settings. |
| Stop terminal | End its processes but keep the saved terminal and settings. |
| Restart terminal | Prepare the next launch, end its processes and launch again. This does not automatically resume a conversation. |
| Remove terminal | End its processes and remove its PiCode record and launch files. Native CLI data stays yours. |

The action menu names the terminal. Interrupting a live terminal requires
confirmation. Closing a browser tab only detaches, and reconnecting never
restarts stopped work. If the CLI exits, its terminal remains an ordinary shell.
Preparation failure leaves the old process intact. A later process-start
failure is still possible; the terminal retains its settings and last failed
attempt so you can repair and retry. Removing a workspace also removes the
private launch files for its terminals, without removing native CLI data.

The **Terminals** tab includes configured CLI terminals and ordinary terminals
where a supported CLI is observed. Launch identity, live CLI presence and
activity are separate: **Installed** is not **Working**, and an enabled hook
is not a received event. See [activity reporting](terminal-status).

## Scope

Launch defaults apply to terminals opened by this manager. Commands typed
manually in ordinary PiCode terminals retain the existing wrapper integration,
without adopting these default arguments or environment overrides. PiCode does
not rewrite your global CLI settings or grant blanket hook trust.
