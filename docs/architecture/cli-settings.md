# Native CLI settings (ADR-0101)

> Part of [PiCode's architecture](../architecture.md) (ADR-0105: one file per subsystem). Edit here; the index only links.

Agent CLIs has CLIs and Messages tabs; a CLI's page hosts Launch, Terminals,
Sessions, Providers, Settings, Packages and Connectors panes (a run/setup
split in the inner tablist). Canonical settings are `#/clis/<cli>/settings`.
The shared `cliSettings` domain module parses canonical/legacy routes and
declares native settings capabilities, currently Pi only. Each app owns
`CliSettings` and the embedded Pi editor; no presentation crosses app
boundaries.
The native settings view does not load terminal inventory or installation jobs.
Explicit agent IDs are validated against Pi's report before looking up their
workspace. Free agents have no project layer; missing identities and unsupported
CLIs show recovery actions without falling back to Pi or another agent.
Pi settings/keys APIs, native files and trust remain unchanged. A transient
context refresh failure keeps the mounted editor and its drafts, with writes
blocked until a successful retry. A missing agent/workspace removes the editor.
Native defaults have a separate loading/error boundary from agent controls and
key bindings; the mobile quick sheet remains editable when the native file is
unreadable, without turning guessed defaults into agent overrides.
A failed desktop restart propagates to the editor as partial success after
PATCH; a failed stop prevents start. Success is reported only after the full
sequence completes. Desktop and mobile retain their own runtime UI adapters.

Each app owns an `AgentClisFrame` for all Agent CLIs tabs. The desktop frame
uses one 1240px maximum width, a consistent unpadded card and a stable scrollbar
gutter; mobile uses the full page width. Route IDs do not control page sizing.

## Pane shape (2026-09-12, `docs/plans/cli-settings-ux.md`)

The pane edits **one layer at a time**. A labelled switcher (*This machine*,
the workspace, the agent) writes `layer=global|project|agent` onto the route
beside `agentId`, the body renders only that layer's rows, and the file it
writes is named under the switcher. Values a layer does not set come from its
parent, and the row says so (`From This machine`, `Pi default`); a row this
layer sets carries the accent bar, `Set here`, and **Use inherited**, which
sends `patch.reset[]` so `pisettings.Apply` deletes exactly those keys (an
empty `compaction` object goes with its last key; an unknown name is refused
with 400 rather than reporting an inheritance that did not happen). Compaction,
steering and follow-up are live-applied from the *effective* values after a
reset, so a running agent never keeps the override that just left the file.

The agent layer keeps PiCode's own fields (Model, Tools, Checklist) and the
`PATCH /api/agents/{id}` path, and it sits outside the native defaults
fieldset: an unreadable `settings.json` leaves it editable (ADR-0101 recovery).
The `layer` param survives a reload; an unknown value is dropped and the pane
falls back to the default layer — the deepest layer the route names (an agent
link opens the agent's layer), or the machine layer for a `focus=scoped-models`
shortcut. Uncommitted pattern text is kept per layer while the pane stays
mounted. Clicking the pane tab itself is plain navigation: it lands on the
defaults.

The keyboard map is the **Keyboard** pane next to Settings
(`#/clis/pi/keyboard`, owner's call 2026-09-12: two tab rows inside one pane
read as nesting). It is machine-wide — `keybindings.json`, its own endpoint
`/api/pi-keys` — so it has no layer switcher and keeps its own filter, and a
failing `settings.json` read cannot lock it. Its link carries the settings
context (`agentId`, `layer`) so a round trip lands back on the same agent and
layer; the route ignores the rest. A `?tab=keys` link from the sub-tab day
redirects to it.
