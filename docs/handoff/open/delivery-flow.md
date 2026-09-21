# Delivery flow

Plan: docs/plans/delivery-flow.md
Research: docs/benchmarks/2026-09-21-delivery-governance.md
D0 evidence: docs/plans/delivery-flow-evidence.md
D0 surface contract: docs/plans/delivery-flow-design.md

## Next

- Implement D2: explicitly associate the local PiCode environment, record deployment attempts and verify running revision/provenance under ADR-0170.
- Complete D2 and measure the observation pilot; D3/D4 execution authority requires separate owner decisions.

## Debts

- [ ] Native authenticated sessions for the nine vendors have not exercised the delivery command/MCP end to end; common catalog-principal and CLI-to-daemon fixtures are covered. The existing tool picker has no delivery option; explicit MCP selection uses client configuration.
- [ ] Historical land/deploy queue wait has no authoritative start timestamp; D0 measured seven source lookups only. Capture owner-task time in the D1+D2 pilot and operation wait after D3/D4.
- [ ] D1b reports incomplete coverage after 1,000 receipt files/changes or 32 checkout statuses; add paginated older evidence and on-demand omitted checkout reads before scaling beyond the measured local pilot. No evidence is pruned.
- [ ] Native Windows ACL inheritance and physical mobile behavior were not exercised by the Linux scratch observation tests.
