# omp-checklist

Opt-in checklist mirror for omp: omp's native `todo` tool stays the plan —
this extension watches its committed results and mirrors them to
[PiCode](https://github.com/cfpperche/picode), which shows the current step
on the sidebar (ADR-0055), the way `pi-checklist` serves pi.

- **No new tool, nothing gated.** The todo contract, omp's own reminder and
  the TUI stay authoritative; the mirror only publishes what omp committed.
- **Statuses.** Omp's five map onto PiCode's three: `pending` and `blocked`
  → `pending` (a blocked task names its blocker in the text), `in_progress`
  → `in-progress`, `completed` → `completed`; `abandoned` leaves the plan
  (the TUI keeps the history). Plans over 50 tasks clip to PiCode's store
  cap; multi-phase plans prefix the phase name.
- **Resume-safe.** `session_start` replays the last committed snapshot from
  the branch, or resets the row — a resume never resurrects a dead task's
  plan. Headless runs (`omp -p`) stay silent through the TUI-mode guard, and
  so does any session outside a PiCode terminal.

## Install

The extension file is `extensions/checklist.ts`; any omp extension scope
works, per session:

- **Global** — every omp session on the machine:
  `omp config set extensions '["/abs/path/to/omp-checklist/extensions/checklist.ts"]'`
  (or install the package and point at the store's copy), or PiCode's
  Agent CLIs ▸ Omp ▸ Packages ▸ Global.
- **This workspace** — a row in the workspace's `.omp/settings.json`:
  `"extensions": ["<path>/extensions/checklist.ts"]`, the way `pi-browser`
  ships in the picode repository.
- **Agent** — the agent's package list (Packages pane, agent scope); each
  entry launches as `-e` (ADR-0176 slice 4).

## How it reaches PiCode

`tool_result {toolName: "todo"}` carries the committed snapshot
(`details.phases`); the mirror flattens it and POSTs
`{sessionId, items: [{text, status}]}` to `/api/agents/{id}/checklist` (a
bound agent id wins) or `/api/terminals/{id}/checklist`, resolved from
`PICODE_URL` / `PICODE_TERM_URL` / `<data dir>/server.json` with the
install token — the same resolution the picode-mcp families use.
