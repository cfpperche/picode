# Managed CLI principals (ADR-0159 → ADR-0160 Fatia C)

> Part of [PiCode's architecture](../architecture.md) (ADR-0105: one file per subsystem). Edit here; the index only links.

A **principal** is who a grant, Inbox item, or automation is for.
ADR-0160: a workspace instance of a launchable CLI **is an agent**
(`agents.cli`). Fatia C copied leftover `managed_clis` rows onto `agents`
and dropped the table. ADR-0184 ended unbound CLI terminals: migration 067
binds every launch terminal an agent does not own (sign-in terminals excepted)
to a new agent `a-<terminal id>`, copies its `term:` grants onto that agent and
drops every `term:` grant. `Runtime.Start` never runs for a non-Pi agent.

**Terminal identity holds no grant (ADR-0184).** A caller that carries only a
terminal id resolves to the agent bound to that terminal (`callerAgentID`);
otherwise it keeps the `term:<id>` identity for audits and a browser session
drive, but `computer.Resolve` reads it as off, `browser.Resolve` as the
default, grant edits aimed at it answer 400, the policy lists no longer show
terminal rows, and delivery refuses it (403).

`internal/grant.Principal` is `{kind: agent|terminal, id}`. `Key()` is the
house spelling ADR-0143 already uses: the agent id, or `term:<id>`.
`grant.Key(agentID, termID)` is `FromIDs(…).Key()` — agent wins.

A CLI agent's interactive process is `agents.terminal_id` (unique).
Pi also uses this binding for newly opened interactive processes (ADR-0162).
It is allocated lazily and atomically with its terminal and launch configuration;
RPC-only Pi agents do not need a terminal. Existing live legacy Pi panes retain
their original address until explicitly stopped or restarted. The resolver keeps
old agent terminal URLs working after migration.

Pi's shared launch preserves agent model/provider/thinking, packages and their
isolation, prompt, readonly tools, roles, checklist and compaction identity, cwd
and private session directory. Native session reports update both the terminal
pin and agent session ownership. Agent routes keep the Ask receiver/proof and
prompt/drop contracts; bound Pi terminal doors route through that agent identity.
The active mode selects the activity source: terminal events for interactive,
RPC snapshots for managed. The terminal is not a second fleet entry. Agent
menus retain Pi settings and expose the bound terminal's launch settings.

Agent lifecycle routes and bound-terminal lifecycle routes use the agent lock.
Preparation precedes replacement; shutdown captures owned processes, verifies
exit and keeps a durable refusal receipt when a writer outlives the pane.
Terminal launch refuses a live RPC or legacy TUI rather than creating a second
writer. No deployment or browser reconnect migrates a live process.

Binding does not start a process and does not call `Runtime.Start`.
Deleting the agent deletes that terminal; deleting the terminal deletes
the CLI agent row (announced `agent.deleted` then `terminal.deleted`).
A bound terminal borrows its agent's name at bind time; renaming the
agent renames the terminal in the same mutation (announced
`terminal.updated`), so surfaces that label the process with `term.name`
— the tab strip, the dashboard fleet — follow the rename.
The address resolver exposes the bound tmux session to `runMode`; activity
still comes from the bound terminal, not the process-presence flag. For
non-Pi agents the subtitle names the CLI
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

When an interactive Pi TUI starts without a stored session path, launch mints a
pending session id. A short background resolver watches for the JSONL file and
binds it to the agent as soon as Pi creates it, emitting the normal agent update
so `Continue in…` appears without opening the Sessions view. The resolver is
bounded and harmless when Pi exits before writing a session.

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
`{cli}` (Fatia B); the sidebar's free **New agent** opens the same picker
and POSTs `/api/agents` `{cli, name, path}` (ADR-0179) — a guest gets a free
launch terminal on its work folder, Pi keeps provider/model/thinking. The
picker lists Pi first when it is installed and only installed CLIs after it;
an absent Pi is absent. `#/clis` remains the runtime hub. `#/clis/new`
remains a free CLI terminal.

Decision table: [ADR-0160](../decisions/0160-cli-runtimes-are-agents.md).
