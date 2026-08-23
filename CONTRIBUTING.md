# Contributing to PiCode

Humans and Pi agents contribute under the same contract. The full operating
rules live in [AGENTS.md](AGENTS.md) — this file covers the mechanics.

## For humans

1. Fork / branch, do your work, keep changes small and reviewable
   (benchmark: one logical change per PR).
2. `make ci` must pass locally (gofmt, vet, tests, build).
3. PR description must include:
   - What changed and why (link an issue when one exists).
   - Justification for any new non-stdlib dependency.
   - Doc updates that traveled with the change (they must — see below).
4. UI changes: run the `/skill:uiux-review` checklist mentally or attach
   a screenshot; the benchmarks in [docs/benchmarks.md](docs/benchmarks.md)
   are enforced in review.

## For Pi agents

1. Read `AGENTS.md` (auto-loaded) and `docs/handoff.md` **before** starting.
2. Use the skills: `/skill:quality-gate` before declaring done,
   `/skill:handoff-update` before ending.
3. Never break the build. Architectural change → ADR
   (`docs/decisions/template.md`).

## The documentation contract (applies to everyone)

Code and docs change together, in the same commit:

| You changed... | Then also update... |
|---|---|
| Behavior or architecture | `docs/architecture.md` (+ ADR if architectural) |
| Anything user-visible | `CHANGELOG.md` → `[Unreleased]` |
| Project state at all | `docs/handoff.md` (session end) |
| A benchmark we hold | `docs/benchmarks.md` with rationale |

## Language

English is the repository's official language — code, docs, commits,
changelog, issues and PRs (policy in [AGENTS.md](AGENTS.md)). Reviewers
should request translation rather than merge non-English content.

## Releases

Maintainers cut releases: version tags follow SemVer; every release
compiles its section from the changelog. No release without a green CI.
