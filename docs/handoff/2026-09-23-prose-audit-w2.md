# 2026-09-23 — prose-audit-w2: wave 2 of the architecture prose audit
Shipped: wave 2 (Agent CLIs stack). cli-terminal-launch.md carried all
five drifts: two dead `POST /api/clis/<cli>/terminals` mentions (now the
handoff door / `POST /api/agents`, ADR-0184), the npm-install enumeration
missing omp and opencode (`ForMissing`), the `surface: terminal`
description of Muse/Antigravity (full rows since launch-parity), and the
pane-tab paragraph missing Memory (ADR-0163, six CLIs) and Models
(ADR-0181, omp only). cli-providers, cli-settings, cli-memory,
cli-session-handoff, packages, mcp and integrations verified clean by
targeted sweeps (every cited path exists, all 17 cited ADRs exist,
constants match: 150ms resolve retry, 4MB drop cap, delivery receipts).
Verified: `make ci-scoped` PASS; each fix grounded in a grep; the three
apparently-missing paths are external projects' docs (pi, omp) or files
the doc itself says were removed. Method blind spot: cli-settings and
packages were swept, not read line by line.
visual-review: n/a
Not done / debts: wave 3 (native surfaces/apps) and wave 4 (stable)
remain in docs/plans/docs-prose-audit.md.
Merge: fast-forward ready.
