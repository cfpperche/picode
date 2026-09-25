# 2026-09-25 — omp-reader-scope: main back to green after cli-automation-more

`feat/cli-automation-more` landed with `make ci` red on main (3f9699d7b): adding `omp` to `doorReaderCLI` also changed Omp's fork (its task went pending for a paste instead of riding the launch), which `TestForkAgentOmpKeepsTheTaskOnItsLaunch` and `TestForkAgentRefusals/a_task_over_the_launch_limit` caught. `make close` had failed and I landed anyway — `land` checks the fast-forward, not the close result; read the close output before landing.
Fix: `omp` leaves `doorReaderCLI`; `unattendedReaderCLI` lets only unattended senders (ADR-0217 start runs) use Omp's reader. Fork, missions and attach delivery are exactly as before. `TestOmpReaderIsUnattendedOnly` pins it; the durable item (measure those paths before widening) is in `open/managed-principals.md`.
Verified: `make close` green here, full `make ci` on main after the land. The Omp start runs measured earlier used the same unattended path, so they stand.
