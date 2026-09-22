# Delivery flow

Plan: docs/plans/delivery-flow.md
Research: docs/benchmarks/2026-09-21-delivery-governance.md
D0 evidence: docs/plans/delivery-flow-evidence.md
D0 surface contract: docs/plans/delivery-flow-design.md

## Next


## Debts

- [x] The tool picker had no delivery option, so it could not be switched on for an agent whose launch settings had been customized — a launch that never touched them already received every family — paid 2026-09-21 (feat/delivery-tool): `PICODE_TOOL_FAMILIES` offers **Delivery**, `internal/server/cli_tools_test.go` holds that list against `mcptool.FamilyNames()`, and the scratch proved the path form → config → plan (`mcp_servers.picode-delivery.args=["mcp","delivery"]`). `packages/pi-delivery` gives an agent whose CLI is Pi the same tool, held against the MCP schema by `internal/mcptool/delivery_test.go`.
- [ ] Native authenticated sessions for the nine vendors have not exercised the delivery command/MCP end to end; common catalog-principal and CLI-to-daemon fixtures are covered. An unbound terminal's inherited identity was exercised on the scratch (`capabilities` → `register` → `request-review` → `show`, principal `term:<id>`), and the MCP face over stdio with a real `tools/call` (`serverInfo.name=picode-delivery`) — but no vendor CLI process has run the tool itself.
- [ ] Historical land/deploy queue wait has no authoritative start timestamp; D0 measured seven source lookups only. Capture owner-task time in the observation pilot and operation wait after D3/D4.
- [ ] D1b reports incomplete coverage after 1,000 receipt files/changes or 32 checkout statuses; add paginated older evidence and on-demand omitted checkout reads before scaling beyond the measured local pilot. No evidence is pruned.
- [ ] Native Windows ACL inheritance and physical mobile behavior were not exercised by the Linux scratch observation tests.
