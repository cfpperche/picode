# 2026-09-21 — feat/terminal-attach-theme: attach bar follows terminal theme, defaults move to user menu
Shipped: `.term-attach` re-points its tokens under
`:root[data-term-theme]` (both directions, values mirror theme.css;
accent stays the app's); gear removed from the Terminals header;
`termset` row in the user menu (PiCode group, searchable, `go()`
needs no routing change); `userMenuModel.test.js` +1 case.
Verified: `make ci-scoped` PASS; `make web` ok; scratch live: dark card
under app-light+term-dark, light card under app-dark+term-light (both
screenshots read), gear gone from the header, menu row navigates to
`#/termset`, theme flip via the real UI moves the dataset live,
overlayAudit ok. Out of scope: `.term-find` bar, mobile sheet.
visual-review: PASS (attach-dark-card + attach-light-card +
terminals-no-gear read; card 5/5; audit ok)
Not done: none.
Merge: fast-forward ready.
