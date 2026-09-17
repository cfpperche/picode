# 2026-09-17 — omp-sessions: Omp lists its own sessions

Shipped: `OmpSource` over `~/.omp/agent/sessions` (`$PI_CODING_AGENT_DIR`
honored) — schema-v3 header for id/cwd, newest `title` record for the name,
provider-qualified `model_change` for the model, first user text as
preview; rows carry `--resume <id>`, verified headless against omp 18.2.4.
Sessions tab live for omp; pin-on-stop joins muse/agy/omp in the wrapper-
less table. Format probed against real sessions, not docs — findings and
the Fatia 3–6 gates updated in `docs/plans/omp-cli.md` (Fatia 4 measured
**GO**: the `-e` extension fires and keeps `PICODE_TERM_ID`).
Verified: `make close` green (fmt,vet,go,docs); scratch `ompb` desktop —
Sessions tab read with 3 real sessions, honest ⋯ menu ("No other CLI can
receive this session yet") and the one-line empty state; overlayAudit ok.
visual-review: PASS (omp-sessions-tab.png, omp-session-menu.png,
omp-sessions-empty.png; card 5/5)
Not done / debts: omp Reader/Writer/Prompter stay out until Fatia 6
(Continue in… shows the honest one-liner); Bun upgraded to 1.4.2 on this
machine — the Fatia 1 Check-setup debt is paid.
Merge: fast-forward ready.

## Next up

- Omp Fatia 3 (launch): PATH wrapper + presence lease, `PI_CODING_AGENT_DIR` added to launcher-owned env keys, `OMP_MAINT` passthrough list. Plan: `docs/plans/omp-cli.md`.

## Debts

- `TestCLIRestartPreparationFailureAndWorkspaceCleanup` is flaky on this machine — fails on `main` too (tmux PanePID race, 2026-09-17), unrelated to omp.
