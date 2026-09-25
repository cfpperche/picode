# 2026-09-25 — feat/md-mobile-live: Live and Split for markdown on the phone
Shipped: the phone's Files screen offers Preview · Live · Split · Edit for markdown.
Split stacks in portrait and goes side by side at >=700px, synced against
`.m-file-preview`. The choice shares the desktop key `picode-md-view` (the phone's
Edit is the desktop's "raw").
Moved: the Live editor and scroll sync went from web/browser/src/lib to
web/shared/editor/ (mdLive, mdLivePlan, mdLiveTables, mdLiveMath, and the new
React-free splitSync.js with `attachSplitSync`); the desktop keeps a thin
useSplitSync hook. Shared gained @codemirror/state, view, language, katex and
mermaid (@codemirror/lang-markdown as a dev dep); its test script now runs
editor/*.test.js. Live CSS moved to web/shared/styles/md-live.css, imported by the
browser, desktop and mobile index.css; selectors use `:is(.file-cm, .m-file-editor)`
so each rule keeps its weight. The broken-diagram text no longer says "Click".
Verified on the scratch, one round: at 390x844 the four buttons sit in one row,
36px each; Live renders everything, the wide table scrolls inside itself, document
and `.cm-scroller` measure 390/390; portrait Split syncs both ways with stable
readings, landscape is side by side; the last choice survives reload; dark theme
via the real Settings; desktop Live unchanged and Split syncs; no console errors,
overlay audit ok, visual card 5/5. Screenshots: var/screenshots/md-mobile/.
Minors, not fixed (first two already debts in the topic): pipes look like `/` in
source views and fenced code is unhighlighted in the editor; Live→Split re-measure
jump; frontmatter, kbd and details stay raw in Live (predates this branch). Links
in Live need Ctrl/⌘+click, which touch lacks (already a Next item).
visual-review: PASS
Landed ae962a8d8 (Next/debts: docs/handoff/open/markdown-preview.md). Merge gate: 1st `make ci` failed on clisession TestWriteTextToolsModeAndNeverOverwrites (passes 10/10 isolated; swaps the global session.TestRoot — load flake); 2nd failed because the lockfile change had left web/node_modules without vite (concurrent `npm ci` in the root); one `npm ci`, then green.
