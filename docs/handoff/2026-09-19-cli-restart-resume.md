# 2026-09-19 — cli-restart-resume: Restart reopens the pinned CLI session
Shipped: confirmed Restart on an Agent CLI pins, then prepares the next
generation with that session's verified resume recipe (ADR-0158, amends
0069/0084). `launchWithPinnedSession` is shared with `start?resume=true`
and peer reconnect. No pin → current settings, fresh chat. Stop then
Start still starts fresh. Daemon reconnect still does not relaunch.
Verified: `make close` green; `TestLaunchWithPinnedSession` plus
`TestCLITerminalRestartResumesPinnedSession` (real tmux, argv `--resume
sess-1`, process replaced). Blind spot: not clicked on a live Claude
Code pane after this binary is running.
visual-review: n/a (copy-only on existing confirm/tooltip)
Not done: none.
Merge: catch main, then fast-forward from the root.
