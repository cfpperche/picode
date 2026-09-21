# 2026-09-21 — delivery-agent-contract: shared agent delivery declarations

Implemented: ADR-0171, `picode delivery`, optional MCP delivery family and
`POST /api/delivery/tool`; SQLite declarations, retry receipts and events commit
together. Review requests remain declarations; updates clear them. Source
revision observations do not claim checks, integration or publication evidence.
Verified: `make close` (fmt, vet, hooks, 21 Go packages, docs build and Vale),
focused bound agent/terminal identity test; implementation session also ran
`TestDelivery*` with the race detector across four packages and the event invariant.
Decision table: `docs/plans/delivery-agent-interface.md` maps the conditions to tests.
visual-review: n/a — CLI/MCP/API and public command documentation; no app UI changed.
Limits: fixtures exercise catalog principals, not authenticated vendor sessions;
the fixed tool picker has no delivery option. Durable follow-ups remain in
`docs/handoff/open/delivery-flow.md`; queues, execution and the delivery screen
are separate slices. Public capture staleness was an advisory in docs-check.
Merge: fast-forward ready; main CI and deployment are outside this branch close.
