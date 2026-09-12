# 2026-09-12 — feat/board-doc-refs: the instructions stop pointing at the board

Leftover from ADR-0123: the board stopped being editable, but nine documents
still told the reader to write in it — `CONTRIBUTING.md` (the human's contract:
the reading step and the "project state at all" row), the `uiux-review` and
`visual-review` skills, `docs/philosophy.md`, `docs/design/file-preview-
roadmap.md` and five plans. Each now points at `docs/handoff/open/<topic>.md`
or the session note, and says the board is generated.

Deliberately not touched: `docs/handoff-archive.md`, `docs/benchmarks/*.md` and
`CHANGELOG.md` (frozen history — they describe the file as it was), and
`cmd/picode-docs-fixture/main.go`, which seeds a `docs/handoff.md` inside its
synthetic workspace as UI data, not as an instruction.

Verified: `make close` (docs scope: docs-check, the site build, Vale).

visual-review: n/a (no UI change).

Merge: fast-forward ready.
