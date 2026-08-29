# Adopt a Pi session as a PiCode agent

- **Date:** 2026-08-29
- **Status:** in flight (`feat/adopt-pi-session`)
- **Ratified:** 1C · 2C · 3C

A Pi TUI session (JSONL under `~/.pi/agent/sessions/`) becomes a PiCode
agent. The original Pi is never stopped, stolen, or written.

## Decisions

| # | Choice | Meaning |
|---|---|---|
| 1C | Live or file | Anything on disk is adoptable. A live TUI still has a file. |
| 2C | **Copy** | New JSONL next to the source. `--session` on the copy. User closes the TUI themselves if they want. |
| 3C | Place | Known workspace path → new agent there. Else free agent with `workPath` = session cwd. |

ADR-0006: one writer per file. Copy is the only safe adopt.

## Refuse

| Temptation | Why not |
|---|---|
| Attach the live tmux pane | Steal. User said they close/remove the terminal themselves. |
| `--session` on the original file | Two writers if the TUI is still up. |
| Auto-add a workspace | 3C is free agent when the folder is unknown. |

## API

- `GET /api/pi-sessions` — every JSONL on this machine
- `POST /api/pi-sessions/adopt` `{ "path": "…" }` — copy, create agent, stopped

## UI

New agent → **From a Pi session**. Empty: “No Pi sessions on this machine.” + New agent.
