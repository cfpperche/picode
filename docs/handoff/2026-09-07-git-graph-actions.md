# 2026-09-07 — feat/git-graph-actions: the graph answers a right-click

ADR-0096 (accepted with the owner the same day) closes the four Refuse rows
that kept the graph read-only by wiring it to the three doors ADR-0078 already
shipped. **Phase 1 only: nothing here runs git.**

Server: `loadRefs` gained `%(upstream:short) %(upstream:track,nobracket)
%(worktreepath)` in the call it already made — measured 9-22 ms either way —
plus one cheap exec each for `--merged HEAD` and `git remote` (7 ms); graph
load stayed ~0.15 s. `canonicalPath("")` resolves to the process cwd, so an
unchecked-out branch is guarded to an empty path.

Client: `shared/domain/graphActions.js` (pure, 19 tests) decides the menu;
`ContextMenu.jsx` renders it — the one PiCode menu, not a second. The
differentiating row: a branch held by a sibling worktree offers **Open
\<agent\>** instead of a checkout git refuses. Pills are spans inside the
row's button, so the row's menu carries each as a submenu (WCAG 2.1.1).
`gitActionCommand`/`gitActions`/`askGitPrompt` were byte-identical in desktop
and mobile; both now re-export `shared/domain/gitCommands.js`.

Deliberate deviation: the plan's "command preview with no send button" is
**not** built — rows that show a command they cannot run are the dead chrome
`uiux-review` calls FAIL. The targeted composer stays the phase-2 substrate.

visual-review: PASS (gg-menu-worktree, -branch-elsewhere, -open-agent,
-commit-submenu, -remote-dark; overlayAudit ok, light and dark; card 5/5).
An `ok:false` was chased down to a synthetic contextmenu on an off-screen
pill, not the product.

Debts: the busy line is unit-tested but never rendered live (needs a real
streaming agent); mobile has no long-press sheet yet, though the module is
shared and ready.
