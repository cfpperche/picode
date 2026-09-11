# 2026-09-11 — feat/wsl-review: adversarial pass over the WSL disk branch

Shipped: fixes to feat/wsl-control before anyone has run it. **The tooltip
warning was being erased within one health-probe interval** — `diskTick` wrote
the tooltip with the warn suffix, `setStatus` (5 s poll) rewrote it without;
the warn flag now lives on the tray and `refreshTooltip` is the single writer
(tray_windows.go). `distroUsed` no longer spawns `wsl.exe` when the Windows
half already failed. Two false claims fixed — `picode disk --json` is read by
`picode-desktop disk`, not by the tray (usage string + guide) — and the guide
now says measuring starts a stopped distro. Plan notes the P0 deviation
(dedicated `disk` command, not `doctor --disk`).

Verified: go vet + tests (hostfs, desktop, cmd/picode-desktop), Windows
cross-compile, `make ci-scoped` (docs + vale); `make close`.

visual-review: n/a — tray menu only, unscreenshottable from WSL; tooltip
composition is single-path by construction, wording still unit-tested in
`diskLine`.

Debts: unchanged from the wsl-control note (merged report needs the deploy;
warning is words only). The codex-resume one-off repair id dropped out of
`docs/handoff.md` in the 8 KB squeeze — recoverable from its git history.

Merge: fast-forward ready; owner decides the tray restart.
