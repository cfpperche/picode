# 2026-09-11 — feat/cli-detail-panes: Sessions live in the selected CLI's pane
Shipped: ADR-0079 amended 2026-09-11. A CLI page is Launch | Terminals | Sessions
(desktop + mobile). The catalog is the CLI picker; switching CLI keeps the pane.
Canonical hashes `#/clis/<cli>`, `#/clis/<cli>/terminals`,
`#/clis/<cli>/sessions[/wsId]`. Old `#/clis/sessions*` and `#/sessions*` rewrite.
Outer strip lost the Sessions tab. TopSessions names the CLI in `sessionsHash`.
Fragment `docs/changelog.d/cli-detail-panes.md`. Commits `b97ecddc`, `5831801d`.
Verified: `make ci-scoped` PASS after merging main. Visual-review PASS on scratch
http://localhost:8473: Launch/Terminals/Sessions panes, empty sessions (one line +
New terminal), Codex catalog switch kept Sessions pane, legacy redirect
`#/clis/sessions?cli=grok` → `#/clis/grok/sessions`, Customize + lifecycle menu
overlayAudit ok, mobile 390×844 Launch (Customize on tab row) and Sessions empty.
Screenshots `var/screenshots/cli-detail-panes/` (not committed).
visual-review: PASS
Not done / debts: Agent CLIs still not in docs-shots SURFACE_PROFILES (same gap
as Canvas). Isolated scratch had no real session files (empty states only).
Outer strip still clips "Messages" to "Message" on 390px — pre-existing.
Merge: not fast-forward ready (main moved to da038181 after close; merge main, then `make close`).
