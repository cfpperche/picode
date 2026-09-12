# 2026-09-11 — feat/clis-review-fixes

Adversarial review of feat/clis-terminals-section (merged as 0f83ff5b).
Four follow-ups:

- `docs-site/guide/agent-clis.md` still described the removed general
  **Terminals** tab — and claimed it listed ordinary terminals, which it
  never did (the list was CLI-only). Rewritten for the per-CLI section.
- `docs/benchmarks.md` width table still enumerated "Terminals" among the
  `#/clis/*` routes. Dropped.
- `#/clis/terminals?…` rendered the catalog but kept the stale address:
  the rewrite compared the hash exactly. Both shells now compare the
  query-less hash (`#/clis/terminal/<id>` stays untouched). One Fixed
  fragment.
- `docs/handoff.md` still listed `feat/clis-terminals-section` as in
  flight; the branch is merged (0f83ff5b). Row removed.

**Gates** — `make ci-scoped` PASS; `make close` clean; main
fast-forwarded. Verified on the scratch build of the final main: six
tabs, per-CLI section, old address (plain and with query) lands on the
catalog with the hash rewritten.
