# Changelog fragments (ADR-0105)

`CHANGELOG.md` is assembled, never edited on a branch. Every branch that
changes something user-visible leaves **one file here**, named after the
branch (`<branch-slug>.md`), holding the Keep a Changelog sections it needs:

```markdown
### Added
- **Automations: several schedules per automation.** One line per change,
  written for the user, present tense.

### Fixed
- Sidebar checklist no longer shows `undefined/undefined`.
```

Valid sections: Added, Changed, Deprecated, Removed, Fixed, Security.

`make changelog` folds every fragment into `[Unreleased]` (newest first
inside each section), deletes the fragments and stages both — run it on
`main` before a release. The pre-commit hook refuses a `CHANGELOG.md` edit
that is not that assembly, so two branches never fight over the same lines.
