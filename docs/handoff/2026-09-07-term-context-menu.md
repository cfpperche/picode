# 2026-09-07 — feat/term-context-menu: the terminal pane gets PiCode's menu

Shipped: right-click in a pane opens a menu built from what that pane can do
(`web/desktop/src/lib/termMenu.js`, pure + 11 tests; actions in
`lib/termActions.js`): copy/paste/select all; `Ask <CLI> about this` and
`Attach files…` only while a launched CLI runs (ADR-0089 door); `Open <token>`
by the same rules as the Ctrl+click underline; go to the end, text size;
`Clear` (Ctrl-L) on a bare shell only; rename/settings/folder/close/remove
only for a terminal of its own, never an agent's TUI pane. The bypass modifier
(Shift) still hands the click to the browser; the right press no longer
reaches tmux. Popover → Radix DropdownMenu (roving focus, typeahead, submenu).
The attach bar now opens from the menu and closes on its X or Escape, seeded
with the selection (one line → message, longer → `selection.txt`).
Study: `docs/benchmarks/2026-09-07-terminal-context-menu.md`.

Verified: `make close` green. Live on scratch `ctxmenu` (:8471), real shell and
real `pi` CLI terminals: menu shape per pane type, Copy of the word under the
cursor, both Ask shapes, bar open/close with the pane at 32↔27 rows, Clear,
Open file, Open folder in Files, submenu, ArrowDown focus, Shift+right-click
not prevented and no menu. Two defects found and fixed there: the menu read
`state` while closed (blank desktop app), and tmux drew its own menu under ours.

visual-review: PASS (ctx-cli-menu, ctx-attach-open, ctx-attach-selection,
ctx-final; overlayAudit ok; card 5/5).

Debts: an agent's TUI pane (`kind: "agent"`) has unit tests only — no live
agent TUI ran in the scratch. Find-in-pane and "copy last output" stay
refused for v1 (dependency / shell integration).

Merge: fast-forward ready.
