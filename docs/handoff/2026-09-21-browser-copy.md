# 2026-09-21 — feat/browser-copy: the agent rows describe driving the tab, not sites
Shipped: one line in `web/browser/src/components/BrowserPage.jsx` (commit b9d6f3f2) — the "Agent
permissions" section description said "Which agents and terminals may use the built-in browser, and on
which sites."; it now reads "Which agents and terminals may drive the built-in browser, and on what
terms." The rows themselves, the filters and the raw-protocol grant are untouched.
Why it was wrong: a site list no longer gates the session tab — ADR-0172 binds the split to the
principal that opened it and lets it drive any http(s) URL with no stored grant (the human's eyes and
the close button are the boundary), so the old line promised a permission the rows do not ask for.
The section's other copy already reads that way ("An agent opens a browser beside its session. Raw
protocol is the only extra grant.").
Verified: found by the visual pass of the landed session-browser slice; re-captured on scratch
`sesscopy` with `window.__picodeOverlayAudit()` ok — evidence
`var/screenshots/browser-agent-rows-copy-fixed.webp` (gitignored, not committed). Gate: `make close`
— `ci-scoped: PASS (fmt,vet,hooks,test-js,build; 1 path vs main)`. Blind spot: the copy was read on
that scratch instance, not in the owner's live Windows window.
visual-review: PASS
Merge: not fast-forward — main advanced 6 commits past 080753a1, the tip this branch already merged;
one file, one line against the merge base.
