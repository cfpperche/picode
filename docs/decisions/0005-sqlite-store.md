# ADR-0005: SQLite (pure Go) as PiCode's store — orchestration data only

- **Status**: accepted
- **Date**: 2026-08-23

## Context

M2+ needs persistent state beyond the M1 flat JSON registry: task queues
(`steer`/`follow_up` with delivery status), agent configurations (M3 wizard),
orchestration events (activity/audit), and later broker inboxes (M4).
Options:

1. **Flat JSON files** (status quo) — fine for one list; race-prone and
   unqueryable for tasks/history.
2. **SQLite** — one file, real queries, transactions, mature.
3. **bbolt** — key/value only; task/history queries become manual scans.

Constraint that dominates: PiCode ships as a **single cross-compiled binary**
(AFR ADR-0001). CGO-based SQLite (`mattn/go-sqlite3`) breaks trivial
cross-compilation. `modernc.org/sqlite` is a pure-Go SQLite driver —
compiles anywhere Go does, no C toolchain, keeps `go build` honest.

## Decision

**SQLite via `modernc.org/sqlite`**, one database at `~/.picode/picode.db`,
embedded migrations (go:embed `.sql`, sequential, transactional), WAL mode.
Schema v1: `workspaces`, `agents`, `tasks`, `messages` (reserved for M4),
`events`, `settings`, plus `schema_migrations`.

### The data boundary (the important part)

PiCode's DB stores **only the orchestration overlay**. Canonical data stays
where it already lives, owned by Pi:

| Data | Source of truth | PiCode's role |
|---|---|---|
| Session history/tool calls | `~/.pi/agent/sessions/*.jsonl` | read-only index/view (SessionReader) |
| Provider credentials | pi's `auth.json` | **never copied** — drive `/login` in the terminal |
| MCP servers | `.mcp.json` / `~/.config/mcp/mcp.json` | edit the files (visual manager) |
| Skills/extensions | files on disk | manage files |
| Agent runtime truth | live tmux/RPC state | cache status only (`last_*` columns) |

Consequences: no sync hell, no credential liability, and a user can drop
PiCode without losing anything Pi doesn't already have. The `events` table
is orchestration audit (started/stopped/task events) — **not** a chat log.

## Consequences

- **Easier**: tasks/history queries; transactions for multi-step lifecycle
  ops; inspectable via `sqlite3` CLI (plain-formats philosophy holds).
- **Harder**: one justified dependency (pure-Go SQLite, large but
  dependency-free itself); migration discipline from day one (cheap —
  embedded sequential migrations).
- **Deletion semantics (v1)**: hard delete + `ON DELETE CASCADE`, documented.
  Soft-archive is a future migration if the lifecycle moat demands it.
  Session JSONL and project folders stay unless the UI opt-in purge runs
  (last occupant of that cwd; work folder only under `~/.picode/work/`).
- **If wrong**: the store package is the only SQL consumer; swapping the
  engine later touches one package.

## Alternatives considered

- **Keep JSON forever**: rejected — task queues + event history are the
  product's spine (the moat is lifecycle control); unqueryable files
  would cap M2.
- **bbolt**: rejected — no queries; we'd hand-roll indexes.
- **Postgres/remote DB**: rejected — violates single-binary local-first
  tool; nothing here is multi-host (yet).

## Schema v1 summary

- `workspaces` — id, name, unique path, created_at
- `agents` — per-workspace pi instances (v1: one "default" per workspace;
  M3 wizard creates more); nullable config (provider/model/thinking);
  cached `last_status*`
- `tasks` — kind (`prompt|steer|follow_up`), payload, source
  (`user|broker|api`), status machine (`queued→delivering→delivered|failed`,
  `cancelled`), attempts/last_error
- `messages` — broker inbox (M4): from/to agent, body, read_at
- `events` — append-only orchestration audit (JSON `data`)
- `settings` — key/value app settings (e.g. default theme)
