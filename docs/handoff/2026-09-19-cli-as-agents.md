# 2026-09-19 — cli-as-agents: Fatia A (ADR-0160)
Shipped: `agents.cli` (default `pi`); `AddAgentWithCLI`; `Runtime.Start`
refuses a non-Pi agent (`ErrManagedPiOnly`). ADR-0160 supersedes 0159's
"never an agents row". Plan: `docs/plans/cli-as-agents.md`.
Verified: `make close` PASS; store + rpc tests for default Pi, Claude
row, Start refuse. No UI this slice.
visual-review: n/a (no UI)
Not done: Fatia B (create guest as agent + one New → Agent picker);
migrate `managed_clis`. Topic: `open/managed-principals.md`.
Merge: fast-forward ready after this note.

## Next up

- Fatia B: POST /agents `{cli}` + one New → Agent picker
