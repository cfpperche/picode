# 2026-09-09 — feat/checklist-undefined: no more undefined/undefined

Shipped: sidebar disclosure uses `checklistProgress` (absent/unknown =
silence, never `undefined/undefined`). pi-checklist serializes POSTs and
skips a `blocked`/`absent` marker once this task already wrote a plan.

Verified: `make close` green. Scratch `checklist-undefined` at :8474 —
empty silence, real `1/3`, POST blocked → silence (not undefined),
expanded `2/3` list; overlayAudit ok.

visual-review: PASS (checklist-empty/step/absent/open.png; overlayAudit ok)

Not done / debts: production still shows the bug until deploy; the agent
must restart to pick up the pi-checklist race fix.

Merge: fast-forward ready (`5fca6f0a` + captures `25cf8626`).
