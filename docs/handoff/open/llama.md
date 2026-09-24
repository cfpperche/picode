# llama.cpp manager

## Next

- llama delivery 3 live validation; owned-service ARM64 acceptance.

## Debts

- [ ] **`TestConnectionTimeoutAndTransport` failed once on the ubuntu leg of a
  full-matrix run** (2026-09-24, the manual run over `main`):
  `Get "http://127.0.0.1:32771/models": dial tcp … connection refused
  (Client.Timeout exceeded …)` — the test's own listener was not up when the
  client asked. It then passed 10 runs in a row under the gate's `PATH`, so the
  shape is a timing race in the fixture (or a listener that the OS refused under
  load), not a product fault. Nothing else in `internal/llama` failed.
- llama: ARM64 hardware and GPU / non-b10809 cancellation unverified; an unknown download with an absent model keeps its reservation; history pruning deferred.
