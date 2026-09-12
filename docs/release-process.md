# PiCode release process

> **Status:** operational draft. The cadence proposal in
> [ADR-0064](decisions/0064-release-cadence.md) is not accepted yet, so this
> document does not set official dates or add a scheduled CI trigger.

This is the maintainer runbook for turning an accepted commit into a public
PiCode release. It is intentionally separate from the user-facing
[changelog](../CHANGELOG.md) and from the in-product What's New copy.

## Release boundaries

| Boundary | Meaning | Release identity |
|---|---|---|
| `main` / `make deploy` | Source build for local development and dogfood | Build identity; `release: false` |
| `vX.Y.Z` tag | Public Stable release boundary | SemVer-stamped binary and GitHub Release |
| `vX.Y.(Z+1)` patch tag | Regression or security fix outside the train | Patch release with the same validation |

`make deploy` rebuilds the current checkout and restarts the installed service;
it does not publish a GitHub release. `picode update` consumes the published
GitHub artifacts, not a source deployment.

## Before the train

1. The owner selects the candidate version and release window. Until
   ADR-0064 is accepted, the window is intentionally unset.
2. Review the changes since the previous release. A Stable release needs at
   least one meaningful user-facing highlight or an urgent fix. Do not create
   a release just to satisfy a calendar.
3. Run `make changelog` on `main` to fold the `docs/changelog.d/` fragments
   into `[Unreleased]` (ADR-0105), commit, then confirm every user-visible
   change has an honest entry in [`CHANGELOG.md`](../CHANGELOG.md). The
   version cut itself — renaming `[Unreleased]` to `[x.y.z] - date` — is the
   one `CHANGELOG.md` edit the pre-commit hook accepts without fragments.
   The fold also **normalises the block**: one `### Section` heading per type,
   in Keep a Changelog order. It has to, because the cut publishes that block
   verbatim as the release body — 0.2.0 was one fold away from announcing six
   separate "Fixed" headings inherited from the direct edits that predate
   fragments. If you ever see a repeated heading in a release section, a fold
   heals it; do not hand-sort it.
4. Add or revise the matching entry in
   [`web/shared/data/whats-new.json`](../web/shared/data/whats-new.json). Keep the
   catalog concise: benefit-led titles and summaries, with no more than nine
   highlights for the surface.
5. Bump `Version` in [`internal/version/version.go`](../internal/version/version.go)
   to the candidate. The release workflow stamps the tag into the binary it
   builds, but **a source build keeps whatever the constant says** — so a stale
   constant makes `make deploy` report the previous SemVer on `/api/version`
   and makes `picode update` offer the release to a checkout that already
   contains it. 0.2.0 shipped with the constant still at `0.1.0`; the deploy
   that followed is what caught it.

## Release train

### 1. Freeze scope

Record the candidate commit and included changes. Stop feature work for the
candidate; only release-blocking fixes may enter after the freeze. If a fix
changes the user-visible story, update both the full changelog and the concise
What's New entry before continuing.

### 2. Verify locally

Run the complete quality gate from a clean, up-to-date checkout:

```bash
make ci
node scripts/release-notes.mjs "$VERSION" --check
git diff --check
```

The release-note check must find both a non-empty `CHANGELOG.md` section and a
matching What's New catalog entry. For UI changes, read the required visual
evidence before the release is considered ready.

### 3. Publish the chosen commit

After review, commit the release-note changes on the accepted `main` history
and create an annotated tag. Replace `0.2.0` with the selected version:

```bash
VERSION=0.2.0
git tag -a "v$VERSION" -m "picode v$VERSION"
git push origin main "v$VERSION"
```

The tag is the approval boundary. Do not create a second, unreviewed commit
after tagging. If a manual retry is needed, the existing
`workflow_dispatch` input accepts a tag that already exists.

### 4. Let CI build and publish

`.github/workflows/release.yml` will:

1. resolve the version from the tag;
2. validate the matching changelog and What's New entries;
3. build the Vite UI and all supported binaries;
4. generate `SHA256SUMS`; and
5. create or update the GitHub Release with the changelog section as its body.

Do not hand-edit a release body to bypass a failed validation. Fix the source
documentation or the build, then rerun the workflow for the same tag only
when the published artifacts have not been replaced.

### 5. Verify delivery

Check the GitHub Release page, every expected asset and its checksum. Install
one representative artifact and verify:

- `GET /api/health` reports `status: ok`;
- `GET /api/version` reports the selected SemVer and `release: true`; and
- the What's New surface opens once for the stamped build, remains manually
  reachable, and does not interrupt an active Inbox or creation flow.

Run the artifact against an **isolated** `HOME` and `PICODE_DATA` on a free
port — the binary's default is the server, on the production data directory.

**What's New opens on the empty instance you just installed.** That is the
check: a fresh data directory, no workspace, no agent, no terminal, first
load — the surface opens once, and a reload after closing it stays closed
(the browser wrote `picode-whats-new-seen`). Seeding a terminal first was the
old recipe, for the product-state gate ADR-0063 shipped with; its 2026-09-11
amendment removed that gate, so seed nothing. An empty instance showing **no**
What's New is now a regression to chase, not the decision working.

Deferral is still real and still worth one look: with a dialog open, a
recovering connection, a waiting agent, an Inbox badge or an active
create/share flow, the surface waits rather than interrupting.

Record the tag, commit, CI run, artifact verification and observation owner in
the release issue or handoff. The installed source service can be dogfooded
with `make deploy`, but that is not evidence that the public release shipped.

### 6. Observe and close

Watch the release for the owner-selected observation window. Record incidents,
update failures, support questions and What's New feedback. A release that
needs a corrective version becomes a patch release; do not silently rewrite a
published tag. Close the release only after the owner has reviewed the result.

## No-release and hotfix rules

| Situation | Action |
|---|---|
| No meaningful user-facing change and no urgent fix | Keep work under `[Unreleased]`; do not tag |
| Quality gate or artifact check fails | Hold the release and fix or defer |
| Regression or security issue after Stable | Cut a patch tag with changelog and What's New validation |
| A feature misses the freeze | Move it to the next train; do not add it informally |

## Release-note shape

The full release body stays in Keep a Changelog form:

```markdown
## [0.2.0] - YYYY-MM-DD

### Added

- Benefit-led description of the shipped user-visible change.

### Fixed

- Honest description of the user impact and correction.
```

The matching What's New entry is shorter and written for a person scanning the
product. It links to the complete GitHub release for technical detail; it does
not replace the changelog.

## Ownership and changes to this process

The maintainer cutting the tag owns the checklist and post-release observation.
The owner accepts or changes the cadence in ADR-0064. A new channel, automated
calendar, release branch, or different versioning contract requires an ADR
amendment or successor before the workflow is changed.
