# 2026-09-22 — feat/tmux-empty-server: "no current target" is an absent session

Found by the CI run that verified the vendor-test repair: `TestPaneCwdIsRightFromTheFirstInstant` (`internal/tmux`) failed on ubuntu with `NewSession: tmux has-session "picode-born-7": no current target`.
The failing call is `NewSessionEnvSize`'s pre-check, so no session was lost — but the pre-check read tmux's *"no current target"* as a real failure. That message is what a live server holding **no sessions at all** prints: the window between the last session's death and the server's own exit, which the test's twelve create/kill rounds walk straight into, and which a user hits by closing a terminal and opening one right after.
Fix: `hasSession`'s `notThere` list gains the message (same family as `can't find session` / `no server running`), so the create proceeds and tmux starts a server.
Proof: the new row in `TestStartupCommandsRetryTheServerStartRace` fails on the old list with the CI's own words (`got has=false err=tmux has-session "x": no current target`) and passes on this one; `go test ./internal/tmux` green against real tmux; `make ci-scoped` PASS (fmt,vet,hooks,go[8]; 2 paths vs main).
Fragment: `docs/changelog.d/tmux-empty-server.md` — `### Fixed`, because a refused terminal creation is user-visible.
Not done: the window's other half (two clients racing to start the first server) is already covered by `serverStartRace`'s retry; this is proven by the deterministic row, not by a runner.
Blind spot: it does not reproduce locally on demand — ten repetitions were green before the fix — so the runner is still the only place the original race shows.
