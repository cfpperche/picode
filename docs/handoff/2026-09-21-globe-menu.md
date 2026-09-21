# 2026-09-21 — globe-menu: header globe launcher

Shipped: header globe click binds a split to the selected agent or agent-CLI
terminal (same door as the pane's Open browser). Shift+click and the
right-click **Open in new tab** mint a tab with no agent. Right-click is
this menu, not Copy/Reload. Context menus stack above `.shell-row` (z-index
65). Policy is `web/browser/src/lib/globeMenu.js` (click table + menu rows).

Verified: `make close` green. `globeMenu.test.js` covers bind × shift ×
split (8 click rows) and both menu shapes. Scratch `globe-menu` on
http://localhost:8473: Atlas right-click showed Open browser + Open in new
tab; Open in new tab landed on `#/web/1`; globe click on Atlas set
`agent-split-on`. overlayAudit ok on both menus. Not run inside the Windows
shell (no native WebView2 split).

visual-review: PASS (globe-menu-agent.png + globe-menu-newtab.png +
overlayAudit ok; card 5/5)

Not done: globe still lives in `#main-tabs`, which is `hidden` while no tab
is open (pre-existing).

Merge: fast-forward ready (`feat/globe-menu`).
