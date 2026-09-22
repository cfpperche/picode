# 2026-09-22 — feat/packages-scope-context: every packages pane names its workspace and agent
Shipped: the vendor drivers' scope radio (internal/pkgs/guest.go, scopesForGuestContext) now reads
the context the read carried — the workspace scope's label is the workspace's own name and the
agent scope's note spells the agent out — and a layer the read cannot answer is not offered at
all (a project mutation without a folder refuses anyway). Pi's pane already did this
(scopesForContext); the eight guest CLIs ignored q.WorkspaceName, which read as a generic "This
workspace" on every non-pi pane (owner report, Omp).
Verified: decision-table test on the helper (TestGuestScopesNameTheContextOrStepAside) and the
server read updated to the new contract (unbound read offers machine only; a bound read names
workspace + agent — packages_vendor_test.go). Visual on the qa-scratch instance
(scopectx, Omp ▸ Packages bound to the seeded workspace): the radio reads "Global | QA";
overlayAudit ok, rows on the 36px rhythm.
visual-review: PASS (scopectx-omp-packages.png + overlayAudit ok; card 5/5)
Merge: fast-forward ready.

## Debts

- Todo plans over 50 tasks clip to the store cap (carried from feat/omp-checklist-pkg; unchanged).
