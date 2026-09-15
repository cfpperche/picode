# 2026-09-15 — terminal-agents-principals

The ADR for the gap the owner asked about: CLI agents never appear in
Agent permissions.

## Done

- ADR-0143 (accepted): identity tuple agent → terminal → unmanaged, grants
  keyed `browser.policy.term:<id>` beside the existing bare agent key,
  unmanaged stays read-only by construction and the UI says so.
- `docs/handoff/open/work-browser-tabs.md` carries the implementation order
  (extension → resolver → listing → UI → tests per decision-table row).
- Docs-only: `ci-scoped` PASS, no deploy (no UI or binary change).

## Next up

- Execute the five steps above in a fresh session; ADR-0143 is the spec.
- Unchanged behind it: the Ask prompt (closes Browser permissions), the
  tab-strip crash on "new browser tab", then v2a (unused sites).
