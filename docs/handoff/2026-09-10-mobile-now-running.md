# 2026-09-10 — feat/mobile-now-running: drop the Running section from the mobile Now home

Shipped: `web/mobile/src/screens/Now.jsx` keeps Needs you / Recent results / Today;
the Running section (live terminals + running agents rows) is removed, with its now-unused
props (`running`, `liveTerms`, `workingIds`, `onOpenTerm`) dropped from the `Now` call in
`web/mobile/src/App.jsx`. `running` memo stays (still the `last`-agent fallback); `liveTerms`
const deleted. Desktop untouched.

Verified: `make ci-scoped` PASS (fmt, vet, hooks, test-js 54/54, build); scratch instance
qa-scratch `mobile-now-running` — mobile Now at 390×844 shows the three sections with their
empty states, DOM asserts no "Running" heading, `__picodeOverlayAudit` ok, Work tab still
lists agent/terminal with state chips.

visual-review: PASS (mobile-now-after.png + mobile-work-tab.png read; card 5/5)

Not done / debts: none for this diff. Now home no longer shows who is running — owner's
explicit product call (2026-09-10); revisit only if the owner asks it back.

Merge: fast-forward ready — `cd /home/goat/picode && git merge --ff-only feat/mobile-now-running && make ci`
