# 2026-09-11 — the two stumbles the 0.2.0 cut found (`feat/changelog-normalize`)

**Shipped.** `make changelog` now normalises the `[Unreleased]` block on every
fold: one `### Section` heading per type, canonical order, entries kept where
their heading put them. `normalizeBlock` runs before the fold (an inherited
duplicate would swallow the new entries into whichever copy came first) and
after it. Three tests: a drifted block, canonical order with its
unknown-section refusal, and a fold into a drifted changelog that heals it
without touching a released section.

**Why.** The cut publishes that block verbatim as the release body.
`[Unreleased]` held five `### Added`, six `### Changed` and six `### Fixed`
inherited from the direct edits that predate fragments (ADR-0105); `assemble`
inserts into the *first* match and walked past the rest. 0.2.0 was one fold
from announcing six "Fixed" headings; it was normalised by hand at the cut.

**Documented, not changed.** What's New does not auto-open on a fresh install
— ADR-0063's table waits for one workspace, agent or terminal. Step 5 of the
runbook now says to seed a terminal through the instance's own API, verify,
delete it by exact id, and run the artifact against an isolated `HOME` and
`PICODE_DATA` (the binary's default is the server on production's data dir).

**Verified.** `ci-scoped: PASS`; 6 assembler tests; a dry-run fold against the
real `CHANGELOG.md` left one heading and the 0.2.0 section byte-identical.

**Debt.** Nothing refuses a duplicate heading at commit time — the fold is the
only healer. **Merge**: `git merge --ff-only feat/changelog-normalize`.
