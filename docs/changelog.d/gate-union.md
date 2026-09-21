### Fixed

- **The ADR-0048 mutation gate no longer has the two blind spots it shipped
  with.** It read SQL as text, so a statement assembled from a package-level
  constant, or a table name its pattern could not spell, went unchecked — both
  were filed as a known limitation rather than fixed. A write is now either
  SQL the test can read *or* a call that runs one, which between them have no
  gap: every write in the store reaches SQLite through one `Exec` or the
  other. `VacuumInto` joins the listed exceptions.
