# 2026-09-23 — feat/wsl-mgmt-restart-memory: Management System tab — memory, WSL version, restart and update

Owner asked for all 8 items of the WSL Management list; this branch ships items 1, 2 and 6 and is the first of four (next: wsl.conf + root caches via `wsl.exe -u root`, which the owner chose over a sudo password; move/export with space checks; disk history).

Shipped (a1ca58a4):
- System tab: memory on both sides with a swap warning, WSL version beside the newest GitHub release, Update WSL; Restart WSL; Config shows "Restart WSL to apply".
- Bounds: `timedRunner` 30 s reads / 2 min shutdown+start / 15 min update; keepalive paused during update; `--if-unreachable` recovery door that never overrides a busy answer.

Verified: read-only live probe with the new `picode-desktop host` — Windows 64 GB RAM, VM limit 31 GB, vmmemWSL 23 GB, swap 6 of 8 GB, WSL 2.7.0.0 vs newest 2.7.14. `wsl-restart` and `wsl-update` were never run live (a `--shutdown` ends every session on the owner's machine): covered by a decision-table test (restartFlow order, declined update shuts nothing down), `TestTimedRunnerBounds` and stubbed-Tauri QA.
Adversarial review fixed: declined update shut WSL down; missing timeouts; inbox-WSL usage text accepted; Windows memory read before the distro's; vmmem preferred over vmmemWSL; unbounded GitHub body (now LimitReader).
visual-review: PASS (sys/, 11 states) after three FAIL rounds.
Not done: GitHub "latest" can run ahead of what `wsl --update` installs (Store rollout); "Restart to apply" lasts one window session only; the page never offers `--force` past a busy answer, by design.
Merge: `make close` had not run when this note was written.

## Next up

- Owner tries Restart WSL (and Update to 2.7.14) from the System tab with no agent working, after `make deploy` + `make desktop-restart`.
