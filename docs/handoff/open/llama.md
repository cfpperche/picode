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
- [x] Cleanup hashed selected cache files under the service lock (2026-09-23
  `llama-jobs-resolve` debt): paid 2026-09-25 by `feat/llama-debts` — hashes
  are taken before the lock and kept by file identity (`hash_memo.go`).
- [x] A download's starting file list (`doc.Downloads`) stayed forever when the
  download did not succeed: paid 2026-09-25 by `feat/llama-debts`.
- [x] Supervisor kill after the router is reaped could race PID reuse: paid
  2026-09-25 by `feat/llama-debts` (group killed before the reap).
- [x] A SIGKILL of the llama supervisor itself leaves the router's per-model
  children running (2026-09-25). Closing it moves the process boundary
  (ADR-0090: supervisor reports the router PID to PiCode, or a cgroup per
  service) — the owner's call.
- [x] An unknown download whose model is absent is released only by Abandon —
  a documented choice (docs/plans/llama-manager.md, ADR-0083), not a leak. An
  automatic release after N quiet polls would amend ADR-0083 — the owner's call.
- [x] `llama_jobs` rows are never deleted (~1–3 KB each). A retention rule
  (e.g. non-active rows older than 30 days beyond the newest 500) is a
  persistence decision (ADR-0083) — the owner's call.
  The three above were decided by the owner on 2026-09-25 and paid by
  `feat/llama-owner-calls` (ADR-0083 and ADR-0090 amendments).
