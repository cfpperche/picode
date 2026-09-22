---
description: Extras for Pi and for the other agent CLIs, installed by each CLI's own commands. Full access — only install what you review.
---

# Packages

Extras for Pi (search, tools, themes) and, on each of the other agent CLIs'
pages, that CLI's own plugins. They run with **full access** — only install
what you review.

- **Where:** select an agent, then **Agent CLIs → Packages**, or **Packages** on that agent's ⋯ menu.
- **Not this:** not MCP servers (that is [Connectors](/guide/mcp) after the adapter) and not API keys ([Providers](/guide/providers)).

Canonical: [pi packages](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/packages.md).

Pick a target and Install. **Installed** lists what each target has (Update when a
newer version exists, Remove); **Marketplace** is the pi.dev gallery. Without
an agent selected, only **This machine** is offered.

| Target | Who gets it | How |
|---|---|---|
| This machine | Every agent on this computer | `pi install` |
| This workspace | Every agent in this folder | `pi install -l` |
| This agent | Only that agent, every session | PiCode remembers it and starts pi with `-e` |

This agent is PiCode-only. pi has no per-agent install. Restart the agent after adding or removing.

If a package on this machine or in this folder is behind, that row shows **Update**. The desktop sidebar's Agent CLIs icon marks when any are. Nothing updates until you click. Git and pinned versions stay as they are.

**Only this agent's packages** (checkbox): skip machine and folder packages. The agent starts with its own list, or none. Also skips skills/themes discovered from those places. Restart to apply.

| | pi TUI | PiCode |
|---|---|---|
| Machine / project | `pi install` / `-l` | same |
| One agent | — | **This agent** |

Search: install `npm:pi-web-search` (This machine). Chat expands `web_search` into source cards. Needs a model that supports native search (Gemini / OpenAI / Anthropic).

MCP servers: install `npm:pi-mcp-adapter`, then Agent CLIs → Connectors. Guide: [MCP](/guide/mcp).

Checklist: install `packages/pi-checklist` — an extension, not core — so an agent plans before it changes anything and the sidebar shows its current step. Guide: [Checklist](/guide/checklist).

Work browser: install `packages/pi-browser` — an extension, not core — so an
agent can open a browser beside its session and drive that tab. Headless
browsing stays on the headless tool the runtime already provides.
Guide: [Browser tools for pi](/guide/browser-tool).

Computer use: install `packages/pi-computer` — an extension, not core — so an
agent can use the Windows desktop through the desktop app: screenshots,
mouse, keyboard, windows, the clipboard, opening programs. Off until you
switch the agent on in Settings ▸ Computer; on, the agent acts with your own
permissions on your desktop, without a sandbox.
Guide: [Computer use for pi](/guide/computer-tool).

Docker sysadmin: install `packages/pi-sysadmin` to give an agent Docker
inspection and operation tools. The Docker App works independently of the
package. Guide: [Docker and sysadmin](/guide/docker).

Model roles: install `packages/pi-roles` — an extension, not core — so a session switches to a vision or plan model by content, and stays dormant until a roles file exists. Guide: [Model roles](/guide/roles).

Inbox tools: install `packages/pi-inbox` — an extension, not core — so a plain terminal `pi` can file notifications and blocking questions into the Inbox app. Guide: [Inbox tools for pi](/guide/inbox-tools).

Delivery: install `packages/pi-delivery` — an extension, not core — so a pi agent
declares a change and asks the human for review in the project's **Git ▸
Delivery** view without shelling out. Agent CLIs opened with the **Delivery**
switch under their **PiCode tools** receive the same tool over MCP. Guide:
[Follow changes in Delivery](/guide/delivery).

Compact earlier: install `packages/pi-compact` — an extension, not core —
so long sessions summarize at 100k tokens or half the window, with a cheap
model, instead of waiting for the edge. Dormant until a config file exists.
Guide: [Compact earlier](/guide/compact).

Diff panel: install `packages/pi-diff` — an extension, not core — so `/diff`
in the pi TUI opens a side panel with every changed file and the hunks of the
one the agent touched last. Guide: [Diff panel for pi](/guide/diff-panel).

## Other agent CLIs

On Claude Code, Codex, Grok, Hermes Agent, OpenCode, Muse Code, Antigravity
and Omp, **Packages** manages that CLI's own plugins. PiCode does not keep a
list of its own: it asks the CLI, and the CLI's answer is what you see. So
the pane only ever offers what that CLI can do — where a CLI has no way to
turn a plugin off, or no plugin marketplace at all, the pane says so in one
line instead of showing a control that would do nothing.

| CLI | Installed list | Install / Remove | Turn on / off | Marketplace |
|---|---|---|---|---|
| Claude Code | yes | yes | yes | this machine, the workspace, or local to you |
| Codex | yes | yes | no — install or remove are its only plugin commands | yes |
| Grok | yes | yes | yes | yes |
| Hermes Agent | yes | yes | yes | its curated catalog |
| OpenCode | yes | yes | remove to turn off | none — plugins are npm modules named in `opencode.json` |
| Muse Code | yes | yes | yes | yes |
| Antigravity | yes, read from its own text output | yes | yes | linked from its own command |
| Omp | yes | yes | yes | sources only |

Scopes follow each CLI: most install for the machine, Claude Code, OpenCode,
Muse Code and Omp also for the workspace, and **no** CLI has a per-agent
plugin list (that one is Pi's alone).

Installing or removing runs in the background, so you can close the page. It
refuses while that CLI has a terminal open, because a plugin only loads the
next time the CLI starts — stop the terminal, or confirm and restart it.

Where a plugin comes with a permission prompt of its own (a marketplace that
wants to run a command, a plugin that wants to replace built-in tools), the
pane shows the refusal and the exact command to run in a terminal. PiCode
never answers that kind of prompt for you.

Two details worth knowing: **OpenCode** has no command to remove a plugin, so
PiCode edits the `plugin` list in your own `opencode.json` — only that entry
changes, comments and the rest of the file stay as they were. And a pane that
cannot read a CLI's list shows the CLI's own error rather than an empty list,
so "nothing installed" always means nothing is installed.

## Package links

The machine view lives at `#/clis/pi/packages`. Links opened from an agent
retain that agent and its workspace through reloads. Old `#/packages` links
redirect to the new view. Missing targets show a recovery action.

## Configure a package

Packages that are known to have settings show **Configure** on their card
(desktop layout; a phone offers to open it there). The form is generated
from a short description of the package's settings, so every package looks
the same: a few typed fields, a Save button, and the exact file path shown
before anything is written.

The file stays the package's own. PiCode only writes the fields you
filled in — anything else already in the file is kept exactly as it was,
and a file it cannot read is never replaced behind your back (you are
asked first).

Where the file lives depends on the package: machine-wide (under
`~/.pi/agent`, for every agent on this computer) or in the workspace
folder (for agents working there). The page says which before you save.

### Describe a package yourself

If a card has no **Configure** and you know the package reads a settings
file, use **Describe config…**: name it, say where the file lives and
which fields it has (text, on/off, number, a choice from a list, or a
masked secret). Your description is saved locally and the card gains
**Configure** right away.

The config page shows **Described by you.** with actions to edit or
delete the description at any time. Deleting it removes only the
description — the settings file itself is never touched. If the package
also ships its own description later, the package's own wins over the
catalog, and yours wins over both.

## Tool captures

PiCode can display **Last capture** inside a tool result when a compatible
package supplies one. Click the image to enlarge it. A capture is a still
image, not continuous monitoring or browser control. No capture means the
ordinary tool row remains unchanged.

Image display is built in. Emission must be enabled in a compatible package
loaded by the agent; a machine installation can serve multiple workspaces.
The browser-package emitter is not available yet, so installing
`pi-agent-browser-native` alone does not enable this feature today.

Captures accept bounded PNG/JPEG data images, not external image URLs.
**Capture unavailable** means the image was refused or could not load; use
**View result**, or **Retry** after a load failure. Images omitted from a
final result are not retained by PiCode's conversation replay.

Enabling emission can save sensitive page content in Pi session files and
backups, visible to paired devices. PiCode cannot redact secrets from pixels.
The package and browser run on the agent's machine, not on the observing phone.
