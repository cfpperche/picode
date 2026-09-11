# 2026-09-11 — feat/wsl-actions: the compact action, tray and CLI

Shipped (fast-forward ready, tray swap still the owner's): the held-space
action. `picode-desktop disk-compact` (`internal/desktop/compact.go`,
`cmd/picode-desktop/compact.go`) + tray item **Give back ≈N GB…** (greyed with
a reason when there is nothing, someone is mid-compact, or the WSL build cannot
convert). Flow: readiness interlock (`GET /api/deploy/readiness` — refused
without an answer, `--force` overrides on the CLI) → confirmation dialog naming
the cost → `wsl --terminate` → `--set-sparse true` (no elevation; Optimize-VHD
is CLI-from-admin-terminal only, because `elevate()` re-uses the tray's own
args and would spawn a second admin tray) → distro restarted → **keepalive
re-armed** (the terminate kills it; nothing else restarts it) → before/after
measured on the file. `--dry-run` stops nothing.

Verified live: `disk-compact --dry-run` (218 GB, ≈91 GB held, sparse plan) and
the interlock refusing for real (`still working: terminal desktop; terminal
scrollbar`). Unit-pinned: the argv sequence, **restart even when the conversion
fails**, Optimize-VHD LiteralPath quoting, unknown-method refusal, PlanCompact
fallback. Decision table as implemented lives in `docs/plans/wsl-control.md`;
guide + architecture updated.

visual-review: n/a — tray menu and dialogs are Windows surfaces not
capturable from WSL; flow logic is in `internal/desktop` (unit-tested), tray
code is thin wiring.

Debts: Optimize-VHD from the tray needs an elevation design that does not
relaunch the tray as admin; cache prunes as reviewed actions are still P2 (with
its ADR). The confirm dialog is MessageBoxW — plain and topmost, no theming.

Merge: fast-forward ready; owner decides the tray restart.
