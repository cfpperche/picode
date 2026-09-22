# 2026-09-22 — feat/ade-copy-tail

Shipped: the copy tail, the last planned branch of `docs/handoff/open/multi-cli-ade.md` (ADR-0179). With it all five planned branches have landed; only that file's debts remain.

Docs: the docs-site configuration cluster (Configure, Settings, Providers, MCP, Integrations) covers all nine CLIs and names Pi only where a feature is Pi's; Backup says only Pi's files are copied. `docs/guidelines.md` "Related pi documentation" → "Related vendor documentation", and `docs/benchmarks.md` "Vendor correlation". `docs/philosophy.md` modularity table without Pi-only primitives. The `docs/architecture.md` component diagram gains the Agent CLIs stack and marks the RPC channel Pi-only. CONTRIBUTING and docs README wording.

Web: `go()` in `web/browser/src/lib/routes.js` takes `extra.cli`; App passes `agent?.cli` from the user menu and the palette (Pi when no agent is selected); new rows in `routes.test.js`. Label fixes in both apps (palette "CLI settings", Browser/Computer identity notes, "Open Agent CLIs", start automations "create a Pi agent").

Server: the never-returned `errAgentCmdMissing` removed; `AgentCmd` comments say Pi-only.

Verified: route and menu tests 33/33; `make ci-scoped` PASS. Visual spot check on a scratch instance: the first palette capture did not show the typed filter (evidence FAIL); recaptured with real keystrokes → a single "CLI settings" row, overlay audit ok. The Computer page shows "A CLI started outside PiCode…".
visual-review: PASS (v2-palette-settings.png, computer-page.png; overlayAudit ok; card 5/5)

Deliberately not changed:
- Mobile search aliases still open Pi's panes: mobile has no selected agent. Debt in the topic file.
- The packages watch still scans `~/.pi`: its test depends on that dir. Debt in the topic file.
- `notice.js` keeps `cli || "pi"`: a blank cli is a legacy Pi row.
