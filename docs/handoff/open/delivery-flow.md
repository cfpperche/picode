# Delivery flow

Plan: docs/plans/delivery-flow.md
Research: docs/benchmarks/2026-09-21-delivery-governance.md
D0 evidence: docs/plans/delivery-flow-evidence.md
D0 surface contract: docs/plans/delivery-flow-design.md

## Next

- Decide proposed ADR-0170 (read protocol, producer receipts and explicit local observer), then implement D1b observation and its screen from D0, joining ADR-0171 declarations.
- Deliver D1+D2 observation first; D3/D4 execution authority requires separate owner decisions.

## Debts

- [ ] Native authenticated sessions for the nine vendors have not exercised the delivery command/MCP end to end; common catalog-principal and CLI-to-daemon fixtures are covered. The existing tool picker has no delivery option; explicit MCP selection uses client configuration.
- [ ] Historical land/deploy queue wait has no authoritative start timestamp; D0 measured seven source lookups only. Capture owner-task time in the D1+D2 pilot and operation wait after D3/D4.
