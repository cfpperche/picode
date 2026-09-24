# 2026-09-23 — feat/workspace-overview: routed workspace overview

Shipped: workspace menu opens a routed overview with attention, agents, project, missions, and scoped agent activity. ADR-0207 records the workspace metrics boundary; `/api/workspaces/{id}/stats` uses path ownership. The global machine dashboard remains available.
Verified: npm tests and build, `make ci-scoped`, and scratch UI QA on port 8893 with seeded data. Visual review read menu, populated, empty, and unavailable screenshots; menu overlay audit was `ok: true`. Scratch QA did not use authenticated vendor data and was not run inside the Windows shell.
visual-review: PASS (four screenshots; menu overlay audit `ok: true`)
Merge: main was integrated into the branch at db9598e38.
