# 2026-09-15 — browser-permissions-store

First half of Browser permissions, as agreed: the data foundation the shell
will feed. Approved scope and the measured API notes are in
`docs/handoff/open/work-browser-tabs.md`.

## Done

- Migration 050 + `internal/store/browser_permissions.go`: one standing per
  (origin, kind), upsert in place, closed kind and decision vocabularies,
  list by kind/origin, delete, per-kind and full clear. Table test covers
  every row of that decision table, plus three rows in the events invariant.
- `internal/server/browser_permissions.go`: the four endpoints, tested
  (accepts a real decision, refuses an unknown kind or decision word, lists
  by kind, per-kind clear leaves the rest).
- OpenAPI regenerated (the new routes).

## Not wired yet — deliberately

- Nothing produces standings: the shell's `PermissionRequested` handler and
  the Site settings dialog are the next slice (`attach_permission_handler`,
  the Ask prompt with the deferral, `btab_set_permission_policy`). Until
  then this is storage with tests and no surface, which is why it is its own
  commit.
