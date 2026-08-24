---
name: quality-gate
description: Run PiCode's code quality gates before declaring work done — formatting, vet, tests, build, changelog check. Use when finishing any code change in this repo.
---

# Quality gate

Run this checklist **before declaring any code work done** in PiCode.
Rules come from `docs/benchmarks.md` (engineering section); the contract
is in `AGENTS.md`. Be honest — a failed gate reported honestly beats a
passed gate reported falsely.

## Steps

1. **Formatting**: run `gofmt -l .` from the repo root.
   - Any output = fix with `gofmt -w` before proceeding.
2. **Vet**: run `go vet ./...`. Must be clean.
3. **Tests**: run `go test ./...`. All pass, no skips without a linked
   issue/TODO in `docs/handoff.md`.
4. **UI build** (if `web/` changed): `cd web && npm run build`. Must succeed.
5. **Build**: run `make build` (UI + Go) or `go build ./cmd/picode` after `make web`. Must compile.
6. **Diff review**: `git diff --stat` —
   - One logical change? If not, propose splitting the commit.
   - New code without tests? Add tests (table-driven) before finishing.
   - New non-stdlib dependency? It needs explicit justification in the
     commit/PR description (AGENTS.md rule #3). No justification = remove it.
7. **Changelog check**: is anything in this diff user-visible?
   - Yes → there must be a new entry under `[Unreleased]` in
     `CHANGELOG.md` (Keep a Changelog verbs: Added/Changed/Fixed/Removed).
8. **Docs check**: did behavior or architecture change?
   - Yes → `docs/architecture.md` (and an ADR if architectural) must
     change in the same commit.
9. **Visual gate** (if `web/` or any user-facing surface changed):
   `/skill:visual-review` must be PASS (screenshot **read**, overlayAudit
   ok, visual-card answered). Skipped or FAIL → this quality-gate is FAIL.
   `eval` JSON is not a visual pass.
10. **Handoff**: run `/skill:handoff-update` to close the session state.

## Report format

End with a one-line verdict, e.g.:

```
quality-gate: PASS (fmt ✓ vet ✓ 12 tests ✓ build ✓ visual ✓ changelog +1)
quality-gate: FAIL (visual: overlay clipped — see visual-review)
quality-gate: FAIL (vet: 1 finding in internal/server/server.go:42)
```
