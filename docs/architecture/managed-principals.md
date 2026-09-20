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

A CLI agent's interactive process is `agents.terminal_id` (unique).
Binding does not start a process and does not call `Runtime.Start`.
Deleting the agent deletes that terminal; deleting the terminal deletes
the CLI agent row (announced `agent.deleted` then `terminal.deleted`).
Because `runMode` never sees the terminal's tmux session, the row reads
its state from the bound terminal: the subtitle names the CLI
(`agentSubtitle`), the status pill and the Start/Restart/Stop menu act on
the terminal (`agentRowMenu`, shared with the node tests), and Launch
settings opens `#/clis/terminal/<terminalId>` — the same screens the
Agent CLIs hub offers. Its face is the CLI's favicon, the same one the
terminal rows wear (`ProviderFace` → `CliAgentFace`); a CLI agent never
offers Open chat: the TUI is the conversation (owner 2026-09-19).

Workspace menus name lifecycle actions **Start / Restart / Stop agent**
for every CLI. Order is lifecycle, Launch settings, chat (Pi only),
terminal, rename, then separated removal. Pi settings open
`#/clis/pi/settings?agentId=<id>`; other CLIs keep the launch editor above,
hidden when there is no binding or the adapter does not support it.
Stop/restart confirmations and completion messages use the agent wording;
unbound terminal menus retain terminal wording. Pi restart preserves its
current mode: interactive uses `open?restart=1`, managed closes then starts
the managed run. Both interrupt current work and require confirmation.

| Agent / state | Lifecycle actions | Settings / chat |
|---|---|---|
| Pi stopped | Start agent | Scoped Pi settings / chat |
| Pi interactive | Restart agent (interactive), Stop agent | Scoped Pi settings / chat |
| Pi managed | Restart agent (managed), Stop agent | Scoped Pi settings / chat |
| Other CLI, running terminal | Restart agent, Stop agent | Bound launch settings when supported / no chat |
| Other CLI, stopped or missing terminal | Start agent | Bound launch settings when supported / no chat |
| Other CLI, no binding or unsupported adapter | Lifecycle as above | No launch settings / no chat |

Menu coverage: `web/shared/domain/agentRowMenu.test.js`; browser QA exercises
confirmation cancellation, mode-specific restart routing and failure feedback.

HTTP:

- `GET /api/workspaces/{id}/principals` — the workspace's agents as
  `principalView` (kind `agent`, id/key the agent id, `cli`, `terminalId`
  when the TUI is a terminal).
- `POST /api/workspaces/{id}/principals` `{cli, name?, terminalId?}` —
  creates an agent (same class as `POST /agents`). Empty `terminalId`
  attaches a new launch terminal for a CLI agent; a given id binds that
  workspace terminal. `ws_free` is 400.
- `DELETE /api/managed-clis/{id}` — alias for `DELETE /api/agents/{id}`
  (one-release compat). The terminal goes with the agent.

Workspace list/get carry `agents` only. The change feed patches
`agent.added` / `agent.deleted`. Structured chat, JSON-RPC, ACP and
`SendTurn` for CLI agents are out of scope (ADR-0091).
When a CLI agent TUI reports `needs-you`, PiCode files one blocking Inbox FYI
(`reason=cli-needs-you`, source the agent — Fatia E). The item closes when
the CLI moves on or the agent (with its terminal) is deleted. Push rides
`inbox.created`. Unbound terminals stay chips-only.
The Inbox action **Open terminal** focuses the pane (`goto: term:<id>`).
No composer. Deploy readiness already treats any working terminal as busy.

A CLI agent is the opt-in for `picode mcp` at launch (ADR-0154): Claude
Code, Codex and OpenCode get every family (`computer`, `browser`, `inbox`,
`checklist`) when Tools was unset. An explicit Tools list, including
empty, is left alone. Other CLIs stay Connectors-only. Every CLI agent launch
carries `PICODE_AGENT_ID` alongside `PICODE_TERM_ID` (Fatia E), so
`grant.FromIDs` resolves the agent principal and the grants given to the
agent in Settings ▸ Computer / Browser match without per-terminal
configuration — the same spelling Pi agents get from `Agent.SpawnEnv`.

A workspace **New → Agent** picker POSTs `/api/workspaces/{id}/agents`
`{cli}` (Fatia B). Pi is always listed; other CLIs must be installed.
`#/clis` remains the runtime hub. `#/clis/new` remains a free CLI terminal.

Decision table: [ADR-0160](../decisions/0160-cli-runtimes-are-agents.md).
