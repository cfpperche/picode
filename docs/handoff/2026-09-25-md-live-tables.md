# 2026-09-25 — feat/md-live-tables: GFM tables render in the Live view
Shipped: each table no selection touches is a block widget from a StateField (focus
via `EditorView.focusChangeEffect`): header, column alignment from the delimiter row,
inline code/bold/italic/strike/links built as DOM nodes (never parsed HTML); a table
in a blockquote keeps the quote bar. A click puts the cursor at the start of the
clicked cell and the source returns (positions re-read at click time); Up/Down stop
on the table's near row (`tableSkip`) instead of stepping over it; Ctrl/⌘+click
follows a cell link. `contain: inline-size` makes a wide table scroll inside itself.
Files: web/browser/src/lib/mdLiveTables.js (5 tests), the table part of mdLive.js,
`.cm-md-table-widget` CSS in web/browser/src/styles/app.css.
Verified on the scratch, three rounds:
r1 FAIL (399px pane: wide table one letter per column, `.cm-lineWrapping` let cells
wrap anywhere; ArrowUp/Down jumped over every table; minors: quote table lost its
bar, zebra rows invisible);
r2 FAIL (words whole, but the table's min-width widened `.cm-content` to 745px in the
399px pane, prose cut at the pane edge; arrows, quote bar, zebra and 1920 passed);
r3 PASS: `.cm-scroller` 399/399 at every scroll position, widget 713/367 and scrolls;
no console errors, overlay audit ok, visual card 5/5. Caveat: headless hides
scrollbars, so the widget's scrollbar was not seen. Screenshots: var/screenshots/md-tables/.
visual-review: PASS
Not done: a cell click lands at the start of the cell, not at the clicked character.
Gates: `make close` green, main merged in, fast-forward ready.
Next and debts: docs/handoff/open/markdown-preview.md.
