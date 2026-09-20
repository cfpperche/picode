# 2026-09-19 — cli-principals: bind CLI terminals as managed principals
Shipped: ADR-0159; `grant.Principal`; `managed_clis`; `POST/GET
/api/workspaces/{id}/principals`; `DELETE /api/managed-clis/{id}`. Guest
TUI is `term:<id>`, never an `agents` row, never `Runtime.Start`.
Verified: `make ci-scoped` PASS; store decision table + HTTP tests.
Blind spot: no UI; not exercised on a live workspace.
visual-review: n/a (API + feed only)
Not done: Fatia 1 (fleet / Inbox / push). Topic: `open/managed-principals.md`.
Merge: catch main, then fast-forward from the root.

## Next up

- Fatia 1: fleet, Inbox “Open terminal”, deploy readiness for bound CLIs
