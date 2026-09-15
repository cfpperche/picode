# 2026-09-15 — board-stale-line

The board still said ADR-0143 had two steps open; it is finished. A silent
`str.replace` in the term-ui slice had not matched the file's wording, so the
correction never landed — the lesson is that a docs edit needs its result
printed, not assumed.

Docs-only: `ci-scoped` PASS, no deploy.
