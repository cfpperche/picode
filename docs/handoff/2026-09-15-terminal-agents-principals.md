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
