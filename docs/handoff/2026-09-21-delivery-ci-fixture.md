# 2026-09-21 — delivery-ci-fixture: isolate Omp preview checks

Changed: the Omp trusted-extension conflict decision table supplies a temporary
executable fixture instead of requiring an Omp installation on the test host.
The preview never launches that fixture; product behavior and all five table
rows remain unchanged.

Verified: focused TestOmpTrustedExtensionConflictTable passed; make close passed
scoped Go checks, then reused their covered content after merging current main.
This validates preview decisions, not an installed Omp runtime.
visual-review: n/a (test-only change).
Not done: no deployment or runtime compatibility claim.
Merge: fast-forward ready; full CI belongs to the landing step.
