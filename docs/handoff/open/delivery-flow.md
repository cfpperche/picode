# Delivery flow

Plan: docs/plans/delivery-flow.md
Research: docs/benchmarks/2026-09-21-delivery-governance.md
D0 evidence: docs/plans/delivery-flow-evidence.md
D0 surface contract: docs/plans/delivery-flow-design.md

## Next

- Decide proposed ADR-0170 (read protocol, producer receipts and explicit local observer), then implement D1 from the D0 contract.
- Deliver D1+D2 observation first; D3/D4 execution authority requires separate owner decisions.

## Debts

- [ ] Historical land/deploy queue wait has no authoritative start timestamp; D0 measured seven source lookups only. Capture owner-task time in the D1+D2 pilot and operation wait after D3/D4.
