# 2026-09-25 — feat/md-preview: markdown files preview like GitHub
Shipped: one shared pipeline, web/shared/domain/mdPipeline.js (rehype-raw +
rehype-sanitize with GitHub's schema, then katex, alerts, heading ids);
mdDocument.js holds frontmatter, slugs, alerts and link/image resolution.
MarkdownDoc.jsx in browser and mobile, loaded as a lazy chunk. Shared CSS in
web/shared/styles/markdown-doc.css + highlight.css, now imported by all three apps,
so mobile chat code blocks get highlight colours for the first time.
New deps in @picode/shared: rehype-raw, rehype-sanitize.
Verified: `make close` green, main merged in. Scratch instance, three rounds:
r1 FAIL (desktop missing CSS import; scrollIntoView scrolled clipped ancestors and
hid the pane header; stuck outline highlight; double frame on code blocks; alert
spacing; repo SVGs broken since the blob API answers 415 for .svg, so SVG now goes
through the text route as a data: image; README link dead in the file tab);
r2 FAIL (mobile jump picked the wrong scroller; late outline entries never
highlighted); r3 PASS, overlay audit ok, visual card 5/5.
Screenshots: var/screenshots/md-preview/{,r2,r3}.
visual-review: PASS
Known, not fixed: Preview/Raw chips are 34px beside 36px Save/Close in the file
pane toolbar (pre-existing). An SVG with only a viewBox fills the column (GitHub
does the same).
Next, debts and the owner's pending ADR call on sanitized raw HTML:
docs/handoff/open/markdown-preview.md.
Merge: fast-forward ready.
