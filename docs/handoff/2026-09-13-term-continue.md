# 2026-09-13 — feat/term-continue: Continue in… on terminal menus
Shipped: **Continue in…** on sidebar ⋯, Agent CLIs Terminals ⋯, and pane right-click. `sessionFromTerminal` + lastSession pin; feed `terminal.last_session`. Study `docs/benchmarks/2026-09-13-ai-memory.md` (wiki brief ≠ native Continue). No new ADR. Fragment `docs/changelog.d/term-continue.md` already present.
Verified: `make close` PASS (ci-scoped fmt,vet,hooks,go[18],test-js,build,docs); scratch `qa-scratch term-continue` :8474; overlayAudit ok; shots `var/screenshots/term-continue-*.png` (sidebar running/stopped/shell, clis right-edge, pane, dialog blocked).
visual-review: PASS (card 5/5)
Not done / debts: mobile rows still only Remove (`open/terminal.md`). Did not e2e a real native handoff from a live pin (dialog path reused ADR-0088 tests; fake pin preview refused missing file).
Merge: fast-forward ready from root (`git merge --ff-only feat/term-continue && make ci`). Deploy is the owner's call.
