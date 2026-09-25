# 2026-09-25 — restart-flake

`TestCLIRestartPreparationFailureAndWorkspaceCleanup` ("no server running",
three times in `make close`, never alone): most likely cause found and closed — the rename of the
launch folder raced `/bin/sh` opening `launch.sh`; the dead pane was the only
session on the suite's private tmux server, so the server exited. Test-only
fix: wait for the launch banner first (the mechanism reproduces the exact message on a private socket; the race itself was not caught in vivo); `paneAlive` reports a dead pane's
status and lines. Ruled out on the way: the sh-open race alone at shell level
(120/120 alive), CPU load on the isolated test (75/75), and kill/has-session
(exact `=name` targets).
