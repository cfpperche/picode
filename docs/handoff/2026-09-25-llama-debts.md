# 2026-09-25 — llama-debts: hashes leave the lock, ended downloads are forgotten, kill before reap

Shipped (37bccb330, 83a1ba9d5): cleanup/cache hashes are warmed outside the
service lock and memoized by file identity including change time
(`internal/llamaservice/hash_memo.go`); the symlink refusal still runs first and
`verifyRelease` (guards each start) never uses the memo. `doc.Downloads` no
longer leaks: llamajob `ended()` reports every non-active download (Abandon
included), `ObserveDownload` forgets the list on failure and on a success for a
stopped or moved service, and `New()` drops lists of jobs that are over or
missing. Supervisor now does waitid WNOWAIT → kill group → reap, closing the
PID-reuse window when all group members are dead. llama-manager.md updated.

- Pays the hash-under-lock debt of `2026-09-23-llama-jobs-resolve.md`; owner
  asked to work the llama.cpp debts on 2026-09-25.
- Tests: Snapshot answers while a cleanup hash is blocked; an in-place rewrite
  with mtime restored is rehashed. `TestSupervisorClosesDescendantsWhenTheRouterDies`
  would pass on the old order too (its comment says so). -race green; windows/darwin vet clean.
- Adversarial review: 1 medium + 2 low in the first commit, fixed in the second.

## Debts

- Owner decisions in `open/llama.md` (unchecked): SIGKILL of the supervisor
  leaves the router's per-model children (ADR-0090); an unknown download with
  an absent model is released only by Abandon (ADR-0083); `llama_jobs` rows are
  never pruned (ADR-0083). Still unverifiable here: ARM64 hardware, GPU.
