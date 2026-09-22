# 2026-09-21 — feat/guest-add-bar: Providers bar carries each CLI's own Add as primary
Shipped: every CLI's Providers bar now carries the CLI's own Add as the bar's
primary — for guests (omp included) it opens the key dialog whose select lists
that CLI's declared providers, beside the Custom provider door (omp). The
per-row Add API key buttons are gone where the bar offers the primary; they
remain only where the bar cannot (no providers / no add). Follow-up to
ADR-0169's surface, closing the vocabulary gap the owner flagged: pi had
"Add provider", omp had only per-row adds.
Files: web/browser + web/mobile CliCredentials.jsx (barPrimary rule +
perProviderAdd suppression).
Verified: ci-scoped green (fmt, vet, hooks, test-js, build) at commit ca64b918.
visual-review: PASS (2 stills: bar with primary; dialog listing omp's 11 providers).
Not done / debts: none.
Merge: done — landed at `e0bd79ab` (full ci green); serving in `0.4.0+b0f4ba4`. Branch and
worktree removed.
