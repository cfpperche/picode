# 2026-09-17 — feat/connectors-parity: Connector drivers registry (Phase 0, ADR-0150)
Shipped: Phase 0 of connector parity (ADR-0150 accepted, owner-approved
2026-09-17 with 4 decisions). CONNECTOR_DRIVERS registry in
web/shared/domain/integrations.js replaces the fixed CLI_CONNECTORS list
(same behavior: only pi); /api/mcp accepts cli= and requireConnectorDriver
refuses a driverless CLI with 400 (GET/POST/PATCH/DELETE/import/auth/logout);
internal/mcp untouched.
Verified: go build ./...; go test ./internal/server -run TestMCP;
go test ./internal/mcp/...; node --test web/shared/domain/integrations.test.js
(6 pass); make web ok; make close green (GO/WEB/METADATA).
visual-review: n/a (registry refactor; pi panel untouched; non-pi
placeholders unchanged — no screenshots taken)
Merge: fast-forward ready

## Next up

- Phase 1 — guest drivers in internal/connectors, starting with Claude Code
  (project .mcp.json; user scope via claude mcp add) and Codex (TOML
  ~/.codex/config.toml), with golden-file tests per codec.

## Debts

- TOML/YAML formats will need 2 new deps to justify in the PR
  (docs/handoff/open/connectors-parity.md).
