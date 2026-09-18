# 2026-09-18 — distro-keepalive: the keepalive outlives the shell (ADR-0155)

Shipped: the day's mass session deaths were never the deploy — all ~15
deploys finished with "all N session(s) still alive". The killer was
`make desktop-restart`: taskkill of the resident ended its job-object
keepalive, the distro had no Windows-side client left, and WSL's idle
reclaim terminated the instance (journal: InitTerminateInstanceInternal →
reboot(RB_POWER_OFF); fully reconstructed for 18:05:05). Fix: the
scheduled task `PiCodeDistro` now owns the distro lifetime
(`wsl.exe [-d <distro>] --exec /bin/sleep infinity`, logon trigger,
no time limit, IgnoreNew, restart-on-failure). The shell ensures it on
every health poll (register when the action differs, `schtasks /run`
otherwise; child-spawn kept as fallback); `desktop-swap.sh` ensures it
BEFORE the taskkill. Registration uses PowerShell cmdlets — schtasks
/create cannot (onlogon needs admin, /sd is locale-brittle). The live
machine has run the task since 18:39 local; registered, Running.
Also: `make adr` now survives a pruned worktree directory (xargs 123
under pipefail).
Verified: cargo xwin check clean; ci-scoped PASS (full); the here-doc
ensure executed live against the real task store.

## Next up

- Owner runs `make deploy` + `make desktop-restart` from the root to ship
  the shell half; the task already protects the current machine.

## Debts

- docs/handoff/open/wsl-keepalive.md: attribute the 16:44/17:06
  terminations (sparseVhd suspect — a keepalive cannot cure that), and
  the Rust register/run paths are compile-checked only.
