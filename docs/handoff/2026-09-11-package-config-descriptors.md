# 2026-09-11 — feat/package-config-descriptors: Configure for every declared package

Shipped (C0–C2 of `docs/plans/package-config-manifest.md`): ADR-0119 seeds
the descriptor contract; `internal/pipkg/configdescriptor.go` resolves
descriptors (catalog with `pi-web-search` first, then `picode.config`
manifests beside installed packages); `packages_config.go` generalizes
GET/PUT/DELETE to descriptor-driven configs inheriting every ADR-0099
guarantee (unknown keys preserved, atomic writes, 409 + explicit replace,
`packages.config` feed event); `PackageConfigGeneric.jsx` renders the form,
with `descriptorValuesSchema` in the shared contracts. pi-roles keeps its
bespoke editor; decision-table tests cover both layers.

Verified: `make ci-scoped` (fmt, vet, hooks, go[10], test-js, build),
`make close` green after merging main (providers-density landed mid-session).
visual-review PASS on a scratch instance (`qa-scratch.sh pkgcfg`, screenshots
read from `var/screenshots/pkgcfg-*.png`): list card shows Configure;
empty ("Not configured yet — saving creates …"), inline Zod error ("Model is
required."), saved ("Saved." + file on disk verified), invalid-file conflict
banner → Replace file… → resolved. `__picodeOverlayAudit()` ok, footer row
36px uniform.

visual-review: PASS

Not done / debts: descriptor v1 is agent-scope, single-file (workspace scope
and per-agent overlays stay roles-bespoke); no cross-field validation (model
belongs to provider is the tool's own check); C3 catalog sweep + upstream
manifest proposal remain.

Merge: fast-forward ready.
