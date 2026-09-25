# 2026-09-25 — feat/md-live-math: Live renders math and Mermaid
Shipped: `$$` block math, inline `$…$` math and ```mermaid diagrams render in Live.
mdLiveMath.js finds them outside code, raw HTML and link targets with pandoc's
dollar rule. One block field draws tables, math blocks and diagrams; Up/Down stop
at any of them. Inline math is a per-line widget that owns its text. KaTeX and
Mermaid load on first use; output and failures are cached by source; diagrams
redraw on a theme switch (data-theme MutationObserver → effect); hover ring on blocks.
Fixed beyond Live: every Mermaid render (Live, Preview's MermaidPreview, chat
MermaidBlock, both apps) passes `suppressErrorRendering: true` and removes its
`d<id>` scratch node on failure — a broken diagram used to leave a 109px error
graphic in <body>, piling up per re-render and shifting the app (seen at 1000px).
The shared pipeline adds `remarkPandocDollars`, so Preview keeps "$5 and $10" as
text like Live (remark-math made it math; from the md-preview branch).
Files: web/browser/src/lib/mdLiveMath.js (3 tests), mdLive.js, mdPipeline.js,
FilePreview.jsx and MermaidBlock.jsx in both apps, app.css.
Verified on the scratch, three rounds:
r1 FAIL (Mermaid body leak; minors: stale diagram after theme switch, uneven
alignment, KaTeX error red 3.2:1 in dark, revealed `$$` not monospace, stripe on
the code-block cursor line, no hover cue);
r2 all six checks PASS, leaks 0 in Live and Preview; found the Preview price defect;
r3 PASS: prices are text in Preview, Split and Live; math still renders; leaks 0;
overlay audit ok, visual card 5/5. Screenshots: var/screenshots/md-math/.
visual-review: PASS
Gates: `make close` green, main merged in, fast-forward ready.
Next and debts: docs/handoff/open/markdown-preview.md.
