# 2026-09-14 — chrome-tabs: Chrome CR23 editor tab strip

Shipped: editor tabs in `/browser` and `/desktop` (shared `AgentTabs` +
`app.css`) use Chrome CR23 language — raised active surface, hover pill,
idle separators, circular close. Overflow, keys and the all-tabs list
unchanged. Concave ears omitted: they only read as ears on a same-color
toolbar.

Verified: `make close` green; scratch `:8471` desktop + browser; light and
dark; overlayAudit ok.

visual-review: PASS (tabs-desktop-final.png, tabs-strip-hover-final.png,
tabs-strip-dark.png; overlayAudit ok; card 5/5)

Not done: none.

Merge: fast-forward ready after this note.

## Next up

(none — merged; bullet pruned 2026-09-14 to hold the board budget)
