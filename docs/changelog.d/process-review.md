### Changed
- **Development process (ADR-0105).** Deploy happens only when the owner runs
  `make deploy`; the deploy timer and `make deploy-batch` are gone. The
  changelog is assembled from `docs/changelog.d/` fragments, the handoff
  keeps no shipped-work prose, `docs/architecture.md` is an index over
  `docs/architecture/`, and `make adr` seeds a decision record with a
  boundary line. Gates: `internal/server` tests run sharded across four
  processes, `make ci` runs its gates in parallel, `make web` is a no-op when
  nothing under `web/` changed, and `make close` reuses a green
  `ci-scoped` for an unchanged tree.
