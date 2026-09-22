### Added

- **Omp model roles in Settings.** `#/clis/omp/settings` now edits Omp's own
  role map: one row per role the CLI has (DEFAULT, SMOL, SLOW, VISION, PLAN,
  COMMIT, TINY, MEMORY, TASK, ADVISOR and the five kind roles), plus any role
  you invented, each with a model picker fed by Omp itself and a thinking-level
  select. **Fallbacks** edits `retry.fallbackChains` as ordered lists, and
  **Quick-switch cycle** edits `cycleOrder` — the list Omp's own hub draws as
  `⟳ N` beside a role. Both layers, including the workspace file `omp config
  set` cannot reach. New rows for `modelRoleStorage`, `defaultThinkingLevel`
  and Omp's eight retry knobs (ADR-0181).
- `GET /api/cli-models?cli=&workspace=` asks a CLI which models it can reach,
  in the workspace being edited. Omp only; it runs when a picker opens.

### Changed

- A settings row may now be a model selector at a vendor-supplied path, or an
  ordered list of strings. Everything else is unchanged: a key nobody declares
  is never written, a document that does not parse is never overwritten, and
  comments and unknown keys survive a save.

### Fixed

- **Guest CLI settings never saved from a browser.** The pane handed a raw object to
  `fetch`, so every save since 2026-09-20 answered "invalid request body" (found by
  driving a real write on a scratch instance). The body is JSON-encoded now, for every
  guest CLI's Settings pane.
- A settings value carrying a line break is refused by name instead of being
  written as a YAML block the next save would duplicate.
- Writing a three-level YAML key whose middle map was missing appended a second
  top-level block instead of joining the existing one.
