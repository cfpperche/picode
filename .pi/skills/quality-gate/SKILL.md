---
name: quality-gate
description: Run PiCode's review checklist before declaring work done — `make close` for the mechanical gates, this file for the judgement (tests, decision tables, changelog, visual). UI work without visual-review PASS (screenshot read) is this gate FAIL.
---

# Quality gate

Run this **before declaring any code work done** in PiCode. The rules come
from `docs/benchmarks.md` (engineering section) and `AGENTS.md`. Be honest —
a failed gate reported honestly beats a passed gate reported falsely.

## The mechanical part is one command (ADR-0086)

```bash
make ci-scoped   # while iterating: only the gates this diff can break
make close       # at the end: gates + regenerated artifacts + ff check + summary
```

`make close` refuses a dirty tree, regenerates OpenAPI/llms.txt/captures
when the diff invalidated them (and commits them), and tells you whether
`main` can fast-forward. Do not hand-run gofmt/vet/tests/build one by one
in twenty turns; if `close` fails, fix the cause and rerun it.

## The judgement part (read the diff: `git diff --stat main...HEAD`)

1. **One logical change?** If not, split the commit.
2. **New code without tests?** Table-driven tests before finishing.
3. **Interacting conditions** (delete/restore/auth/cascade/mode)? A decision
   table exists and every row is tested or named as debt. Two clicks ≠ matrix.
4. **Timed UI** (jobs, overlays, lists)? Motion on enter/step/exit; the
   optimistic next state, not a static wait then a jump.
5. **New non-stdlib dependency?** Justified in the commit/PR description
   (AGENTS.md rule #3). No justification = remove it.
6. **Store mutation without an event**, or a new poll against `/api/*`
   without a stated reason the feed cannot cover (ADR-0048) → FAIL.
7. **Changelog**: user-visible → an entry under `[Unreleased]`.
8. **Docs**: behavior/architecture changed → `docs/architecture.md`; a
   boundary (protocol, persistence, security, process) → ADR.
9. **Visual gate** (any user-facing surface changed): `/skill:visual-review`
   and `/skill:uiux-review` on a **scratch instance**
   (`scripts/qa-scratch.sh`), never on production. Screenshots read,
   overlayAudit ok, visual-card in the reply. Skipped or FAIL → this gate
   is FAIL. `eval` JSON is not a visual pass.
10. **Handoff**: `/skill:handoff-update` (one note in `docs/handoff/`).

Deploy is **not** a step: `main` ships in batches (`make deploy-batch`).

## Report format

```
quality-gate: PASS (close ✓ tests +4 decision-table 6/6 changelog +1 visual ✓)
quality-gate: FAIL (visual: overlay clipped — see visual-review)
quality-gate: FAIL (close: go test internal/server — 1 failure)
```
