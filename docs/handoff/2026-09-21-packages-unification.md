# 2026-09-21 — feat/packages-unification: one package engine behind a driver interface (slice 1)
Shipped: ADR-0176 (proposed; supersedes ADR-0167's engine clause, amends ADR-0010) and `docs/plans/packages-unification.md`, plus slice 1 of that plan. `internal/pkgs` holds the model (`Row`, `Report`, `Caps`, `Scope`), the registry (`DriverFor`, `Known`, `CLIs`) and `SourceName`; Pi is a driver over `internal/pipkg` and one guest driver sits over `internal/clipkgs`, each engine untouched. New read: `GET /api/packages/report?cli=&scope=&workspace=&agent=&refresh=` answers the unified report for any CLI. `/api/packages*` and `/api/cli-packages*` keep their routes and their JSON exactly — no file under `web/` changed.
Verified: `go test ./internal/pkgs/` and `go test ./internal/server/ -run TestPackageReport` green here; `git diff --stat main...HEAD -- web/` empty, which is the no-regression proof; gates: `make ci-scoped` PASS (fmt,vet,hooks,go[5],docs; 11 paths vs main); landed on main with `make ci` green. Method blind spot: no live renderer or browser pass, because no pane changed.
visual-review: n/a
Not done / debts: one deviation from the plan's letter — see `## Debts`.
Merge: fast-forward ready (3 ahead, 0 behind `main`, clean).

## Next up

- (1c) route the legacy `/api/packages*` handlers through the driver, so the pane reads one source of truth; then slice 2 (the guests' mutation verbs) and slice 3 (one pane, its controls gated by `Caps`).
- Then slice 4 (agent scope as a declared capability, Omp first) and slice 5 (Omp's own `extensions` reader, closing the debt in `docs/handoff/open/packages.md`); plan: `docs/plans/packages-unification.md`.

## Debts

- `/api/cli-packages?cli=pi` does not start answering in slice 1, though the plan's slice-1 row says it does: answering it would need a second renderer (the guest shape) for one CLI, which the plan itself refuses. That request is still refused with 400 (`internal/server/cli_packages_test.go`); the `cli=pi` answer lands with slice 3's single shape at the unified route. Owner: `docs/plans/packages-unification.md`.
