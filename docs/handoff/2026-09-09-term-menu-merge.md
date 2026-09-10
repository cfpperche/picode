# 2026-09-09 — feat/term-menu-merge: one terminal menu for sidebar and Agent CLIs

Shipped: `web/desktop/src/lib/termRowMenu.js` is the single contract for the "…" menu on an Agent CLI terminal row; `WorkspaceRows.jsx` (sidebar) and `AgentClis.jsx` (CLIs list) both render it. Rows: Rename…, Launch settings (`#/clis/terminal/<id>`), Terminal settings (`#/termset/<id>`), Start terminal (stopped) or Restart+Stop (running), Remove terminal. App.jsx gained `launchTerminalAction` (same `/api/terminals/<id>/launch/<op>` the CLIs list uses, confirm on stop/restart); AgentClis receives `onRenameTerm` (App's `renameTerminal`).
Verified: `make ci-scoped` PASS (fmt,vet,hooks,test-js,build); desktop `npm test` 284/284; scratch instance (qa-scratch term-menu-merge): both menus screenshotted in running and stopped states, overlayAudit ok, rename round trip, stop→confirm→stopped row→Start navigates, both settings navigations.
visual-review: PASS (term-menu-sidebar-running/stopped.png, term-menu-clis-running.png; card 5/5)
Not done / debts: web/mobile terminal rows still only offer Remove (same merge not ported); sidebar Remove still uses `DELETE /api/terminals/<id>` while the CLIs list uses `/launch/remove` — same outcome, two paths.
Merge: fast-forward ready (`git merge --ff-only feat/term-menu-merge && make ci` from the root).
