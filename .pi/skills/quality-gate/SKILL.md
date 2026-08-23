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
4. **Build**: run `go build ./...`. Must compile.
5. **Diff review**: `git diff --stat` —
   - One logical change? If not, propose splitting the commit.
   - New code without tests? Add tests (table-driven) before finishing.
   - New non-stdlib dependency? It needs explicit justification in the
     commit/PR description (AGENTS.md rule #3). No justification = remove it.
6. **Changelog check**: is anything in this diff user-visible?
   - Yes → there must be a new entry under `[Não lançado]` in
     `CHANGELOG.md` (Keep a Changelog verbs: Adicionado/Alterado/Corrigido/Removido).
7. **Docs check**: did behavior or architecture change?
   - Yes → `docs/architecture.md` (and an ADR if architectural) must
     change in the same commit.
8. **Handoff**: run `/skill:handoff-update` to close the session state.

## Report format

End with a one-line verdict, e.g.:

```
quality-gate: PASS (fmt ✓ vet ✓ 12 tests ✓ build ✓ changelog +1 docs +1)
quality-gate: FAIL (vet: 1 finding in internal/server/server.go:42)
```
