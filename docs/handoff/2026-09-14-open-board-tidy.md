# 2026-09-14 — feat/open-board-tidy

Paid debts came off the board: `docs/handoff/open/terminal.md` lost the
`make ci` red line (`TestPeerStopStubbornChildStaysPending`), the
`TestCLITerminalResumeDecisionTable` block and the `TestPaneRootSurvivesSIGHUP`
block — all three are fixed on `main` now, and their branches' notes plus
`git log` are the record (`docs/handoff/open/README.md`: delete a bullet in the
branch that paid it).

The durable lesson moved under `## Traps`: a fixed tmux session name shared
across the four shard binaries was the cause, so fixtures need process-unique
names plus a ready-marker handshake before a stop signals a pane.

Bullets went back to one line each, as the README asks; `terminal.md` and
`providers-custom.md` both carried wrapped ones. `make handoff` renders the
board (79 lines) without complaint.

Docs only — no code, no changelog fragment (the board is not user-facing).