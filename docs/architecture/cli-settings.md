# Native CLI settings (ADR-0101)

> Part of [PiCode's architecture](../architecture.md) (ADR-0105: one file per subsystem). Edit here; the index only links.

Agent CLIs has CLIs, Settings, Packages and Messages tabs; a CLI's
page hosts Launch, Terminals, Sessions and Providers panes. The shared
`cliSettings` domain module parses canonical/legacy routes and declares native
settings capabilities, currently Pi only. Each app owns `CliSettings`,
`CliTabs` and the embedded Pi editor; no presentation crosses app boundaries.
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
