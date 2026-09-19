# Managed CLI principals (ADR-0159)

> Part of [PiCode's architecture](../architecture.md) (ADR-0105: one file per subsystem). Edit here; the index only links.

A **principal** is who a grant, Inbox item, or automation is for. Pi agents
are rows in `agents`. Guest coding CLIs stay Agent CLI terminals
(ADR-0069). This subsystem binds a terminal to a workspace as a durable
identity without making it an agent.

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
Inbox, fleet, automations and `picode mcp` rekey onto `Principal` in
later slices; they must not grow a second identity rule.

Decision table: [ADR-0159](../decisions/0159-managed-cli-principals.md).
