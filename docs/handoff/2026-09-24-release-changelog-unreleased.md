# 2026-09-24 — release-changelog-unreleased: restore changelog preview after a release

Shipped: the changelog assembler recreates `[Unreleased]` before the first release when a release cut consumed the previous section. Regression coverage and a changelog fragment accompany the fix.
Verified: 11 Node tests, `node scripts/docs-changelog.mjs`, `make ci-scoped`, and full `make ci` on `main` passed. The failure was found by full CI after Settings pilot commit `f94e54f98`.
visual-review: n/a (generated documentation).
Deploy: none.
Merge: landed by fast-forward at `9af0093c6`.
