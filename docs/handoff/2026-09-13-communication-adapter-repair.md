# 2026-09-13 — communication-adapter-repair: row 9 recovers end to end

Shipped: `TestPeerOnboardingAdapterRepair` proves the row 9 repair loop — missing adapter reports `adapter-missing`, mints no capability; installing the adapter through Packages recovers on a later pass without re-selection; the pass is idempotent. Onboarding row 9 and the topic debt move to row 12 only.
Verified: `make close` ci-scoped next; test ran green with pipkg.UserDir overridden in-process.
visual-review: n/a
Not done / debts: row 12 (stubborn child) is the remaining onboarding partial.
Merge: fast-forward ready after this close.
