# 2026-09-21 — feat/audit-fixes: an adversarial pass turns two bugs and four stated rules into gates

Shipped, five commits (12c38326..fa57f4d8). Two real bugs. The published OpenAPI document was missing every `/api/auth` route and `/pair`, because `registerAuthRoutes` returned early on a nil gate while `Routes()` records registration against a zero `Deps` — and docs-check byte-compared the committed JSON against that same generator, so CI stayed green on 331 paths instead of 338; registration is now unconditional (a nil gate answers 404 at request time) and `TestRoutesCoverEveryRegisteredPattern` requires the zero-`Deps` and populated-`Deps` route sets to match. `backup.Restore` deleted the live pins directory and pi's session tree before copying the snapshot over them, with no rollback, after the database had already been swapped in; `swapTree`/`swapRegular` now build beside the target and rename into place, and a replaced credential vault is kept as `credentials.json.replaced` (decision table, live present × copy succeeds, in `restore_test.go`).

Enforced now, not before: `TestEveryExportedMutationAnnouncesOrIsListed` parses the store and requires every exported mutator that writes to a table to append an event or be named in `silentMutators` with a reason (13 reasoned exceptions), replacing a 118-case hand-written checklist; a `noValidate` check over every `<form>` (three did not opt out); and three living-docs checks in `scripts/docs-check.mjs` — ADR index/file status parity, architecture-index completeness, relative-link resolution in `docs/`. Eight ADR index rows saying proposed while their own file said accepted were synced.

Docs: user guides for Pins, Backup and Dev servers — three shipped surfaces that had none — and `docs/architecture/climetrics.md` for a ~3,900-line subsystem with no architecture file. `CHANGELOG.md`'s header contradicted ADR-0105; the pre-commit hook gained a third legitimate direct edit (the preamble above the first version heading), covered both ways by hooks-selftest.

Verified: `make close` green, full scope. visual-review: n/a — the only UI diff is a `noValidate` attribute on three forms, nothing rendered changes. Nothing was deployed.
Merge: fast-forward ready (base 434f2951, head a6d7db4e); the merge and `make ci` on main have not run.

## Next up

- Two questions this branch deliberately left to the owner are recorded in `docs/handoff/open/process.md`: whether `/pair` should be guarded-and-exempt, and whether ADR-0154 and ADR-0156 are accepted.
