# 2026-09-24 — exit-model

B0 of the insights study: a terminal CLI's exit records its model (ADR-0194
amendment 2026-09-24). The rest of option B is backlog by the owner's call,
until the catalog holds more answers (`docs/handoff/open/agent-exits.md`).

- `climetrics.SessionCost.Models` counts assistant turns per model;
  `ExitCost.Models` sums them; `dominantModel` fills `model` when the agent
  row has none (tie → first by name). A model set on the agent is kept.
- Verified on a real Codex rollout (`turn_context.model`); the old Codex
  fixture predates that field, so the test adds the line.
