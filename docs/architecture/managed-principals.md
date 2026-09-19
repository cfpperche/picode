# Managed CLI principals (ADR-0159)

> Part of [PiCode's architecture](../architecture.md) (ADR-0105: one file per subsystem). Edit here; the index only links.

A **principal** is who a grant, Inbox item, or automation is for.
ADR-0160: a workspace instance of a launchable CLI **is an agent**
(`agents.cli`). This file still describes the ADR-0159 `managed_clis`
binding (transitional) until the migrate slice copies those rows onto
`agents`. Unbound CLI terminals stay terminals. `Runtime.Start` never
runs for a non-Pi agent.

`internal/grant.Principal` is `{kind: agent|terminal, id}`. `Key()` is the
house spelling ADR-0143 already uses: the agent id, or `term:<id>`.
`grant.Key(agentID, termID)` is `FromIDs(…).Key()` — agent wins.

`managed_clis` stores the guest binding: workspace, catalog CLI, terminal
(unique), name. The terminal, `terminal_launches`, tmux and vendor files
are unchanged. Binding does not start a process and does not call
`Runtime.Start`. Unbinding deletes only the row. Deleting the terminal
announces `managed_cli.removed` then `terminal.deleted`.

HTTP:

- `GET /api/workspaces/{id}/principals` — agents and bound CLIs as
  `principalView` (kind, id, key, name, cli / agentId).
- `POST /api/workspaces/{id}/principals` `{cli, name?, terminalId?}` —
  bind an existing workspace terminal or create one (launch set, not
  started). `ws_free` is 400.
- `DELETE /api/managed-clis/{id}` — unbind. The terminal stays.

Workspace list/get carry `managedClis` next to `agents`. The change feed
patches `managed_cli.added` / `managed_cli.removed`. Structured chat,
JSON-RPC, ACP and `SendTurn` for guests are out of scope (ADR-0091).
When a bound CLI reports `needs-you`, PiCode files one blocking Inbox FYI
(`reason=cli-needs-you`, source the terminal). Push rides `inbox.created`.
Leaving `needs-you` marks that item done. Unbound terminals stay chips-only.
The Inbox action **Open terminal** focuses the pane (`goto: term:<id>`).
No composer. Deploy readiness already treats any working terminal as busy.

Binding is the opt-in for `picode mcp` at launch (ADR-0154): Claude Code,
Codex and OpenCode get every family (`computer`, `browser`, `inbox`,
`checklist`) when Tools was unset. An explicit Tools list, including
empty, is left alone. Other CLIs stay Connectors-only. Grants are still
Settings ▸ Computer / Browser keyed `term:<id>`; the MCP server never
invents identity (`PICODE_TERM_ID` → `grant.FromIDs`).

A workspace **New → Agent** picker (Cursor: runtime ≤2 clicks from the
sidebar) POSTs `/api/workspaces/{id}/agents` `{cli}` (ADR-0160 Fatia B).
Pi is always listed; other CLIs must be installed. A guest agent gets a
launch terminal (`agents.terminal_id`); `Runtime.Start` is not called.
`#/clis` remains the runtime hub (install, Launch, Sessions, Providers,
Settings, Packages, Connectors). `#/clis/new` remains a free CLI terminal.
Uninstalled guests are hidden.

Decision table: [ADR-0159](../decisions/0159-managed-cli-principals.md).
