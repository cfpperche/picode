# 2026-09-13 — feat/tab-agent-fetches: only an agent tab asks for an agent's role

Shipped: `isAgentTab` (`web/browser/src/lib/routes.js`) — a tab id is a tagged surface (`t:` `f:` `d:` `g:` `x:` `w:`) or a bare agent id — and the two fetches that assumed the latter: `fetchRoleState` and the composer's slash list. Selecting a file, folder, git or app tab used to ask `/api/agents/<tab id>/role-state` and `/slash`: two 404s per selection, caught and silent.
Verified: `make ci-scoped` PASS; 310 browser lib tests (1 new); scratch `tabagent` driven by agent-browser with the network buffer cleared between steps — a git tab produces **no** `role-state`/`slash` request at all, an agent tab still produces both with `200` (var/screenshots/tabfix-1-agent.png, tabfix-2-git.png).
visual-review: PASS (the two surfaces read; nothing about them changed on screen)
Not done / debts: none. The graph-tab 404 pair recorded in `docs/handoff/open/ui-chrome.md` is what this branch pays.
Merge: fast-forward ready
