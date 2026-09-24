# ADR-0205: A removed agent stays restorable while its transcript is on disk — the exit record is the tombstone

- **Status**: accepted (owner direction, 2026-09-23); amended by ADR-0211 (a restore keeps the agent's id)
- **Date**: 2026-09-23
- **Boundary**: persistence + protocol — `agent_exits` gains `forgotten_at`
  (migration 072) and its `sessions` JSON gains `cwd`; new routes
  `GET /api/agent-history`, `POST /api/agent-history/{id}/restore` and
  `POST /api/agent-history/{id}/forget`. Amends ADR-0194 (an exit can now be
  consumed by a restore, not only by the eight-second Undo).

## Context

The owner asked for an agent history (2026-09-23): a soft delete that keeps a
removed agent while its transcript exists on disk, and a panel to manage those
agents' lifecycle and location, so a person can bring back a session they need.

What existed after ADR-0194:

| Fact | Consequence |
|---|---|
| Removal deletes the `agents` row and writes an `agent_exits` row in the same transaction | The frozen setup and session pointers already outlive the agent |
| The exit keeps the Pi session path or the CLI's last session id/path, not the folder the CLI ran in | Claude Code files sessions by folder; a lookup needs it |
| Undo is client-side, from an in-memory snapshot, eight seconds long | Nothing brings an agent back after the toast |
| Undo recreates a CLI agent with a fresh terminal | Only Pi got its conversation back |
| `clisession` reads all nine CLIs' stores and returns verified `ResumeArgs` | Finding and resuming a session is solved per CLI |
| `?sessions=1` deletes Pi's files only | Another CLI's transcript outlives a "purged" removal |

The owner chose, among the options presented (2026-09-23): the exit record as
the tombstone rather than a `deleted_at` on `agents`; show where the agent lives
and allow restoring into another workspace, never moving another CLI's files;
an entry whose transcript is gone leaves the history (the record stays in
Outcomes).

## Decision

**The agent history is the set of exits whose transcript a
`clisession.Locator` still finds on disk.** It is computed on each read: an
exit is listed when it is not restored (`undone_at`), not forgotten
(`forgotten_at`) and points at a session that exists — a Pi file by its path,
any other CLI's session by its id in the folder it ran in, then machine-wide.
There is no retention timer: the CLI (or the person) deleting the transcript is
what ends an entry.

**Restoring** (`POST /api/agent-history/{id}/restore {workspaceId?, name?}`)
creates a new agent from the exit and resumes that session:

| Condition | Answer |
|---|---|
| Exit already restored (Undo or history) | 409 |
| Transcript not found | 410 |
| Target workspace missing (its own was removed, none chosen) | 409 `workspace_gone` |
| The folder the conversation ran in is gone | 409 `folder_gone` — not recreated |
| The CLI cannot launch (not installed, bad settings) | the launch pre-flight's answer |
| Pi | agent with the frozen provider, model, thinking, tools, checklist, extra prompt; the old agent's private session folder (ADR-0040) moves to the new id, where the chat reads, and `sessionPath` = the moved file |
| Any other CLI | agent + terminal with the frozen launch; the session is pinned as the terminal's last session; the client starts it with `resume`, so the CLI gets its own resume arguments |

The agent works in the folder its conversation ran in, whichever workspace it
joins: Claude Code and others resume by id only from that folder, and a session
moved to another folder is a different conversation. "Restore elsewhere" means
another workspace, not another folder. Environment variable values were never
recorded (ADR-0194 keeps names only); the answer lists the names so the person
sets them again. The exit is marked with `undone_at` and the new agent's id, as
Undo does: it leaves the Outcomes counts, and the next removal writes a new exit.

**Forgetting** (`POST /api/agent-history/{id}/forget {deleteFile?}`) sets
`forgotten_at`; the exit stays in Outcomes and no file is touched. With
`deleteFile`, a **Pi** transcript under Pi's sessions root is deleted too,
refused while a living agent is bound to it. Another CLI's files are never
deleted from PiCode: `clisession` is read-only by design, and those stores
(JSONL trees, SQLite databases) belong to their CLI.

The exit now records `sessions.cwd` (the terminal's folder for a CLI agent,
`AgentCwd` for Pi), and a Pi agent that never bound a session (lazy binding)
records the newest file in its private folder, so its exit still points at its
conversation.

## Consequences

- A removal is no longer the end of a conversation: every agent removed from
  now on can come back while its CLI keeps the transcript. Exits written before
  this ADR have no `cwd`; their lookup falls back to the whole machine for
  that CLI.
- The history costs one listing per CLI and folder per read (memoized per
  request); a machine-wide fallback for old exits reads that CLI's whole store.
  Fine at hundreds of exits; a large Codex tree would make the page slower —
  the listing is capped at 500 exits.
- Restoring creates a new agent id. Automations, pins and Canvas edges that
  named the old id stay broken, as with Undo.
- A CLI that rewrites or prunes its store (Codex archiving, OpenCode
  compaction) can silently end an entry; that is the rule working, not a bug.
- If we are wrong about "the folder must exist": a person whose worktree was
  cleaned cannot bring the agent back from the history. The refusal names the
  folder; recreating it restores the option.

## Alternatives considered

- **`deleted_at` on `agents`** (classic soft delete, same id back). Lost: every
  agent query, lock, feed consumer and name-uniqueness check would need to
  filter dead rows, and the exit record already holds the frozen setup.
- **A retention timer** (keep 30 days). Lost: the owner's rule is the
  transcript's existence; a timer would hide agents that can still resume, or
  list ones that cannot.
- **Moving transcripts to relocate an agent.** Lost: Claude Code and Codex key
  sessions by folder and some CLIs store them in SQLite; moving would forge
  another CLI's data.
