# 2026-09-11 — feat/cli-providers-pane: Providers live in the selected CLI's pane
Shipped: ADR-0103 amended 2026-09-11. A CLI page is Launch | Terminals | Sessions | Providers
(desktop + mobile). The catalog is the CLI picker; CliCombo is gone. Canonical hashes
`#/clis/<cli>/providers` and `#/clis/<cli>/providers/new`. Old `#/clis/providers*`,
`#/providers*` and `#/more/providers*` rewrite. OAuth returns to `#/clis/pi/providers`.
Non-Pi CLIs: blocked notice + Open Pi providers. Outer strip lost the Providers tab;
Settings/Packages/Messages stay. Fragment `docs/changelog.d/cli-providers-pane.md`.
Commits `63d831b8`, `6d38ea02`.
Verified: `make ci-scoped` PASS after merging main. Visual-review PASS on scratch
http://localhost:8472: Pi roster, Codex blocked, Add provider dialog, account •••
menu overlayAudit ok, mobile 390×844 Providers (stacked cards). Screenshots
`var/screenshots/cli-providers-pane/` (not committed).
visual-review: PASS
Not done / debts: Agent CLIs still not in docs-shots SURFACE_PROFILES (same gap as Canvas).
Merge: fast-forward ready (main merged as 8353bc7b).
