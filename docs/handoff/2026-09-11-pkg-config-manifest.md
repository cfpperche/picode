# 2026-09-11 — feat/pkg-config-manifest: plan for package config descriptors

Shipped: `docs/plans/package-config-manifest.md` only — no code. The owner
asked for GUI configuration for every extension that has a config file
(today only pi-roles shows Configure; `pi-web-search`'s `web-search.json`
was never created because no surface said it existed). The plan pulls
forward the declarative-manifest end state ADR-0099 deferred: descriptors
resolve from the package manifest (`picode.config`) or a PiCode catalog,
drive a generic form and a generalized `packages_config.go`, and inherit
every ADR-0099 guarantee (files stay the truth, unknown keys preserved,
atomic writes, 409 + replace, scoped reset, feed event). pi-roles keeps its
bespoke editor. Phases C0–C4; ADR seeded at C1; 2–3 sessions estimated.
handoff.md Next up gains item 3 pointing at the plan.

Verified: plan-only change — `make ci-scoped` (metadata scope) and
`make close` green; no code paths touched.

visual-review: n/a — docs only.

Not done / debts: everything is the plan; open questions for the owner are
in the plan (default search model, catalog vs upstream-first, secrets).

Merge: fast-forward ready.
