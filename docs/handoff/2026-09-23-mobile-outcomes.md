# 2026-09-23 — feat/mobile-outcomes: Outcomes screen under More on mobile

Shipped (commit caf710abc):
- Mobile Outcomes screen at #/more/outcomes (alias #/outcomes) lists removed agents with answer badge, CLI, lifetime, cost, time
- Row detail: outcome, note, lived, cost, setup, workspace; Answer / Change answer (inline chip editor), Delete record
- "Ask when removing agents" switch at top; empty state one line + "How outcomes work"
- More menu model and mobile router (mobileRoutes.js, moreMenuModel.js) gained "outcomes" section; router's fixed list initially ignored new section, found in first scratch capture and fixed with tests

Verified: JS tests pass. Visual review on scratch at 390px (empty dark, list dark/light, detail and editor light) — no horizontal overflow, PASS. make close PASS. Docs updated: docs/architecture/agent-exits.md, docs-site/guide/outcomes.md, changelog fragment. Debt paid in docs/handoff/open/agent-exits.md.
visual-review: PASS
Not done / debts: tested on browser viewport (390px) not real phone hardware.
Merge: fast-forward ready.
