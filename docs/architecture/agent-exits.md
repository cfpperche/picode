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
| Setup | `provider`, `model`; `config` JSON: thinking, op mode, checklist, packages, extra prompt, work path, `launch` (a CLI agent's `terminal_launches` record: args, path, tools, the applied snapshot, and **env names only** — an override can carry a key, so values never reach the exit), `instructions` (the instruction files the agent read: path, bytes and a 12-hex SHA-256 of each at removal, `source` `observed` from the CLI's record or `declared` from the rules — `cliinstructions.AgentRevisions`; absent on exits before 2026-09-24) |
| Observed | `created_at`, `removed_at`, `lifetime_s`, `turns`, `first_worked_at`, `last_worked_at`; `signals` JSON: last status, Inbox items and how many were blocking, checklist done/total |
| Sessions | `sessions` JSON: the Pi session path (or, for an agent that never bound one, the newest file in its private folder) or the CLI's last session id/path, and `cwd` — the folder its CLI ran in (ADR-0205); `sessions_purged`, `work_purged` |
| Question | `origin` (desktop/mobile/api), `asked`, `ask_skip` (off/idle/brief/client/workspace) |
| Answer | `outcome`, `reasons`, `note` (≤ 1,000 characters), `labeled_at`, `taxonomy` |
| Undo | `undone_at`, `restored_agent_id` — set by Undo and by a restore from the history; undone exits leave every list and count; the export keeps them |
| History | `forgotten_at` (migration 072) — taken out of the agent history; still in the catalog |

Events: `agent_exit.recorded`, `agent_exit.updated` (label, undo, restore, forget),
`agent_exit.deleted`.

## Cost

At removal, before any purge, the server prices the sessions the exit points
at with `climetrics.MeterSessionFile` (the dashboard's rule: the CLI's own
recorded cost, list price per ADR-0185 only for unpriced turns, kept apart as
`estimated`). A Pi agent's own session folder counts whole (scope `agent`);
any other CLI names only the session PiCode last saw (scope `last-session`).
Pi, Claude Code, Codex, Omp and Muse keep sessions as files; for the other
CLIs, or with no session known, `cost` is NULL — "not measured", shown as —,
never $0.00. Column `cost` (migration 071, JSON). The summary adds `cost`,
`estimated` and `costMeasured`.

## Turns

`agents.turns` counts transitions into work, bumped by `Store.NoteAgentTurn`
from three edges:

| Agent | Edge | Where |
|---|---|---|
| Managed Pi | RPC `agent_start` | `internal/rpc/runtime.go` `pumpEvents` |
| Pi TUI | the watcher's busy flip, not its first scan | `internal/server/tui_watch.go` |
| Every other CLI | a `working` report after idle or no state; `working` after `needs-you` is the same turn resuming | `startsTurn` in `internal/server/term_state.go` |

Both terminal report paths apply `startsTurn`: the plain one
(`reportTermStateForRun`) and the native-session one
(`recordNativeTerminalObservation` in `internal/server/native_session.go`),
which writes the state itself and carries every report from an integrated
CLI — Claude Code, Codex, Omp. Until 2026-09-23 only the plain path counted,
so terminal agents recorded 0 turns and every removal skipped the question as
`idle`; exits recorded before the fix keep that `ask_skip` and can still be
answered from Outcomes.

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

## Agent history (ADR-0205)

`#/history` (desktop: user menu ▸ Tools, and the palette; phone: More ▸ Tools ▸
Agent history, `#/more/history`, `web/mobile/src/screens/HistoryList.jsx`,
without the filters) lists the removed
agents that can come back. There is no table: `GET /api/agent-history` reads
`Store.AgentHistoryCandidates` (not undone, not forgotten, points at a
session) and keeps the exits a `clisession.Locator` still finds on disk —
Pi by file, any other CLI by session id in `sessions.cwd`, then machine-wide
(exits written before `cwd` was recorded). The locator memoizes one listing
per CLI and folder for the request. An entry answers the session summary,
the folder, and whether the folder and the workspace still exist. For
file-backed Codex, Claude Code and Omp sessions, the locator first checks
the recorded transcript path and session id. It reuses an unchanged file's
summary across reads (size and modification time validate the cache). A
PiCode-launched Omp agent keeps its transcript under
`<dataDir>/omp-sessions/<agentId>`; history also checks that private root,
including by id when the recorded path is missing. The normal Omp session
picker still reads Omp's default store.
When a recorded file is gone, or an exit recorded none, the locator looks up
the session id in the file names (`filesByID`: Claude Code `<id>.jsonl`,
Codex `rollout-…-<id>.jsonl`, Omp `…_<id>.jsonl`), reading names only; no
match means the transcript is gone. These three CLIs are never listed to
find one session: listing summarized every file on the machine, 8.0 s on
the owner's 13 exits against 1.2 GB of Claude Code and 2.8 GB of Codex
sessions, 0.24 s after (same 8 found, 2026-09-24). Only the DB-backed CLIs
still go through the folder and machine-wide listings. The page names its first load and later refreshes; a
failed refresh keeps the previous results visible with a retry action.

| Route | Does |
|---|---|
| `GET /api/agent-history` | `{entries: [{exit, session, folder, folderExists, workspaceExists, canDeleteFile}]}`, newest removal first, at most 500 exits read |
| `POST /api/agent-history/{id}/restore` | `{workspaceId?, name?, undo?}` → 201 `{agent, terminalId, resume, envKeys}` under the removed agent's id (ADR-0211); the refusals are ADR-0205's table (409 restored or id taken / `workspace_gone` / `folder_gone`, 410 transcript gone unless `undo`, the launch pre-flight). The removal toast's Undo calls it with `undo: true`: no transcript is needed, and a private folder purged under the data dir's `work/` is made again. Both callers go through `restoreAgent` (`web/shared/client/launchAgent.js`), which also starts the agent: a terminal's `launch/start` (with `resume` when a session was pinned), else a Pi agent's `managed/start`, which is what shows its conversation |
| `POST /api/agent-history/{id}/forget` | `{deleteFile?}` → the exit with `forgottenAt`; `deleteFile` only for a Pi file under Pi's sessions root that no living agent is bound to |

A restored Pi agent gets the exit's provider, model, thinking, tools,
checklist, extra prompt and `sessionPath`. It keeps its id (ADR-0211), so
its private session folder (ADR-0040), where the chat and session list read,
is its own again; with a new id (ADR-0205 as first shipped) the conversation
resumed in Pi and showed nowhere. Any other CLI gets a terminal with
the frozen launch (executable, PATH, tools, integration, removed env; env
values were never kept — their names come back as `envKeys`) and the found
session pinned as `terminal_launches.last_session`; the client then calls
`launch/start` with `resume`, which applies the CLI's own resume arguments.
The agent works in the folder the conversation ran in, whatever workspace it
joins. Code: `internal/server/agent_history.go`, `internal/clisession/locate.go`,
`web/browser/src/components/AgentHistory.jsx`.

## Known limits

- Cost covers a CLI agent's last known session only, and nothing for CLIs
  whose sessions live in a database (Grok, Hermes, OpenCode, Antigravity).
- Asking only at removal leaves out good agents that are never removed.
- The phone's Outcomes (More ▸ Outcomes, `#/more/outcomes`,
  `web/mobile/src/screens/OutcomesList.jsx`) lists records, answers later,
  deletes and carries the switch; the numbers' breakdowns and the filters
  stay on the desktop page.
