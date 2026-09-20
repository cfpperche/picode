# CLI runtimes as agents

Owner 2026-09-19: CLI runtimes are the same class as Pi. Talk to them through
the TUI (interactive) until each CLI gets managed mode. ADR-0091 still
holds. ADR-0160 supersedes 0159's "never an agents row".

## Keep (do not rewrite)

The Agent CLIs stack is the **interactive implementation**:

- `clilaunch` catalog, install, check, on-disk sessions
- `terminal_launches`, start / stop / restart, sensors, pins, ADR-0158 resume
- `picode mcp` injection, prompt door (0089/0107), Inbox `needs-you`
- `#/clis` as the **runtime** page (install, machine defaults, providers)

A project shell stays a terminal. `#/clis/new` without a workspace stays a
free CLI terminal, not an agent.

## Move

- Workspace instance: `managed_clis` → `agents.cli` (+ interactive terminal)
- New → **Agent** (one item; picker is Pi + installed CLIs)
- Grant identity for those instances: agent id (`KindAgent`)
- Sidebar: one list of agents

## Slices

| Slice | Ships |
|---|---|
| **A** (this branch) | `agents.cli` (default `pi`); `Runtime.Start` refuses non-Pi; ADR-0160 |
| **B** | `POST /api/workspaces/{id}/agents` `{cli}` creates a CLI agent + launch terminal; one New → Agent picker. `#/clis` hub stays (install, Launch, Sessions, Providers, Settings, Packages, Connectors). |
| **C** | Migrate `managed_clis` → `agents`; principals list = agents; drop the table — this branch |
| **D** | CLI agent rows speak CLI: subtitle, terminal status, Launch settings + lifecycle in the agent menu (desktop and mobile); remaining "Pi agent" labels in peer contact lists |
| **E** (this branch) | Rekey Inbox / `picode mcp` / grants onto the agent id |
| **F** | Automations and Inspector ask through the prompt door (old Fatia 4) |
| **∞** | Managed mode per CLI — out until ADR-0091 is re-measured |
