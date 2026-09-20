# 2026-09-17 — webview-first-frame

The owner's report after the terminal-link fix: Ctrl+click worked, "but at the
moment of the click the layout breaks and then fixes itself".

## What was wrong

`openWebTab(url)` invokes `btab_navigate` immediately — before React mounts the
pane that measures the page region — so the shell's `ensure()` found no pending
bounds and created the child webview at its placeholder rect (120,120,900,640),
visible. A moment later the pane's `btab_bounds` landed and it jumped into
place. The tab strip's own globe button never showed it because an empty tab has
no URL to navigate and the webview is only created by a navigate.

## Fixed (shell, so every creation path is covered)

`ensure()` records a tab as **unplaced** when it had no reported rect, hides it,
and the first `btab_bounds` places it and shows it. An explicit
`btab_visibility` (or closing the tab) clears the flag and wins — the table is
in the code. Popups (`btab://new`) and the split pane's recreations take the
same path, which is why the fix belongs here and not in one JS caller.

Compiles (`cargo xwin build`); ACL unchanged.

## Not verified

The first frame itself: with the swapped exe, Ctrl+click on a link must open a
work-browser tab with no grey rectangle and no jump. The owner's gesture is the
test — nobody has seen it yet.
