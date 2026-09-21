# Docs captures and videos

## Next

- Compose registration ADR; ADR-0064 cadence; docs-video recapture policy.

## Debts

- Capture integration (ADR-0054): no real emitter-to-RPC run, no slow-consumer/cancellation matrix.
- The Servers panel (rail → Servers) has no public capture: `app-inspector` photographs `/browser/` scoped to `#inspector` and waits for the *Uncommitted* header — the Changes tab. Refreshing for the panel's v2 changed nothing (2026-09-17: 5 surfaces, only `app-canvas` changed; `app-inspector` differed by 122 px, kept under the 128-px budget). Capturing the panel is a new scenario, not a refresh.
- `docs-shots` keeps the committed image when a fresh capture differs by ≤128 px, so a small but real change to a captured surface ships without an updated image (measured: `app-inspector` 122 px, 2026-09-17). Decide whether the budget should be per-surface.
- [x] `make docs-shots` leaked its owned fixture whenever a surface failed: `main()`'s `catch` called `process.exit(1)`, which skips the `finally { release?.() }`, so the daemon — plus its `tmux -S /tmp/picode-docs-fixture-<port>/tmux.sock` server and its data dir — stayed alive with no run to use it (seven daemons and thirty stale dirs on 2026-09-21, and a whole browser per failed run: 13 chrome processes, measured). Paid 2026-09-21 (feat/docs-shots-leak): the fixture ends itself on SIGINT/SIGTERM (`shutdown` → `tmux.Manager.KillServer`, then its dir), and the harness closes its session in `main`'s `finally` while keeping the exit code 1. Both failure paths re-run by hand (fixtures 7→7, chromes 11→11); `TestShutdownEndsTheServerBeforeTheDirectory` and the `KillServer` end/refuse tests cover it.
