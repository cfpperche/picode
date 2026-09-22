# 2026-09-22 — feat/packages-docs: the docs catch up with the unified packages engine (ADR-0176, slice 6)
Shipped: one `docs/architecture/packages.md` replaces `docs/architecture/cli-packages.md` — the engine
(`internal/pkgs`: `Scope`/`ScopeRow`/`Catalog`/`Caps`/`Row`/`Report`/`Query`/`Target`/`MarketRequest`/`Command`,
the `Driver` interface, `DriverFor`/`Known`/`CLIs`, the `legacy.go`/`guest_view.go` mappers), the routes (the
unified `GET /api/packages/report` plus the `/api/cli-packages*` one-release alias with its `cli=pi` and
`scope=agent` refusals), the job lane against the direct calls (`Caps.Async`; OpenCode's splice and Omp's
workspace `extensions` write carry no argv), the agent layer (`agents.packages`/`packagesIsolated`, launch
injected: Pi's four isolation flags, Omp's `-e` plus the two that CLI has — `pi` and `omp` are the two CLIs
that declare it), and the measured vendor facts table with its dates. Also: the index
(`docs/architecture.md`, whose "Pi packages stay Pi-only" clause was false), `routes.md` (the packages hash
row and a new HTTP-surface paragraph), the public guides (packages, picode-mcp's "Omp `-e` path is not
built" claim, browser-tool, agent-clis, guide/index), fragment `docs/changelog.d/packages-docs.md`, and
`docs/handoff/open/packages.md` with the alias-removal debt this slice opens.
Verified: `make docs-check` ok (public captures stale, advisory, pre-existing) and vale 0 errors on 48
files; `make close` = ci-scoped PASS (fmt,vet,hooks,docs; 11 paths vs main) after merging main, then
re-run. Every claim was read against the code (`internal/pkgs`, `internal/server/{packages,cli_packages,
package_report}.go`, `internal/clipkgs`, `internal/store/agents.go`, `internal/clijob`, both apps' Packages
panes and `web/shared/domain/cliPackages.js`).
visual-review: n/a (documentation only; no `web/` file changed)
Not done / debts: ADR-0176 keeps `Status: proposed` — acceptance is the owner's to record, and the index
matches its file (docs-check only rejects "accepted in the file, proposed in the index"). Historical plans
and handoffs still name the removed `cli-packages.md` as what a past slice touched; they are records, not
live links (docs-check validates links, not inline code).
Merge: fast-forward ready.
