# 2026-09-22 — feat/packages-alias: the `/api/cli-packages*` alias is gone

Shipped: the one-release alias ADR-0176 promised is removed (the owner closed the window; not a calendar check).
`internal/server/cli_packages.go` -> `packages_vendor.go` keeps the engine and answers the paths Pi's own calls use
(`GET /api/packages/{available,marketplaces}`, `POST /api/packages/{toggle,marketplace,inspect}`, and the `cli`-named
branch of `POST /api/packages`, `/update`, `DELETE /api/packages`) — `ownCLI` tells a CLI's own verb from PiCode's own
call, which never names a CLI. Every answer is `pkgs.Report`/`pkgs.Row`; `internal/pkgs/guest_view.go` and its `Guest*`
mappers are deleted with the tests that pinned their bytes. `web/shared/domain/cliPackages.js` reads the unified paths
for every verb with its transport rules unchanged, and `sourceGroupKey` reads the unified row's `kind`/`installedPath`
where the alias read `sourceKind`/`installPath` — which also restores the pane's plugin grouping. Docs:
`docs/architecture/packages.md`, `routes.md`, the regenerated OpenAPI spec (no `cli-packages` tag), the paid debt
bullet with its caller inventory, and `docs/changelog.d/packages-alias.md`.
Verified: `make ci-scoped` PASS (fmt,vet,hooks,go[5],test-js,build,docs; 15 paths vs main) and `make test-js`; the pane
driven on a scratch instance (stub `omp` on PATH) — install, update, toggle and remove through the UI, each read back
on disk, the Omp workspace extension's own write (`disabledExtensions: ["extension-module:tool"]`, every other key
intact), and the unnamed-module refusal on the OpenCode pane.
Blind spot: `PICODE_PKGS_LIVE=1` was not run (forbidden for this branch), so the vendor seam was a stub, not the real
binary; the mobile pane was not walked on a device. visual-review: PASS (desktop, `__picodeOverlayAudit()` `ok:true`).
Not done / debts: none. `internal/clipkgs`, the vendor facts table and Pi's own `pipkg` paths are untouched; the
surviving `cli-packages` strings name the pane's DOM id (`#cli-packages-view`, `.cli-packages-fields`) and its QA
script, never the removed route.
Shipped in **0.5.0**: `eaf923dd release: cut 0.5.0` folded this fragment with the other 39 and moved
`internal/version.Version`; the owner's `--force` deploy followed (no other mutation in flight) — health `ok`,
`/api/version` `0.5.0+abd9462`, and in production the family answers **404** while `/api/packages/report?cli=omp`
answers **200**.
