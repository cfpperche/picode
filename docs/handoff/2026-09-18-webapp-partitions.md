# 2026-09-18 — webapp-partitions: one WebView2 folder per installed web app

Shipped: ADR-0153 + migration 056 (`webapps.partitioned`, installs born
after it = 1) + the shell derivation: tab id `app-<webappId>` →
`WebView2\webapps\<webappId>` (btab.rs uses `browserlab::profile_for(id)`;
ids outside the minted shape fall back to the shared profile — unit tests
in browserlab.rs, compile-verified via `cargo xwin build/test`; the repo
has no wine/Windows runner that executes shell tests, so the assertions
run on a Windows `cargo test`). Per-profile settings (autofill, download
folder) already apply per-controller and land in each partition. Clear
browsing data clears the work profile only. CDP/permissions unchanged.
Verified: go store+server suites (partitioned round-trip; legacy default
0), invariant rows untouched, ci-scoped PASS, cargo xwin build clean.
visual-review: n/a (storage layout; no UI surface).
Owner verification on Windows: install two accounts of one service,
login survives restart, clear-data does not sign apps out.
Merge: fast-forward ready. Plan: docs/plans/installed-webapps-v2.md
(feature 2 of 2 — the plan closes with this branch).

## Next up

- Owner on Windows: the isolation proof — same service twice, restart, clear-data.

## Debts

- docs/handoff/open/installed-webapps.md: standalone window stays refused; a per-app "clear this app's data" action is a future want (ADR-0153 consequence), not planned.
