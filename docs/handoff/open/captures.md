# Docs captures and videos

## Next

- Compose registration ADR; ADR-0064 cadence; docs-video recapture policy.

## Debts

- Capture integration (ADR-0054): no real emitter-to-RPC run, no slow-consumer/cancellation matrix.
- `app-inspector.png` is older than the Servers panel v2 (PNG last changed 18:54, the UI landed 20:42) and `docs-check --strict` passes anyway: the manifest rewrite that rode the 21:37 deploy (`8395deb0`) recorded the *current* inputs for every capture, so a capture that was never retaken became "current". Recapturing is `make docs-shots`; the checker cannot be trusted for this one until its manifest stops recording inputs it did not photograph. Measured 2026-09-17.
