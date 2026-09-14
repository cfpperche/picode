# 2026-09-13 — peer-stop-child-race: the peer-stop test stops deciding by race

Shipped: `TestPeerStopStubbornChildStaysPending` is deterministic. A ready-marker handshake guarantees the pane installed its TERM/HUP traps before the stop signals it; every tmux command runs without the ambient `TMUX` and targets the session exactly (`list-panes -t =name`), so a test running inside a PiCode terminal no longer reads another pane. The red line in `docs/handoff/open/terminal.md` is paid; the `TestPaneRootSurvivesSIGHUP` load flake stays recorded there.
Verified: 20 consecutive green runs (`go test -count=20`), no leaked tmux session, neighbouring peer tests green; `make close` next.
visual-review: n/a
Not done / debts: none beyond the recorded pane-root load flake.
Merge: fast-forward ready after this close.
