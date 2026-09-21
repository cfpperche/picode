# 2026-09-21 — delivery-clarity: clarify Delivery states
Shipped: Delivery now explains integration and validation states in desktop and mobile detail views; the public guide starts with a visual glossary.
Verified: `make ci-scoped` passed fmt, vet, hooks, JS tests, builds, docs checks and Vale. Domain tests passed. Scratch captures were reviewed for desktop and mobile populated, empty, blocked, error, detail and recovery states.
visual-review: PASS
Not done / debts: D2 deployment observation remains planned. Coverage caps, Windows ACL inheritance and physical mobile behavior remain documented debts.
Merge: fast-forward ready after reconciling the current main.

## Next up

- Implement D2 deployment observation and provenance under ADR-0170.

## Debts

- Continue the documented D1b evidence coverage and native environment validation work in `docs/handoff/open/delivery-flow.md`.
