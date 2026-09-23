# ADR-0194: Removing an agent writes an exit record — frozen setup, observed signals, an optional outcome — in the same transaction as the delete

- **Status**: accepted (owner direction, 2026-09-23)
- **Date**: 2026-09-23
- **Boundary**: persistence + protocol — a new table `agent_exits`, three
  columns on `agents` and three event types; `DELETE /api/agents/{id}` takes an
  optional JSON body and answers `200` with the exit instead of `204`; the
  cleanup preview gains an `exit` block; new routes under `/api/agent-exits`.
- **Study**: [2026-09-23 — feedback when an agent is removed](../benchmarks/2026-09-23-agent-exit-feedback.md)

## Context

The owner wants a loop: ask the person when an agent is removed, catalog the
answers, turn them into insights the person acts on, and have agents work better
over time. The study found that no agent product asks at removal, that the
products which learn converged on eight patterns (outcome apart from reason, one
taxonomy, provenance, approval, expiry, measurement, local by default, asking
little), and that a loop without measurement can make agents worse.

What PiCode had on 2026-09-23 made the first step a persistence problem, not a
screen:

| Fact | Consequence |
|---|---|
| `Store.DeleteAgent` deletes the row; `agent.deleted` carries only the id | The configuration is gone at the click |
| `events.agent_id … ON DELETE CASCADE`, and the feed keeps seven days | The agent's own history goes with it |
| The only snapshot is the browser's, for the eight-second Undo | Nothing survives on the server |
| `?sessions=1` deletes the transcripts | Any later review needs them kept, or pointed to before |
| The guest CLIs' launch setup lives in `terminal_launches`, deleted with the terminal | It has to be read before the terminal goes |
| Turn-level state (`terminal.state`, `agent.state`, `agent.tui`) is ephemeral | "Did the agent work?" is not answerable after the fact |
| Cost is measured per CLI and session, never per agent (ADR-0127) | Per-agent cost is a later derivation from session pointers |

The owner took every recommendation of the study (2026-09-23): ask in the
removal dialog, only when the agent worked (at least one turn and one minute),
with a way to stop asking; the taxonomy below; the loop closes first at creation
(the next option, B); no model calls now; warn that deleting sessions deletes
what the exit could be reviewed from; insights per workspace with a global view.

## Decision

**When a person removes an agent through `DELETE /api/agents/{id}`, the server
writes one `agent_exits` row in the same transaction that deletes the agent row
and appends `agent.deleted`.** The row holds:

- **who** — the removed agent's id, name and CLI, the workspace id and name
  (plain text, no foreign keys: the agent is gone and a workspace removal keeps
  its exits);
- **how it was set up**, frozen — `provider`, `model` as columns; thinking, run
  mode, checklist level, packages, extra prompt, work path and, for a CLI agent,
  its terminal launch record (overrides, applied configuration) as JSON;
- **what was observed** — created and removed times and the lifetime, turns,
  first and last worked times, the last runtime status, how many Inbox items the
  agent raised and how many said it needed the person, checklist progress;
- **where its sessions were** — the Pi session path or the CLI's last session
  id and path, plus whether the removal deleted the sessions or the work folder,
  so cost and a later review can be derived while the files exist;
- **the question** — `origin` (desktop, mobile, api), `asked`, and when not
  asked why (`off`, `idle`, `brief`, `client`);
- **the person's answer** — `outcome`, `reasons`, `note`, `labeled_at` and the
  taxonomy version; a person can label, relabel or delete an exit later;
- **an Undo** — `undone_at` and the restored agent's id; undone exits stay in
  the table and leave every count.

Rollbacks that call `Store.DeleteAgent` directly (a failed launch, handoff,
adoption, principal) write no exit: nobody decided anything.

**Turns are counted on the agent, not inferred.** `agents.turns` counts
transitions into work — a managed Pi `agent_start`, a Pi TUI turning busy, a CLI
terminal reporting `working` from idle or no state (not from `needs-you`, which
is the same turn resuming) — and `first_worked_at` / `last_worked_at` record the
edges. Agents born before this migration read `NULL`, "not measured", never 0.
The counter is bookkeeping without an event: the visible edge already rides the
ephemeral notices, and a durable event per turn would flood the seven-day log.

**Asking** is decided by the server and reported by the client:

| Preference | Worked | Lifetime ≥ 60 s | The dialog showed the question | `asked` | `ask_skip` |
|---|---|---|---|---|---|
| off | any | any | — | false | `off` |
| on | no | any | — | false | `idle` |
| on | yes | no | — | false | `brief` |
| on | yes | yes | yes | true | — |
| on | yes | yes | no (API, an older client) | false | `client` |

"Worked" is `turns ≥ 1`, or a `last_worked_at`, or `turns` not measured.
The preference is the setting `agent_exits.ask` (absent = on). A question the
client showed despite a `no` from the preview (the preference changed in
between) is recorded as asked: the person saw it.

**Taxonomy, version 1.** Outcomes `resolved`, `partial`, `unresolved`, `trial`;
reasons, only after `partial` or `unresolved`: `setup`, `misunderstood`,
`stuck`, `false_done`, `slow_costly`, `switched`. The server owns the codes and
their labels and sends both to the clients; an unknown code is a `400`. A note
is at most 1,000 characters.

**The human answer is never overwritten by a machine.** A later model review
(option C) lands in its own table keyed by the exit id, labeled as inferred.
Insights (option B) cite exit ids as their evidence, with their sample size and
the time they were applied, so an insight can be measured before and after.
Neither table exists yet; the exit id is the unit they will point at.

Everything stays in the data dir. `GET /api/agent-exits/export` hands the
person their catalog as JSON lines; nothing is sent anywhere.

## Consequences

- Every removal from now on can be learned from; every removal before it is lost.
- The removal path is one transaction (it was not): exit, checklist rows, the
  agent row and both events commit or roll back together. The bound terminal is
  still deleted after the commit, as before.
- `DELETE /api/agents/{id}` answers `200` with `{exit}`. Both PiCode clients
  handle it; a script that required `204` has to accept `200`.
- One `UPDATE` per turn on `agents`. Turns are paced by people and models, not
  by the scheduler.
- A note is free text and may hold something sensitive. It never leaves the
  machine, the export is the person's own act, and an exit can be deleted.
- Asking only at removal leaves out good agents that are never removed. Accepted
  for now; option B revisits it with the observed signals of living agents.
- If the taxonomy is wrong, version 2 adds codes and keeps version 1's rows
  readable: `taxonomy` is on every row.

## Alternatives considered

- **Soft-delete agents (`deleted_at` on `agents`).** Every agent query would need
  a filter, and the CLI's launch setup lives in `terminal_launches`, which goes
  with the terminal. An exit record is the catalog's own shape.
- **Keep the snapshot in the event log.** Events prune at seven days and cascade
  when the agent row is deleted.
- **Ask after the removal (the Undo toast, or an Inbox batch).** Cheaper for the
  dialog, weaker answers; the owner chose the dialog. A later label from the
  catalog covers removals that came without it.
- **Infer the outcome with a model.** No model calls now (owner); the column
  would also have mixed an estimate into the person's answer.
- **Keep answers in the browser.** Not durable, and a phone and a desktop would
  keep two catalogs.
