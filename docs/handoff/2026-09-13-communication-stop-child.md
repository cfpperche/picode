# 2026-09-13 — communication-stop-child: row 12 exercised over real tmux

Shipped: `TestPeerStopStubbornChildStaysPending` — a real tmux pane whose writer traps TERM/HUP keeps the shutdown receipt pending through the stop timeout and clears it only after the writer is SIGKILLed. Onboarding row 12 closes; every row of the onboarding decision table now has direct coverage.
Verified: test green in ~0.5s, no leaked tmux session; `make close` ci-scoped next.
visual-review: n/a
Not done / debts: standing limits live in the onboarding plan (native matrix rerun, mobile/non-Linux recovery, PTY atomicity) and in `open/communication.md`.
Merge: fast-forward ready after this close.
