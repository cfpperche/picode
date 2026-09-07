# 2026-09-07 — feat/adr-agent-protocol-deferral: ADR-0091 records the pi-only agent protocol refusal

Shipped: ADR-0091 (`docs/decisions/0091-pi-only-agents-acp-waits.md`) — PiCode's
first-class agent stays pi (own RPC for managed agents, TUI for manual); guest
CLIs remain terminals with ADR-0056 tier-1 sensors; no agent-protocol client
(ACP, Codex app-server, vendor SDKs) and no guest-agent promotion until the
market converges on one standard (named re-measure trigger, no date). Index row
added; handoff.md updated in place (pi-only line; Next up 6 no longer owes a
parity ADR). Converts 0056's open ACP deferral into a deliberate refusal.
Verified: make close — hooks selftest ok, ci-scoped PASS (docs-only diff).
visual-review: n/a
Not done / debts: none new; owner said "nothing more to do" — no implementation
follows from this ADR until the trigger fires.
Merge: fast-forward ready
