# 2026-09-24 — feat/shell-activity: a CLI's running shell command shows as Running

Shipped (ADR-0212, ccaf78181):
- Every 3 s the server reads the pane's process tree for a running shell command (a `!` command such as `!make deploy`, or a background job): a child detached into its own session, or `sh -c` on the tty for Hermes, at least 2 s old; MCP/LSP helpers excluded. Published as ephemeral `terminal.command`, never a hook state or a turn.
- UI: "Running" with tooltip "Running <name>" on the desktop sidebar row and the mobile Work list.
- Measured live 2026-09-24: `!` fires no start hook in Codex 0.156.1 (no hook at all), Claude Code 2.1.281 (Stop only at the end), Grok (per its docs) or Hermes. Tree shape measured for Claude, Codex, Grok, Hermes, OpenCode, Pi and Omp.

Verified: `make ci-scoped` PASS; `make close` PASS (exit 0). Go decision table `TestRunningCommandDecisionTable` and live Linux test `TestRunningCommandSeesDetachedChildLive`; JS tests in `terminalCli.test.js` and `feedReducers.test.js`.
visual-review: PASS on a scratch instance with Codex `!sleep 90`: desktop sidebar "Running" (title "Running sleep"); the API command appeared 3.3–5.6 s after Enter and cleared ~2.3 s after exit; mobile Work list "Running · now"; overlayAudit ok. Screenshots in `var/screenshots/shell-activity/` (not committed).
Unverified: the header chip on the mobile agent screen. That header does not render for a running CLI agent (TerminalScreen has no chip).

Open: `docs/handoff/open/terminal-activity.md` (re-measure pane titles; Muse/agy have no lease; Omp builtins run in-process; the 280px sidebar hides spinner and age).
Not deployed (owner's call). Merge: main moved; merge main and rerun `make close`, then fast-forward.
