# llama.cpp manager

## Next

- llama delivery 3 live validation; owned-service ARM64 acceptance.

## Debts

- [x] **`TestConnectionTimeoutAndTransport` failed on the ubuntu leg of two
  full-matrix runs** (2026-09-24) — not a flake, as the first record called it:
  the fixture gave the client a 1 ms timeout while the handler slept 20 ms, so
  on a loaded runner the *dial* raced the budget and the failure classified as
  the connection rather than the response. The margins are 100 ms against a
  200 ms handler now, and the test is 20/20 alone and 0/4 failing under four
  parallel loops.
- llama: ARM64 hardware and GPU / non-b10809 cancellation unverified; an unknown download with an absent model keeps its reservation; history pruning deferred.
