# Study: release cadence and release-process documentation

- **Date:** 2026-09-04
- **Scope:** How comparable products separate internal delivery from official
  releases, document the release train, and communicate what shipped.
- **Method:** Primary public sources only. A public changelog is treated as a
  communications receipt, not as evidence of an undisclosed internal process.

## Sources

- [VS Code Development Process](https://github.com/microsoft/vscode/wiki/Development-Process)
  and [March 2026 iteration plan](https://github.com/microsoft/vscode/issues/300108)
- [Linear Releases](https://linear.app/docs/releases)
  and the open-source [Linear Release `RELEASING.md`](https://github.com/linear/linear-release/blob/main/RELEASING.md)
- [Zed stable releases](https://zed.dev/releases/stable)
- [Cursor changelog](https://cursor.com/changelog)
- [Go release cycle](https://go.dev/wiki/Go-Release-Cycle)

## Findings

| Benchmark | What is documented publicly | Cadence or channel signal | PiCode adaptation |
|---|---|---|---|
| VS Code | A development-process page describes roadmap, milestones, a Definition of Done, endgame testing, documentation updates, and release notes. The endgame publishes an Insiders build before Stable; a later iteration plan records the move from monthly Stable releases to weekly ones. | A planned iteration can end in a usable release; pre-release validation is an explicit phase. | Keep a written scope, freeze, verification and observation phase. Add a preview lane only when there is a real audience and support capacity. |
| Linear | Release pipelines are modeled as **continuous** or **scheduled**. Scheduled releases have dates and stages; every release records a name, commit SHA, issues, notes and a chronological changelog. Its release utility keeps a repository runbook with preflight, branching, tagging and CI steps. | Internal/nightly and production pipelines may use different policies in one product. | Separate source/dogfood delivery from the stamped Stable release. Keep the release commit and notes auditable in Git. |
| Zed | The public release index exposes Stable and Preview channels and gives each release a short summary followed by organized changes. | The page explicitly describes weekly releases. | Borrow the channel distinction and scannable note shape, not a weekly promise. |
| Cursor | The public changelog records dated, benefit-led feature announcements. | Recent entries make a weekly or fortnightly public cadence visible, but the internal train is not claimed. | Use benefit-first highlights in What’s New; do not infer governance from the changelog alone. |
| Go | The release-cycle page describes a six-month cycle with development followed by a long testing and polishing freeze. | Slow, predictable milestones with a quality-focused freeze. | Use as a reference for larger milestones, not as PiCode’s day-to-day cadence. |

## Convergent pattern

The benchmarks do not share one frequency. They do converge on a process:

1. A release is an explicit unit, not every merge.
2. The unit has a commit/version, an owner, a scope and release notes.
3. User-visible work is frozen before final verification.
4. Internal builds can move continuously while Stable follows a deliberate
   train.
5. The short user narrative is separate from the complete changelog and
   technical detail.
6. The process is written down where maintainers can execute it, while CI
   enforces the parts that are mechanically checkable.

## PiCode adaptation (proposed, not yet accepted)

- **Source / dogfood:** `main` may be deployed after the normal quality gates;
  it is a development build, has no official SemVer release, and must not
  auto-open the stamped-build What’s New experience.
- **Stable:** an intentional `vX.Y.Z` tag is the public release boundary. The
  candidate cadence is a two-week train for a three-release pilot; no first
  release date is set by this study.
- **Hotfix:** a regression or security fix may ship as a patch outside the
  train, with the same changelog and release-note validation.
- **No empty release:** if there is no meaningful user-facing change and no
  urgent fix, keep the work under `[Unreleased]` instead of publishing noise.
- **Review:** after three Stable releases, review lead time, hotfixes, failed
  gates, meaningful highlights, adoption lag and What’s New engagement before
  changing the frequency.

## Explicit non-adoptions

- No calendar-triggered GitHub release is added before the owner accepts the
  cadence ADR.
- No public Preview channel is promised before the team can support a second
  build stream.
- No release note is generated solely from commit subjects; a maintainer still
  curates the benefit-led What’s New entry.

The decision and the executable checklist are separate documents:
[ADR-0064](../decisions/0064-release-cadence.md) records the proposed policy;
the [release runbook](../release-process.md) records how an accepted release
is cut.
