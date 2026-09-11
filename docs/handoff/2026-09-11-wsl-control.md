# 2026-09-11 — feat/wsl-control: the tray and the CLI report the WSL disk

Shipped (fast-forward ready, not deployed): `picode disk [--json]`
(`internal/hostfs`, `cmd/picode/disk.go`) — one `df` + two `du` calls (two
because `du` skips a path it already visited: `~/.cache` came back without its
biggest child), a reclaim table labelled `safe`/`redownload`/`data`, the
top-level home breakdown, and what it could not read named as such. Read-only.
`picode-desktop disk [--json]` (`internal/desktop/disk.go`,
`cmd/picode-desktop/disk.go`) adds the Windows half — the VHDX path from
`HKCU\…\Lxss`, volume and file numbers from one JSON PowerShell call, the
sparse flag from `fsutil` (hex only; every word around it is translated) — and
the tray gains one line every five minutes (`WSL 218 GB · ≈92 GB held by
Windows · C: 27 GB free`, `— low` under 20 GB free). Nothing here deletes:
P2 (the Storage app and the Windows-facts route, whose ADR is not written yet)
and P3 (compact / `--set-sparse` / `--move`, which cost the distro's sessions)
are in `docs/plans/wsl-control.md`.

Verified: `make ci-scoped` PASS (fmt, vet, hooks, go[4], docs-site, vale) and
`make close`; live, `picode disk` (126 GB used, 41 GB reclaimable, 29 GB
unreadable) and the Windows half (218 GB, not sparse, 27 GB free, WSL 2.7.0.0
can `--set-sparse`), with the distro call run through the real parser. The
merge itself is covered by a fake runner; `make desktop` cross-compiles.

visual-review: n/a — no web surface changed; the tray menu cannot be
screenshotted from WSL. Its wording is unit-tested (`diskLine`); its rendering
is the owner's `make desktop-restart`.

Debts: the merged report needs the new `picode` deployed in the distro (today
it answers `unknown command "disk"`, which the error names); the warning is
words only — an alert icon needs a second generated asset.

Merge: fast-forward ready (main merged, `make close` run); the tray restart is
the owner's.
