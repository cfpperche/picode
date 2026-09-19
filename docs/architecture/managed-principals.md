# Managed CLI principals (ADR-0159 → ADR-0160 Fatia C)

> Part of [PiCode's architecture](../architecture.md) (ADR-0105: one file per subsystem). Edit here; the index only links.

A **principal** is who a grant, Inbox item, or automation is for.
ADR-0160: a workspace instance of a launchable CLI **is an agent**
(`agents.cli`). Fatia C copied leftover `managed_clis` rows onto `agents`
and dropped the table. Unbound CLI terminals stay terminals. `Runtime.Start`
never runs for a non-Pi agent.

`internal/grant.Principal` is `{kind: agent|terminal, id}`. `Key()` is the
house spelling ADR-0143 already uses: the agent id, or `term:<id>`.
`grant.Key(agentID, termID)` is `FromIDs(…).Key()` — agent wins.

A guest agent's interactive process is `agents.terminal_id` (unique).
Binding does not start a process and does not call `Runtime.Start`.
Deleting the agent deletes that terminal; deleting the terminal deletes
the guest agent row (announced `agent.deleted` then `terminal.deleted`).

HTTP:

- `GET /api/workspaces/{id}/principals` — the workspace's agents as
  `principalView` (kind `agent`, id/key the agent id, `cli`, `terminalId`
  when the TUI is a terminal).
- `POST /api/workspaces/{id}/principals` `{cli, name?, terminalId?}` —
  creates an agent (same class as `POST /agents`). Empty `terminalId`
  attaches a new launch terminal for a guest; a given id binds that
  workspace terminal. `ws_free` is 400.
- `DELETE /api/managed-clis/{id}` — alias for `DELETE /api/agents/{id}`
  (one-release compat). The terminal goes with the guest.

Workspace list/get carry `agents` only. The change feed patches
`agent.added` / `agent.deleted`. Structured chat, JSON-RPC, ACP and
`SendTurn` for guests are out of scope (ADR-0091).
When a guest TUI reports `needs-you`, PiCode files one blocking Inbox FYI
(`reason=cli-needs-you`, source the terminal — Fatia E rekeys onto the
agent id). Push rides `inbox.created`. Leaving `needs-you` marks that
item done. Unbound terminals stay chips-only.
The Inbox action **Open terminal** focuses the pane (`goto: term:<id>`).
No composer. Deploy readiness already treats any working terminal as busy.

A guest agent is the opt-in for `picode mcp` at launch (ADR-0154): Claude
Code, Codex and OpenCode get every family (`computer`, `browser`, `inbox`,
`checklist`) when Tools was unset. An explicit Tools list, including
empty, is left alone. Other CLIs stay Connectors-only. Grants are still
Settings ▸ Computer / Browser keyed `term:<id>` until Fatia E; the MCP
server never invents identity (`PICODE_TERM_ID` → `grant.FromIDs`).

A workspace **New → Agent** picker POSTs `/api/workspaces/{id}/agents`
`{cli}` (Fatia B). Pi is always listed; other CLIs must be installed.
`#/clis` remains the runtime hub. `#/clis/new` remains a free CLI terminal.

Decision table: [ADR-0160](../decisions/0160-cli-runtimes-are-agents.md).
