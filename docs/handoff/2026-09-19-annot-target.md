# 2026-09-19 — Send lands in the bound chat (feat/annot-target)

Owner: a browser open on his session must deliver to HIS chat, not the
first running terminal. Agreed — first-running was a placeholder, and it
misdelivers by design.

## What landed
- Split panes pass `boundSession` (agent id or `t:<id>`); `resolveSendTarget`
  prefers it with a decision table (bound miss never falls through).
- Agent panes deliver via `/api/agents/<id>/prompt`, terminal panes via
  `/api/terminals/<id>/prompt`; staging rides the terminal row in both.
- Tests: decision-table unit (11/11 file), test-js green, web build green.
  Bound-path live run is the owner's (Windows shell).

## Next up
- Owner: desktop-restart, split browser on the desktop terminal, Send 1.

## Debts
- Agent-bound staging assumes the terminal row shares the agent id/cwd
  (interactive convention) — verify live; managed agents reject paths
  (inline-images variant open).
- Prompt door's 4-files-per-paste cap still bounds 3+ note batches.
