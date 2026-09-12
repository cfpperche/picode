# 2026-09-10 — remove-browser-surface: the browser surface is gone

Shipped: ADR-0117 (supersedes ADR-0114/0115 — both kept as history with
Status lines updated). Removed: the Browser view of the agent tab,
`GET /ws/browser`, the rendezvous discovery in `internal/rpc`,
`POST /api/agents/{id}/browser-input`, BrowserSurface + shared helpers,
phase-2 `/browser-input` additions to pi-browser-capture, the never-released
changelog fragments, and `docs/architecture/browser-surface.md`.
Stays: capture sidecar + chat pill (ADR-0082), the agent's `agent_browser`
tool, the benchmark study (evidence).

Verified: grep for orphan references clean; go vet/build/tests green;
`make ci-scoped` PASS (32 paths); close green, ff-ready. No changelog entry:
the feature never shipped in a release — its fragments were deleted.

visual-review: n/a (removal; production still runs the old binary until the
owner's next deploy — surface remains visible there until then)
Not done / debts: none from this branch. Lesson recorded in ADR-0117: the
hex-slice trap and the self-confirming E2E failure mode.
Merge: ff-ready.
