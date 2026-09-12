# 2026-09-11 — the source version constant lagged the release (`feat/version-bump`)

**Shipped.** `internal/version.Version` is `0.2.0`, and the runbook gains the
step that sets it: bump the constant with the changelog cut.

**Why.** The release workflow stamps the tag into the binary it builds, but a
source build keeps whatever the constant says. 0.2.0 shipped with it still at
`0.1.0`, so the `make deploy` right after the release served
`"semver":"0.1.0"` on `/api/version` — and `picode update` would have offered
v0.2.0 to a checkout that already contained it and more
(`cmd/picode/main.go` compares `install.Newer(version.Version, rel.Tag)`).
Nothing in the runbook named the constant; step 5 does now, with the miss as
its evidence. The file's own comment already said "kept in sync with
CHANGELOG.md releases" — the process just never carried it.

**Verified.** `internal/version` and `internal/install` tests pass (both take
literal versions as arguments, so the bump pins nothing), then `make close`.

**Debt.** Nothing fails when the constant and the newest `CHANGELOG.md`
release disagree; a gate could compare them, and did not exist to be added
here without widening the diff.

**Merge.** `git merge --ff-only feat/version-bump && make ci`.
