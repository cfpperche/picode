# Packages

Extras for Pi (search, tools, themes). They run with **full access** — only install what you review.

Canonical: [pi packages](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/packages.md).

Open **Agent CLIs**, pick **Pi**, then the **Packages** pane. Pick a target and Install. **Installed** lists what each target has (Update when a newer version exists, Remove); **Marketplace** is the pi.dev gallery.

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

MCP servers: install `npm:pi-mcp-adapter`, then `#/mcps`. Guide: [MCP](/guide/mcp).

Checklist: install `packages/pi-checklist` — an extension, not core — so an agent plans before it changes anything and the sidebar shows its current step. Guide: [Checklist](/guide/checklist).

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

## Package links and configuration

The machine view lives at `#/clis/pi/packages`. Links opened from an agent
retain that agent and its workspace through reloads. Old `#/packages` links
redirect to the new view. Missing targets show a recovery action.

Known packages such as `pi-roles` offer **Configure** in the desktop layout.
Their configuration URL retains the workspace and agent. Opening that URL on
a phone offers **Open desktop layout**, preserving the same target.

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
