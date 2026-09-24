# 2026-09-24 — agent-history-loading: faster history and visible loading

Shipped: recorded session paths resolve directly before the existing listing fallback; Agent History now communicates loading, refresh, and retry states. The architecture note and changelog fragment describe the behavior.
Verified: `make ci-scoped` passed on the committed branch before merging main; `make close` after that merge reused the green gate. The Codex benchmark on 100 synthetic files measured `recorded_path` at 1,375 ns/op and `listing_fallback` at 1,718,830 ns/op.
Verified: visual-review PASS after the CSS fix. Loading, populated, and error screenshots are in `var/screenshots/`; the overlay audit reported `ok`. The populated state used a synthetic fixture, and the scratch instance was stopped.
visual-review: PASS
Not done / debts: real Windows latency remains unmeasured (`docs/handoff/open/agent-exits.md`). Legacy or moved session paths still use the broad listing fallback. No deploy was performed.
Merge: main can fast-forward after the close check.
