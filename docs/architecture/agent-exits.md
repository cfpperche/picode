# Agent exits and Outcomes (ADR-0194)

> Part of [PiCode's architecture](../architecture.md) (ADR-0105: one file per subsystem). Edit here; the index only links.

A person's removal of an agent writes one **exit record** in the same
transaction as the delete: who the agent was, how it was set up, what PiCode
observed, where its sessions were, and the person's optional answer. The
catalog is `#/outcomes`; the dashboard's **Agent outcomes** section counts it.
Study: [2026-09-23 — feedback when an agent is removed](../benchmarks/2026-09-23-agent-exit-feedback.md).
This is option A of that study; the insights that read the catalog (B), a model
review at removal (C) and periodic learning (D) point at exit ids and do not
exist yet.

## Which removals write an exit

| Path | Exit | `asked` / `ask_skip` |
|---|---|---|
| `DELETE /api/agents/{id}` (the remove dialog, desktop and phone) | yes | from the body and ADR-0194's decision table |
| `DELETE /api/workspaces/{id}` | one per agent, same transaction | `false` / `workspace` |
| `DELETE /api/terminals/{id}` on an agent's terminal, the launch's `remove` | yes (`endTerminalAgent`) | from the body, else the table |
| `Store.DeleteAgent` (rollbacks of launch, handoff, adoption, principals) | no — nobody decided | — |

`Store.RemoveAgentWithExit` and `Store.DeleteAgent` share `removeAgent`: exit
insert, `agent_exit.recorded`, checklist rows, the agent row and
`agent.deleted` commit together; the bound terminal is deleted after the
commit (unless `LeaveTerminal`, when the terminal route deletes it itself).
Everything the exit freezes is read **before** `Begin`: the store has one
SQLite connection, and a query on `s.db` inside a transaction waits on itself.

## The record

Table `agent_exits` (migration 069), no foreign keys: the agent row is gone by
design and a workspace removal keeps its exits.

| Part | Columns |
|---|---|
| Who | `agent_id`, `agent_name`, `cli`, `workspace_id`, `workspace_name` |
| Setup | `provider`, `model`; `config` JSON: thinking, op mode, checklist, packages, extra prompt, work path, `launch` (a CLI agent's `terminal_launches` record: args, path, tools, the applied snapshot, and **env names only** — an override can carry a key, so values never reach the exit) |
| Observed | `created_at`, `removed_at`, `lifetime_s`, `turns`, `first_worked_at`, `last_worked_at`; `signals` JSON: last status, Inbox items and how many were blocking, checklist done/total |
| Sessions | `sessions` JSON: the Pi session path or the CLI's last session id/path; `sessions_purged`, `work_purged` |
| Question | `origin` (desktop/mobile/api), `asked`, `ask_skip` (off/idle/brief/client/workspace) |
| Answer | `outcome`, `reasons`, `note` (≤ 1,000 characters), `labeled_at`, `taxonomy` |
| Undo | `undone_at`, `restored_agent_id` — undone exits leave every list and count; the export keeps them |

Events: `agent_exit.recorded`, `agent_exit.updated` (label, undo),
`agent_exit.deleted`.

## Turns

`agents.turns` counts transitions into work, bumped by `Store.NoteAgentTurn`
from three edges:

| Agent | Edge | Where |
|---|---|---|
| Managed Pi | RPC `agent_start` | `internal/rpc/runtime.go` `pumpEvents` |
| Pi TUI | the watcher's busy flip, not its first scan | `internal/server/tui_watch.go` |
| Every other CLI | a `working` report after idle or no state; `working` after `needs-you` is the same turn resuming | `startsTurn` in `internal/server/term_state.go` |

Rows born before migration 069 read `NULL` ("not measured", shown as —) and
stay `NULL`; their worked times still move. `NoteAgentTurn` is listed in
`silentMutators`: the edge already rides the ephemeral notices.

## Asking

The cleanup preview (`GET /api/agents/{id}/cleanup`) carries
`exit: {ask, skip, taxonomy}`; the dialog shows the question only when `ask`.
`ExitAskDecision` is ADR-0194's table: the switch (`agent_exits.ask`
setting, default on) first, then "worked" (turns ≥ 1, a worked time, or not
measured), then lifetime ≥ 60 s. The client reports whether it showed the
question; when it did not, the skip is the server's reason or `client`.

The taxonomy lives in `internal/store/agent_exits.go` (codes and labels, v1)
and travels to the clients; `web/shared/domain/agentExit.js` only shapes it.
Both ConfirmDialogs render the same block; its CSS is one shared sheet,
`web/shared/styles/agent-exit.css`, imported by all three entries (browser,
desktop, mobile). The width rule skips `.dlg-sheet`, which keeps its full
width on the phone and in narrow windows.

## Routes

| Route | Does |
|---|---|
| `GET /api/agent-exits` | the catalog, newest first; `workspace`, `cli`, `outcome` (a code or `unanswered`), `range` (today/7d/30d/all), `before` + `limit`; answers `{exits, next, ask, taxonomy}` |
| `GET /api/agent-exits/summary` | counts for the same filters except outcome: outcomes, by CLI, reasons, asked / answered / answered-when-asked, median lifetime and turns. The clients' **Resolved** share divides by resolved + partial + unresolved (`exitHeadline`): a trial set no task |
| `GET /api/agent-exits/export` | every exit, undone included, as JSON lines |
| `GET`/`PATCH`/`DELETE /api/agent-exits/{id}` | one exit; PATCH labels or clears the answer |
| `POST /api/agent-exits/{id}/undo` | links the exit to the agent an Undo brought back |
| `GET`/`PUT /api/agent-exits/prefs` | the ask switch |

## Known limits

- Cost per agent is not recorded: climetrics measures per CLI and session
  (ADR-0127). The session pointers let it be derived later while the files
  exist.
- Asking only at removal leaves out good agents that are never removed.
- The Outcomes page is desktop only; the phone asks the question but has no
  catalog screen yet.
