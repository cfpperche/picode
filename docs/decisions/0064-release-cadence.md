# ADR-0064: Official release cadence and release lanes

- **Status**: proposed
- **Date**: 2026-09-04

## Context

PiCode already has the mechanical boundary for an official release: a
maintainer creates a SemVer tag, the release workflow builds the supported
artifacts, and the workflow refuses to publish without a matching changelog
section and What’s New catalog entry. Source builds and `make deploy` remain
useful for local dogfood, but they are not public releases.

The repository does not yet commit to a release frequency, a freeze window, a
pre-release channel, or a post-release observation period. Choosing a cadence
without documenting those boundaries would make the tag workflow the de facto
policy and would make the What’s New surface noisy if releases were cut only
because a calendar said so.

The benchmark study in
[`docs/benchmarks/2026-09-04-release-cadence.md`](../benchmarks/2026-09-04-release-cadence.md)
compares the documented processes of VS Code, Linear, Zed, Cursor and Go.

## Decision (proposed)

PiCode will use two release lanes:

1. **Source / dogfood is continuous.** Changes on `main` may be deployed after
   the normal quality gates. These builds are not official SemVer releases,
   expose `release: false`, and never auto-open the stamped-build What’s New
   surface.
2. **Stable is intentional and scheduled by a maintainer.** A Stable release
   is an accepted commit on `main` identified by an intentional `vX.Y.Z` tag.
   The initial experiment is a three-release, two-week train. This ADR does not
   choose the first calendar date and does not add a timer to CI.
3. **A patch can escape the train.** A regression or security fix may ship as a
   patch release when ready. It still requires a changelog section, a matching
   What’s New entry, green CI and the normal artifact checks.
4. **A release needs a reason.** A Stable tag is cut only when it contains a
   meaningful user-visible highlight or an urgent fix. Otherwise work remains
   under `[Unreleased]`.
5. **The train has explicit stages.** The maintainer records scope, freezes
   feature work, runs the quality gates, publishes the tag, verifies the
   artifacts and observes the installed release. The exact commands live in
   [`docs/release-process.md`](../release-process.md).

This remains a proposal until the owner accepts it. Until then, the existing
tag-triggered workflow is the only release mechanism and no cadence is
official.

## Decision table

| Condition | Lane / version | Action |
|---|---|---|
| `main` changed, no release tag | Source / development build | Dogfood only; no official release or automatic What’s New open |
| Candidate has a meaningful user-facing change and all gates pass | Stable / `vX.Y.Z` | Freeze scope, tag the accepted commit and let CI build/publish |
| Candidate has only a regression or security fix | Stable patch / `vX.Y.(Z+1)` | Ship outside the train with the same release validation |
| No meaningful user-facing change and no urgent fix | No new version | Keep entries under `[Unreleased]` |
| Any required gate fails | Candidate remains unpublished | Fix or defer; do not publish a partial release |

## Consequences

- Maintainers and users can distinguish a local source build from an official
  binary, which keeps What’s New and update behavior truthful.
- A two-week pilot provides a measurable starting point without pretending
  that official release dates have already been selected.
- The release process gains a freeze and observation step, so the cost of a
  release is more than creating a tag.
- Maintainers must curate both the full changelog section and the concise
  What’s New entry; this is deliberate and already enforced by ADR-0063.
- A very small change may wait for the next train. Critical fixes are the
  explicit exception.
- A future Preview or weekly lane would need a new decision (or an amendment)
  covering support, versioning, artifacts and rollback.

## Alternatives considered

- **Release on every merge:** rejected for Stable. It would turn source
  delivery into a public contract and make the bounded What’s New catalog
  repetitive.
- **Weekly Stable immediately:** rejected as the starting point. It matches
  Zed/Cursor-like output but assumes release automation and verification that
  PiCode has not measured yet.
- **Monthly Stable immediately:** deferred. It is a safe, curated baseline,
  but the two-week pilot gives faster feedback while retaining a release train.
- **Calendar-triggered tags:** rejected for now. An empty or failing release
  must never be created merely because a scheduled job fired.
- **Public Preview channel now:** deferred until a second supported build
  stream and an owner for its support burden exist.
