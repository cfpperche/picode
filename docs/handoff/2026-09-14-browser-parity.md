# 2026-09-14 — browser-parity

Owner's parity mandate: Settings ▸ Browser must look like the ChatGPT
Work reference (cards/rows, switches, Manage dialogs) before covering
every option. This increment is the layout + the master switch.

## Done

- `.set-page` shape in web/browser/src/styles/app.css (title/lede,
  section headers, `.set-panel` + `.set-item` rows with hairline
  dividers, `.set-btn`, `.set-select`, `.set-hist`).
- BrowserPage.jsx rebuilt on that shape; history moved into a dialog
  with Remove/Clear; Browsing data keeps the two-step clear.
- `browser.agentAccess` pref (default on) + `browserAgentAccess()` gate
  in the agent route — the Browser switch off refuses every verb.
- Visual: scratch screenshots read (parity-1/2); root edits moved into
  this worktree before committing (contract §5).
