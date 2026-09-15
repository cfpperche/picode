# 2026-09-15 — feat/retire-tray (ADR-0142 slice 4, code half)

Deleted the Go tray: tray_*, job_*, msgbox, icon*, mkicon,
startSupervised, KeepaliveArgs; systray dep removed via tidy.
`--tray` fails loud naming the repair; bare run prints usage, exit 2.
Install stages the shell (sibling or tag-pinned verified download)
and registers it; ps1 install requires --hidden.
PolicyIssues reworded to resident. Docs rewritten: windows-desktop,
architecture, desktop-v2 reversal, browser-extension, AGENTS/Makefile
rows; vale fix (executables→PiCode Desktop).

## Gates
ci-scoped PASS (full), windows vet, uncached Go tests green.

## Debts
Live migration needs owner at screen (UAC click + icon watch +
reboot test) per docs/plans/retire-go-tray.md.
Release recipe runs for real on next tag.
Host cargo test still broken (pre-existing).

## Next up
Run the migration, then deploy whenever owner wants main live.
