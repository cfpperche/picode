# 2026-09-20 — feat/omp-session-isolation: isolate Omp agent sessions

Shipped: Omp workspace agents launch with a durable per-agent `--session-dir`; standalone Omp terminals retain vendor defaults. Documented native session-scope limits for other CLIs.
Verified: `make close` passed; full Go gates ran in four server shards, with the longest shard at 95s. Targeted Omp isolation test passed. After deploy, the owner confirmed two Omp agents in one workspace show isolated `/resume` sections.
visual-review: n/a
Not done / debts: Existing Omp sessions are not migrated or reassigned; only sessions created after launch use the private directory.
Merge: landed on `main` and deployed.
