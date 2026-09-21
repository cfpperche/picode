# 2026-09-21 — muse-plugin-gate: Muse's plugin roster is read from its record (ADR-0167)
Shipped: `internal/clipkgs/roster.go` gains `parseMuse`/`museRecordRow`, called by
`museRoster`. Muse nests a plugin's id/version/enabled/source under each row's
`record`, not on the row; the tolerant reader skipped every row, so a machine with
plugins saw a refusal where the list belongs. `testdata/muse.list.json` pins the
measured shape (Muse Code 1.3.0, 2026-09-21) with four new `roster_test.go` cases.
The live harness seeds Muse's cached per-machine feature config (from the real home
via `os/user`, since the recipe swaps HOME) into the sandbox and installs a real
bundle, reading the row back. Docs: Muse's row in `docs/architecture/cli-packages.md`,
ADR-0167's depth table and `docs/handoff/open/packages.md` corrected;
`docs/changelog.d/muse-plugin-gate.md` travelled with the code.
Verified: `go test ./internal/clipkgs/...`, `make ci-scoped` and `make close` PASS
(ff-ready); live suite clean end to end (14.9s); `/api/cli-packages?cli=muse` returns
the row (id, version, enabled, sourceKind `native-local`, installPath). The harness's
earlier "build refuses" reading was the vendor's per-machine gate, not the build;
blind spot is the catalog (`--available`) row shape, still unmeasured — Muse's
marketplace spec wants an `install.transport` the harness cannot satisfy.
visual-review: PASS on a scratch instance — Disable/Update/Inspect/Remove,
`__picodeOverlayAudit()` ok.
Not done / debts: Muse's catalog row shape is still read tolerantly.
Merge: fast-forward ready; at close 12 files, +386/-24, scope FULL (GO=1 DOCS=1
METADATA=1); branch 1 ahead, 0 behind, clean.
## Debts
- Muse's catalog row shape stays tolerant until measured: `docs/handoff/open/packages.md`.
