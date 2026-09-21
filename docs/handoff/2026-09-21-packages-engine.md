# 2026-09-21 — feat/packages-engine: the legacy Pi reads answer from the driver (ADR-0176 slice 1c)
Shipped: `GET /api/packages` and `GET /api/packages/updates` now answer from the Pi driver instead of calling `pipkg` themselves. `internal/pkgs` gained the badge read (`Driver.CheckUpdates`, gated by `Caps.Update`), `Query.AgentIsolated` (the agent row's "only this agent's packages" switch, which the model could not otherwise express) and the legacy mappers `Report.Legacy` / `LegacyUpdates` (`internal/pkgs/legacy.go`). `internal/server/packages.go`: reads resolve their driver from `?cli=` through one `packageReadDriver` (shared with `/api/packages/report`), and `loadPackageReport`, `mutateAgentPackage` and `agentHasRolesPackage` thread `r.Context()` so the driver call is never given a fake context. Mutations still call `pipkg`; `/api/cli-packages*`, `internal/pipkg` and every `web/` file are untouched.
Verified: `make ci-scoped` PASS (fmt,vet,hooks,go[5]; 8 path(s) vs main), then `make close` green on this tree after merging main; `internal/pipkg`, `internal/pkgs`, `packages_test.go`, `packages_config_test.go`, `packages_describe_test.go`, `update_test.go` green unchanged. The mapping is byte-identical to `pipkg.List` + `WithAgent` + `Isolated`: a unit test compares the marshalled payloads, and a scratch HTTP run (deleted) compared the served bytes over machine/workspace/agent/isolated, the unknown-agent and empty-report cases, both empty updates payloads, the error bodies and the parse-failure message. `git diff --stat main...HEAD -- web/` is empty.
visual-review: n/a
Not done / debts: no changelog fragment — nothing user-visible changed. Two deliberate deltas are recorded in `docs/handoff/open/packages.md` (guest `?cli=` on the badge route now refuses; `/api/packages/report` still does not carry `Isolated`).
Merge: fast-forward ready (2 ahead, 0 behind `main`, clean).

## Next up

- Slice 2 of `docs/plans/packages-unification.md`: the eight guests behind `pkgs.Driver` — parsers, argv builders and fixtures untouched, `Caps.Update` declared only where the CLI really has a catalog check.

## Debts

- `docs/handoff/open/packages.md`: a guest asked for the badge read at `/api/packages/updates?cli=<guest>` is refused with 400 (its own route is `/api/cli-packages/updates`), and the unified `/api/packages/report` does not pass `Query.AgentIsolated` yet, so the isolation switch is visible only through `/api/packages`.
