# 2026-09-23 — feat/wsl-mgmt-move: move the distro to another drive, back it up, hold it down

Items 4 and 5 of the owner's 8-item WSL Management list; third of four branches (next: disk history).

Shipped (64643d0d):
- System tab card "Where the distro lives": move the disk file to another drive, or back it up as one .vhdx; `picode-desktop places|move|backup`.
- Distro hold file (%LOCALAPPDATA%\PiCode\distro-hold.json, 30 s heartbeat, stale after 2 min), read by the shell's keepalive and server discovery (desktop-shell/src/hold.rs); used by move, backup and wsl-update. The PiCodeDistro task is disabled during the copy (it restarts on failure).

Verified live, read-only, on the owner's machine: disk file 227 GB on C:; C: (10 GB free) refused with the arithmetic; E: (251 GB free, not mounted by WSL) OK for E:\WSL\Ubuntu. Blind spot: move and backup themselves never ran live (an hour of stopped distro and ended sessions); covered by decision-table tests (relocateFlow order incl. task off/on; terminate fails, New-Item fails, copy ok + start fails) and stubbed QA.
Adversarial review, fixed: shell discovery booted the distro every 5 s mid-copy (now held); pause was shell-only (now a file the tool writes); 4 h kill on a move (no deadline now); WSL capability checked before stopping; backup/move folder collision (backup defaults to WSL\Backups, move refuses a non-empty folder); failure after a finished copy read as "not moved" (outcome.copied); same-drive move needs no space; reserved folder names.
Debts added to docs/handoff/open/windows-wsl.md: the Rust .wslconfig editor matches keys without their section; disk-compact should hold the distro and disable the task too.
visual-review: PASS (move/, 11 states) after three FAIL rounds.
Merge: on `main` at 0b690c1d, `make ci` green.

## Next up

- Owner decides whether to move the distro to E: (frees ~227 GB on C:, which has 10 GB left); needs `make deploy` + `make desktop-restart` first.
