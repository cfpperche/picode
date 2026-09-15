# 2026-09-14 — muse-agy-menu: Muse Code and Antigravity share the CLI ⋯ menu

Shipped: Owner-reported UI defect from the live instance — Muse Code and Antigravity had a standalone "Check for updates" button and no ⋯ menu, unlike every other CLI. The check moved into the same ⋯ menu every installed row uses (visibility now follows canUpdate||canCheckUpdate||canReinstall||uninstall), so the header row is Check setup | New terminal | ⋯ with 36px controls in both shells.
Antigravity got a real read-only update check: the vendor's per-platform release manifest from antigravity.google/cli/install.sh (manifests/<os>_<arch>[_musl].json, version field only; musl detected by /lib/libc.musl-*.so.1).
Verified: live on a scratch instance — agy latest 1.2.2 = installed, source channel, installMethod vendor; muse latest 1.2.1-R2847.1 = installed. Tests: clilifecycle TestParseChannelVersion, TestChannelURLFor, agy rows in TestForDecisionTable, agy DetectMethod; server test extended for both channel checks.
visual-review: PASS (var/screenshots/{muse-menu-final,agy-menu-final,codex-reference,agy-menu-mobile}.png; overlayAudit ok; card 5/5)
