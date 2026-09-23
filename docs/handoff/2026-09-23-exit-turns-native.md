# 2026-09-23 — feat/exit-turns-native: terminal agents on native sessions count turns

Found live: the owner removed an Omp agent that had done 175 turns ($0.82 measured) and the exit question never opened. Its exit row had turns=0, first_worked_at NULL, ask_skip='idle'. Every live terminal agent in production (claude-code, codex, omp) also read turns=0.
Cause: integrated CLIs report state with their native session id through `recordNativeTerminalObservation` (internal/server/native_session.go), which writes TermStates directly and never ran `startsTurn`/`noteTerminalTurn`. Only the plain `reportTermStateForRun` path counted, and the prior test covered only that path.
Shipped (fb54de40a): the native path computes `startsTurn(prev, had, state)` and counts through a deferred `noteTerminalTurn` after its locks release. tui_watch skips terminal-bound agents, so there is no double count. Docs: `docs/architecture/agent-exits.md`; fragment `docs/changelog.d/exit-turns-native.md`.
Verified: new test `TestNativeSessionReportsCountTurns` (internal/server/agent_exits_test.go), confirmed to fail without the fix. No green `make close` recorded when this note was written.
Not fixed by this branch: exits recorded before the fix keep ask_skip=idle (they can be answered from Outcomes). Live agents' turn counters start counting from the deploy and read 0 until their next working edge.
Not deployed (owner's call).

## Next up

- After deploy: remove a terminal agent that has worked a turn and been alive ≥ 60 s; the removal dialog must ask the outcome question.

## Debts

- none.
