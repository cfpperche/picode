---
description: Extras for Pi (search, tools, themes). Full access — only install what you review.
---

# Packages

Extras for Pi (search, tools, themes). They run with **full access** — only install what you review.

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
agent can read the page open in PiCode's work browser (snapshot, screenshot,
events). Read is the default and the ceiling until you grant an agent `act`.
Guide: [Browser tools for pi](/guide/browser-tool).

Docker sysadmin: install `packages/pi-sysadmin` to give an agent Docker
inspection and operation tools. The Docker App works independently of the
package. Guide: [Docker and sysadmin](/guide/docker).

Model roles: install `packages/pi-roles` — an extension, not core — so a session switches to a vision or plan model by content, and stays dormant until a roles file exists. Guide: [Model roles](/guide/roles).

Inbox tools: install `packages/pi-inbox` — an extension, not core — so a plain terminal `pi` can file notifications and blocking questions into the Inbox app. Guide: [Inbox tools for pi](/guide/inbox-tools).

Compact earlier: install `packages/pi-compact` — an extension, not core —
so long sessions summarize at 100k tokens or half the window, with a cheap
model, instead of waiting for the edge. Dormant until a config file exists.
Guide: [Compact earlier](/guide/compact).

Diff panel: install `packages/pi-diff` — an extension, not core — so `/diff`
in the pi TUI opens a side panel with every changed file and the hunks of the
one the agent touched last. Guide: [Diff panel for pi](/guide/diff-panel).

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
