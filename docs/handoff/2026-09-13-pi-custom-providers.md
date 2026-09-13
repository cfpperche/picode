# 2026-09-13 — pi-custom-providers: Custom endpoint in the Providers GUI

Shipped: Add provider → **Custom endpoint** (ADR-0129). Form (name, base URL, key, model ids, Advanced) writes pi's `~/.pi/agent/models.json` under the schema's `providers` wrapper; the key lands in `auth.json` via the existing sign-in path. `PUT/DELETE /api/providers/custom/{id}` merge by id in `internal/catalog/modelsjson.go`; catalog rows carry `custom` + editable shape, unsigned definitions stay visible. Roster: `custom` badge, Edit endpoint / Sign out / Remove endpoint. Shared schema + helpers (`web/shared/domain/customProviders.js`), both apps (ADR-0072).
Verified: `make ci-scoped` green (go, test-js, build, docs, openapi regenerated). Decision-table rows covered by Go + JS tests. Scratch instance end-to-end: save → `pi --list-models` lists the gateway, `pi auth check` answers `ready/api_key`; edit keeps the key; delete cleans both files. The `providers` wrapper bug was caught by a real pi run, not unit tests.
visual-review: PASS (desktop form/roster/menu/edit + mobile sheet read; overlayAudit ok; card 5/5)
Not done / debts: P1+ phases from the approved plan (Load models from `GET /v1/models`, Verify with a real cheap call) are future work; custom providers are Pi-only by ADR-0103.
Merge: fast-forward ready (run `make close` again after merging main; main moved).

## Debts

- Verify-with-pi still answers from credential presence, not a real request (pre-existing); the cheap-call variant is P4 of the approved plan (`docs/handoff/open/providers-custom.md`).
