# Data & persistence (ADR-0005)

> Part of [PiCode's architecture](../architecture.md) (ADR-0105: one file per subsystem). Edit here; the index only links.

SQLite (pure-Go driver) at `~/.picode/picode.db` — **orchestration overlay
only**. Pi's own files remain the source of truth for sessions, credentials,
MCP and skills; PiCode never duplicates them. Schema v1: `workspaces`,
`agents` (many per workspace, own model/config; free agents in `ws_free` — ADR-0011),
`terminals` (each owned by a workspace, `ws_free` for free ones — ADR-0026; no FK, cascade is app-driven),
`tasks` (prompt/steer/follow_up queue with a delivery state machine;
`inbox-tui:<itemID>` correlates a terminal reply), `inbox_items` (including
the exact asking `session_path`; boot reconciliation settles pending replies
from a full-payload, post-task-timestamp user row before deciding whether to
reopen them), `messages` (reserved M4 broker inbox),
`events` (orchestration audit),
`settings`,
`snips` (ADR-0130: user prompt/shell templates; not the conversation fence runner),
`webapps` (ADR-0147: user-installed web app shortcuts — name, normalized URL with its fragment (duplicates allowed since migration 057 — each row is one account; the unique constraint is gone) the manifest's `start_url`/`scope`/`display`/`theme_color` when the site is a PWA, icon bytes ≤256 KB, creation time, `partitioned` (ADR-0153: installs born after it get their own WebView2 user-data folder); `webapp.installed/updated/removed` in the same transaction). Embedded sequential migrations; the M1 JSON registry is imported
once and retired (`workspaces.json.migrated`).

Local backup destinations are checked against both live data trees before any
snapshot write. The check canonicalizes the longest existing path prefix and
then restores a not-yet-created suffix; resolving only the complete destination
would miss OS aliases such as macOS `/var` → `/private/var`.

| Destination | Path shape | Action |
|---|---|---|
| empty | any | refuse |
| either live data root, or any descendant | existing, missing, direct, or reached through a symlinked ancestor | refuse |
| outside both live trees | canonical roots differ | allow |

## Delivery declarations

Migration 063 adds `delivery_intents` and `delivery_requests` for ADR-0171. The
store commits each declaration, original retry receipt and `delivery.changed`
event atomically. These are agent declarations, not gate or deploy evidence.
See [delivery architecture](delivery.md) for scope, ownership and limits.
