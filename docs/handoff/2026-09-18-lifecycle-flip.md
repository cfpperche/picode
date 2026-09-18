# 2026-09-18 — lifecycle-flip

Root cause found for the one-request lifecycle flip (all four unclassified
CLIs lost their ••• capabilities for a single GET, then healed).

## Mechanism (proved, not guessed)
- GET /api/clis has NO cache: describeCLI resolves the executable fresh per
  request.
- The four CLIs without an npm package (grok, hermes, muse, omp) carry
  launchers/symlinks the vendor self-updaters rewrite IN PLACE.
- A Stat landing inside that swap window → installed=false → ForMissing
  returns ok=false (no npm package) → lifecycle all-false, uninstall "".
- Next request after the swap: capabilities return. Matches the observed
  flip byte-for-byte; the owner was updating those CLIs that morning.

## Done
- resolveInstalledCLI: one retry (~150 ms, package var for tests) before
  declaring a CLI absent; genuine uninstalls still fail both attempts.
- Test: retry rides out a launcher materialized mid-flight; real absence
  still refused. Architecture doc documents the transient.
