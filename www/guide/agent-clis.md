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
executable shows **Edit launch**; the CLI's documentation link explains its
installation. Checking setup runs `--version`, without starting a conversation.

## Launch settings

Expand **Launch settings** to change defaults for that CLI. New launches use
them; existing processes keep running with their original configuration.

| Setting | Meaning |
|---|---|
| Executable | A command name or absolute path. Leave empty to detect the CLI. |
| Arguments | One argument per line. Spaces and shell symbols stay literal. |
| Extra PATH entries | Absolute directories searched before the service's PATH. |
| Environment | One `NAME=value` per line. Values are stored locally with launch settings. |
| Activity reporting | Adds PiCode's invocation-scoped hooks or extension. |

Use **Customize this terminal** for exceptions. Unchanged fields inherit the
CLI defaults; changed fields override them. Clearing arguments removes the
defaults, and removing an environment line removes that default key. HOME,
SHELL, PATH, GROK_HOME and PiCode's correlation variables are launcher-owned;
use the dedicated PATH field for extra executable directories.

Settings edited after a launch show **Launch changes pending**. **Last applied
launch** shows the resolved executable, arguments with common secret flags
masked, environment names with values hidden, integration choice and start
time. The editor can display saved values: this is configuration storage,
not a credential vault. Prefer the native CLI's authentication mechanism.

## Control a terminal

| Action | Result |
|---|---|
| Open | Attach to the existing terminal. No second CLI process. |
| Start | Start a stopped terminal with current settings. |
| Stop terminal | End its processes but keep the saved terminal and settings. |
| Restart terminal | End its processes and launch again. This does not automatically resume a conversation. |
| Remove terminal | End its processes and remove its PiCode record and launch files. Native CLI data stays yours. |

The action menu names the terminal. Interrupting a live terminal requires
confirmation. Closing a browser tab only detaches, and reconnecting never
restarts stopped work. If the CLI exits, its terminal remains an ordinary shell.

The **Terminals** tab includes configured CLI terminals and ordinary terminals
where a supported CLI is observed. Launch identity, live CLI presence and
activity are separate: **Installed** is not **Working**, and an enabled hook
is not a received event. See [activity reporting](terminal-status).

## Scope

Launch defaults apply to terminals opened by this manager. Commands typed
manually in ordinary PiCode terminals retain the existing wrapper integration,
without adopting these default arguments or environment overrides. PiCode does
not rewrite your global CLI settings or grant blanket hook trust.
