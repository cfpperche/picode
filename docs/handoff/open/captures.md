# Docs captures and videos

## Next

- Compose registration ADR; ADR-0064 cadence; docs-video recapture policy.

## Debts

- Capture integration (ADR-0054): no real emitter-to-RPC run, no slow-consumer/cancellation matrix.
- The Servers panel (rail → Servers) has no public capture: `app-inspector` photographs `/browser/` scoped to `#inspector` and waits for the *Uncommitted* header — the Changes tab. Refreshing for the panel's v2 changed nothing (2026-09-17: 5 surfaces, only `app-canvas` changed; `app-inspector` differed by 122 px, kept under the 128-px budget). Capturing the panel is a new scenario, not a refresh.
- `docs-shots` keeps the committed image when a fresh capture differs by ≤128 px, so a small but real change to a captured surface ships without an updated image (measured: `app-inspector` 122 px, 2026-09-17). Decide whether the budget should be per-surface.
- `make docs-shots` leaks its owned fixture whenever a surface fails: `main()`'s `catch` calls `process.exit(1)`, which skips the `finally { release?.() }`, so the daemon — plus its `tmux -S /tmp/picode-docs-fixture-<port>/tmux.sock` server and its data dir — stays alive with no run to use it. Seven `/tmp/picode-docs-shots-fixture` daemons were alive on 2026-09-21 (started 17:33→19:00) with no `docs-shots.mjs` running; they are detached (`ppid` 308, no port in the log), so they cannot be attributed to a session and were left alone. One `finally` that kills before exiting pays this.
