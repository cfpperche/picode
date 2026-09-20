# 2026-09-20 — feat/omp-session-isolation: isolate Omp agent sessions

Shipped: Omp workspace agents launch with a durable per-agent `--session-dir`; standalone Omp terminals retain vendor defaults. Documented native session-scope limits for other CLIs.
Verified: `make close` passed; full Go gates ran in four server shards, with the longest shard at 95s. Targeted Omp isolation test passed. The native Omp picker was not exercised in a live two-agent workspace.
visual-review: n/a
Not done / debts: Existing Omp sessions are not migrated or reassigned; only sessions created after launch use the private directory.
Merge: fast-forward ready.

## Next up

- Live two-agent Omp `/resume` acceptance after deploy.
