# 2026-09-23 — remove-ws-sessions: workspace card menu drops Sessions
Shipped: the workspace card's ⋯ menu no longer offers Sessions.
`workspaceRowMenu.js` loses the row and its `hasAgents` gate;
`WorkspaceMenu.jsx` drops `onSessions`/the icon; `App.jsx`/`Sidebar.jsx`
drop the callback chain. Sessions remain reachable from Agent CLIs, the
dashboard's top-sessions tiles and agents. `docs/architecture/routes.md`
names the menu as it ships.
Verified: `make ci-scoped` PASS (fmt,vet,hooks,test-js,build); web unit
tests 1021 pass (menu-shape test updated); visual review on a scratch
instance (`qa-scratch ws-sessions`): menu opened, overlay audit `ok:true`,
screenshot read — 9 items, no Sessions, nothing clipped; a real click on
Files still dispatches (`#/tree/w/<id>`).
visual-review: PASS
Not done / debts: none.
Merge: fast-forward ready.
