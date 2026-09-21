### Added

- **Pins, Backup and Dev servers have guides.** Three shipped surfaces had no
  user documentation at all: Pins is announced in Getting started as a sidebar
  tab and was never explained, Backup had seven API routes and not one
  sentence, and the Inspector's Servers tab existed only in the changelog.
  [Pins and reminders](/guide/pins), [Backup and restore](/guide/backup) and
  [Dev servers](/guide/dev-servers) now cover them, including the two things
  a backup user has to know before they need them: a snapshot carries the
  credential vault but never the key that opens it, and what a restore
  replaces.

### Fixed

- **The API reference was missing every authentication route.** `/api/auth/session`,
  `/api/auth/sessions`, `/api/auth/logout`, `/api/auth/mode`,
  `/api/auth/pairings`, `/api/auth/token/rotate` and the device-revoke route
  are served by the daemon and were absent from the published OpenAPI
  document — the surface an API consumer needs first. The generator records
  routes against an empty dependency set, and auth registration returned
  early when the gate was absent, so nine patterns were never recorded. CI
  compared the committed file against that same generator, so it stayed
  green. The spec now lists 338 paths, and the generator's
  `x-undocumented` names the two non-JSON routes it still leaves out
  (`/pair`, `/preview/**`) instead of implying they do not exist.
- **Restoring a backup could destroy what it was restoring.** Pin
  attachments and pi session files were deleted first and copied second,
  with no rollback, after the database had already been swapped in — so a
  copy that failed partway (full disk, one unreadable file) left the live
  files gone and half rebuilt. Each directory is now built beside the live
  one and renamed into place, and the credential vault a restore replaces is
  kept as `credentials.json.replaced`.
- **Eight accepted decisions were still listed as proposals.** ADRs 0110,
  0120, 0135, 0142, 0150, 0153, 0155 and 0157 said `accepted` in their own
  file and `proposed` in the index, which reads as half the recent work being
  speculative.
- **`CHANGELOG.md` told agents to do what the commit hook refuses.** Its own
  header asked for an entry in `[Unreleased]`; ADR-0105 assembles this file
  from `docs/changelog.d/` fragments and the hook blocks the direct edit.
  The header now describes the fragment flow, and the hook allows a
  correction to the preamble above the first version heading.
- Three broken links in `docs/architecture/`, a subsystem file
  (`devservers.md`) that the architecture index never linked, a source
  comment pointing at a handoff topic that does not exist, and three forms
  that did not opt out of native browser validation.
