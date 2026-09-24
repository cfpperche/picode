# 2026-09-24 — feat/workspace-agents-summary: workspace agent summary

Implemented: the workspace overview summarizes agents by current work, with a routed full agent list, search, agent actions, and empty and error states (b971b54d1).
Verified: `make ci-scoped` and `make close` passed. Scratch UI review covered overview, full list, search, empty and error states; the event labels are readable after the visual fix, and overlay audit returned `ok: true`.
visual-review: PASS (overview, agent events, full list, search, empty and error screenshots).
No known debt. No deploy was performed.
