# 2026-09-24 · feat/terminal-activity-2

Follow-up to ADR-0212 (feat/shell-activity), from the owner's "do what you recommend".

- **Sidebar:** at 280px (the default) a Working or Running pill now keeps its spinner and drops only the age (`web/browser/src/styles/app.css`). Measured at 280px: the spinner costs the name about 16px ("shell activi…" instead of "shell activity…"). The full name stays in the tooltip, and the ⋯ menu and pill edges are unchanged.
- **Docs:** `docs/architecture/direct-session-communication.md` now has the 2026-09-24 pane-title measurement. Claude Code, Codex and Grok show busy/idle in the title; the 2026-09-14 "none do" text is corrected.
- **Omp builtins not done, by measurement:** `user_bash` in omp 18.2 fires only at the start. There is no end event, and the command runs after the handler returns, so reporting working would leave the agent stuck. The debt stays open in `docs/handoff/open/terminal-activity.md`.
- **Gates:** `make ci-scoped` PASS, `make docs-check` ok, `make close` PASS.
- **visual-review:** PASS (`var/screenshots/terminal-activity-2/running-280.png`, overlayAudit ok, card 5/5).
- **Not deployed.**
