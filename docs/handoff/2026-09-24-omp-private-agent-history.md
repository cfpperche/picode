# 2026-09-24 — omp-private-agent-history: find private Omp sessions

Shipped: Agent history resolves PiCode-launched Omp transcripts in the agent's private session directory, including lookup by session ID when the recorded path is missing. The six stopped pilot agents were removed without purging their session files; the active agent was left intact.
Verified: `make ci-scoped` and `make close` passed. The locator regression test covers the private Omp directory; this branch was not deployed, so the live Agent History view was not verified with the new binary.
visual-review: n/a
Not done / debts: No code debt recorded for this branch.
Merge: fast-forward ready from `feat/omp-private-agent-history`.
