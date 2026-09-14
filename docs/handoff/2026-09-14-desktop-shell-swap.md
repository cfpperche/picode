# 2026-09-14 — desktop-shell-swap

One-debt branch: `make desktop-restart` now covers the v2 shell.

## Done

- `desktop-restart` builds `desktop-shell` (cargo xwin) next to the tray, and
  `scripts/desktop-swap.sh` stops → copies → relaunches `picode-shell.exe`
  when it was running, via a detached Windows `start` (the shell carries no
  keepalive duty; the tray owns the VM lifetime — the 2026-09-02 rule is
  about WSL-backgrounded exes and stays intact).
- `DRY_RUN=1` prints the whole plan without stopping, copying or starting
  anything; the read-only probes (tasklist, username) still run.
- Verified: `bash -n`, cold `make desktop` + `make desktop-shell` in this
  worktree (2m42s), full `DRY_RUN=1` pass with the owner's shell live —
  stop/copy/relaunch branches all printed, nothing touched.
- This quites the debt recorded in `docs/handoff/open/work-browser-tabs.md`
  (the manual shell swap that masked the CDP bridge behind a Tauri ACL
  refusal for a day).

## Next up

- Owner: next `make desktop-restart` is the live acceptance (watch it swap
  the shell; the app blinks once).
