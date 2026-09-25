# 2026-09-25 — llama-owner-calls: killed supervisor's group cleared, lost downloads released, job history bounded

Shipped (7acf8a8a4, c57f6709f): the owner's three calls on the items left by
feat/llama-debts, recorded as ADR-0090 and ADR-0083 amendments (2026-09-25);
`open/llama.md` items flipped to [x]. (1) The supervisor reports the router PID
(its group) on fd 3 (`PICODE_LLAMA_ROUTER_FD`, close-on-exec, env removed so the
router never inherits it); the daemon reads it (5 s deadline) and after the
supervisor exits runs `clearOrphanedGroup`: waits ≤3 s for the leader to vanish,
SIGKILLs the group only if non-zombie members remain, never while a process holds
the group's number as PID. (2) A lost download (after restart, or unknown) that
the server doesn't list for ≥3 reads over ≥1 min (a failed read resets) becomes
interrupted and releases the model; nothing is sent to the server. (3)
`store.PruneLlamaJobs` at llamajob start deletes finished jobs older than 30 days
beyond the newest 500 (silent mutator with reason; Activity shows newest 50).
- Tests: killed supervisor → child lives until the clear, then dies; live leader
  untouched; router doesn't inherit fd 3/env; lost download released, grace kept,
  a running one followed; prune covers age/keep/active.
- Adversarial review of 7acf8a8a4: 2 real defects (grace too short for a timed-out
  send; fd 3 + env leaking to the router), fixed in c57f6709f; plus test gaps.
  docs-site/guide/llama.md, llama-manager.md and changelog fragment updated.

## Debts

- The `startProcess` wiring (ExtraFiles + `go clearOrphanedGroup`) has no test; the tests call the helpers directly. Still unhandled: daemon and supervisor killed together (group id not persisted).
- `nowUTC` RFC3339Nano trims trailing zeros, so string order of `created_at` can misorder rows within one second (LlamaJobs order, prune cutoff tie; cosmetic).
