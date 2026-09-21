# 2026-09-21 — feat/agent-remove-undo: Undo on the removal toast re-creates the agent

Removal toast now offers Undo (notice model action floor, 8s) — commit 23306387. Undo re-creates the
agent in the same workspace: `POST /api/workspaces/<id>/agents` with `cli`/`name`/`workPath`, or
`POST /api/agents` for free agents, then PATCHes `opMode`/`checklist`/`extraPrompt`/
`packagesIsolated`/`sessionPath` (session only when the confirm dialog's purge was NOT picked). The
old id is gone forever and per-agent package selection cannot be carried over. Tab reopens on a
fresh list from `loadWorkspaces` — but `refreshFleetFallback` returns nothing on a live feed, which
made `openTab` silently skip the reopen (found in E2E).

Debt paid from `2026-09-21-agent-remove-tab.md`: CLI-agent bound-terminal removal now has E2E
coverage — on qa-scratch an omp agent (`cli omp`) with a bound terminal was created via
`POST /api/workspaces/<id>/agents`, its terminal started in tmux, the `t:` tab selected, agent
removed: selection landed on the neighbour, terminal row deleted, tmux session gone.

Undo E2E: Pi agent (with `sessionPath`/`opMode`/`extraPrompt`) and CLI agent (omp, fresh terminal
re-attached) — recreate + config restore + tab reopen + selection verified; toast "X is back."

Verified: `make ci-scoped` PASS. visual-review: PASS (remove-agent-undo-toast.webp; overlayAudit
ok; card 5/5).

## Next up
## Debts

- `sessionPurge` path (sessions=1 → no session restore) is logic-reviewed only, no E2E.
- Undo of a free agent (`POST /api/agents` path) not E2E'd.
