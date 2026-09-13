# 2026-09-13 — pi-custom-providers: Custom endpoint in the Providers GUI

Shipped: Add provider → **Custom endpoint** (ADR-0129). Form (name, base URL, key, model ids, Advanced) writes pi's `~/.pi/agent/models.json` under the schema's `providers` wrapper; the key lands in `auth.json` via the existing sign-in path. `PUT/DELETE /api/providers/custom/{id}` merge by id in `internal/catalog/modelsjson.go`; catalog rows carry `custom` + editable shape, unsigned definitions stay visible. Roster: `custom` badge, Edit endpoint / Sign out / Remove endpoint. Shared schema + helpers (`web/shared/domain/customProviders.js`), both apps (ADR-0072).
Verified: `make ci-scoped` green (go, test-js, build, docs, openapi regenerated). Decision-table rows covered by Go + JS tests. Scratch end-to-end: save → `pi --list-models` lists the gateway, `pi auth check` answers `ready/api_key`; edit keeps the key; delete cleans both files. The `providers` wrapper bug was caught by a real pi run, not unit tests.
visual-review: PASS (desktop form/roster/menu/edit + mobile sheet read; overlayAudit ok; card 5/5)
Not done / debts: P1+ phases (Load models, Verify with a real cheap call) are future work — `docs/handoff/open/providers-custom.md`.
Merge: fast-forward ready (run `make close` again after merging main; main moved).
