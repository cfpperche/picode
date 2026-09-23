# 2026-09-23 — llama-pane-honest: the llama.cpp pane says what is wrong, fast

Shipped (branch 2 of today's adversarial llama.cpp review): `/api/llama` bounded by the request + 4 s, no
capabilities after a failed list, normalized `endpoint` in the answer; `ConnectionFailure` codes refused / dns /
tls / cancelled / timeout; LlamaPanel (desktop + mobile) — Save not blocked by a check, job ends queue a re-check,
other triggers dropped while one runs, honest empty state, HF pick "Reading…" + last-pick-wins, quant rows as a
grid; LlamaService single read + queued rerun, read error clears; HF bounded read, plain errors, size-less quants
kept; `startArgs` IPv6; Add provider no longer says "Signed in" for llama.cpp.
Correction recorded (ADR-0009 amendment, cache comment, two notes): the closed-port hang is of processes launched
from the agent session; the owner's daemon read a cold catalog in 0.43 s.
Verified: `make ci-scoped` PASS; visual-review PASS on scratch after four rounds (a double first check my queue
introduced, and the pre-existing quant-row overlap, found and fixed); check ≈4 s with one request, overlayAudit ok.

## Next up

- Branch 3 of the review (needs the owner's decision): llama job states / service ownership — ADR-0083/0090
