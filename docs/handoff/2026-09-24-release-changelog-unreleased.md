# 2026-09-24 — release-changelog-unreleased: restore changelog preview after a release

Shipped: the changelog assembler recreates `[Unreleased]` before the first release when a release cut consumed the previous section. Regression coverage and a changelog fragment accompany the fix.
Verified: 11 Node tests, `node scripts/docs-changelog.mjs`, and `make ci-scoped` passed. The failure was found by full CI on `main` after Settings pilot commit `f94e54f98`; full CI has not been rerun with this branch.
visual-review: n/a (generated documentation).
Deploy: none.
Merge: fast-forward ready at `b84c44f0c`.
