# 2026-09-25 — flaky-cli-restart: the restart flake now names what died

Not fixed — diagnosed further. `TestCLIRestartPreparationFailureAndWorkspaceCleanup` fails under the four-shard load
with "no server running" on the suite's private tmux socket. Ruled out: the restart handler (a failed prepare returns
400 before any kill). Not reproduced in 5 rounds of the four real shards nor 42 isolated runs. The failure now prints
`privateTmuxReport()`; its first capture showed the socket file present while the server was gone — an abnormal exit
(signal or crash). Leads and the next step (tmux -vv on the private server, or strace for kill) are in
`docs/handoff/open/terminal.md`. Verified: the test passes 3/3 alone; the report compiles and printed on the flake.
