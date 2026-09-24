# 2026-09-24 — feat/workspace-overview-v2: recent workspace activity

Implemented: the routed workspace overview highlights pending questions, agents, project state, missions, and recent changes. ADR-0210 records the activity scope; the server reads workspace events and the UI adds a recent-changes model and error state (f2b49480e).
Verified: `make ci-scoped` passed before the final narrow-layout and mission-event tweaks. Scratch QA on port 8894 used seeded workspace, question, and mission data and checked empty, missing, and error states. This did not exercise authenticated vendor data or the Windows shell.
visual-review: PASS (populated, empty, missing, error, menu, and 390 px attention/timeline captures; menu overlay audit `ok: true`).
