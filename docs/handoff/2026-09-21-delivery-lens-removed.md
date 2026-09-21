# 2026-09-21 — delivery-lens-removed: D2 comes back out of the product

Shipped: the reverse of D2 (ADR-0177). Delivery's Deployment lens, the
per-workspace observer binding, the deployment-receipt producer in
`internal/install`, the runtime-identity additions (`version.Revision`,
`/api/version`'s `revision`, the discovery file's `revision`/`boot`,
`server.BootID`) and the guard's coverage variant leave the product; the
Delivery view is one Integration lens again on desktop and phone, and the read
keeps D1's fields only. `git revert` of `88a54f8c` plus the doc corrections:
the guide's Deployment section, the architecture paragraph and the changelog
fragment go with it, and the D2 handoff note is amended rather than deleted.
Migration 065 keeps its number and its table with no reader — deleting the file
would let a future 065 be silently skipped on databases that already applied it.

Verified: `make ci-scoped`; the reverted tree builds (`go build ./...`) and the
affected Go packages pass; the JS suites and the reverted surface guard pass;
scratch `qa-scratch d2` shows Integration-only on desktop and phone (no lane
nav, no Environment selector), overlay audit `ok`. Blind spot: Linux headless
Chromium; the removal is verified by absence, so a leftover reading of the old
deployment API in a cached client is only covered by the build and the guard.

Not done / debts: `docs/handoff/open/delivery-flow.md` — the D1b list, unchanged
by this branch. The two deployment receipts the forced deploys wrote stay in
`<data>/var/delivery/`; nothing reads them.
Merge: fast-forward ready.
