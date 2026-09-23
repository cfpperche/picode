# 2026-09-23 — feat/ci-docs-links: the living-docs pass runs before main

Scripts only. `feat/agents-md-study` closed green, then failed `make ci` on `main` at 65ba8571 and blocked every landing after it until a follow-up branch. A change under `docs/` is scope `metadata`, whose gates were fmt, vet and hooks, so docs-check's link pass first ran on `main`, after the fast-forward.

**The pass.** The ADR-index, architecture-index and relative-link checks moved from `scripts/docs-check.mjs` into `scripts/docs-living.mjs`, which docs-check still runs. `ci-scoped` runs it alone for a metadata change, and `make close` runs it on every close, reused green run or not. Fenced blocks and inline code spans no longer count as links; that false positive is what started it. Tests: `scripts/docs-living.test.mjs`, 12 cases.

**Why not the reuse rule.** Counting `docs/` as covered (ADR-0124) would re-run every branch's tests whenever a note lands on main. Running the one-second pass at every close leaves ADR-0124 as it is.

Verified: `make ci-scoped` PASS (full, because the change shapes the gates) and `make close` exit 0. The debt in `docs/handoff/open/process.md` is marked paid. Nothing deployed.
